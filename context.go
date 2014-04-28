package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gonuts/logger"
)

type External struct {
	cmd string
	err error
}
type FixFct func(*Context) error

type Context struct {
	msg       *logger.Logger
	cfg       Config
	siteroot  string // where to install software, binaries, ...
	repourl   string
	rpmprefix string
	dbpath    string
	etcdir    string
	yumconf   string
	yumreposd string
	yum       *YumClient
	tmpdir    string
	bindir    string
	libdir    string
	initfile  string

	extstatus map[string]External
	reqext    []string
	extfix    map[string]FixFct
}

func New(cfg Config) (*Context, error) {
	var err error
	siteroot := cfg.Siteroot
	ctx := Context{
		cfg:       cfg,
		msg:       logger.NewLogger("pkr", logger.INFO, bufio.NewWriter(os.Stdout)),
		siteroot:  siteroot,
		repourl:   cfg.RepoUrl,
		rpmprefix: cfg.RpmPrefix(),
		dbpath:    filepath.Join(siteroot, "var", "lib", "rpm"),
		etcdir:    filepath.Join(siteroot, "etc"),
		yumconf:   filepath.Join(siteroot, "etc", "yum.conf"),
		yumreposd: filepath.Join(siteroot, "etc", "yum.repos.d"),
		tmpdir:    filepath.Join(siteroot, "tmp"),
		bindir:    filepath.Join(siteroot, "usr", "bin"),
		libdir:    filepath.Join(siteroot, "lib"),
		initfile:  filepath.Join(siteroot, "etc", "repoinit"),
	}
	for _, dir := range []string{
		ctx.tmpdir,
		ctx.bindir,
		ctx.libdir,
	} {
		err = os.MkdirAll(dir, 0644)
		if err != nil {
			ctx.msg.Errorf("could not create directory %q: %v\n", dir, err)
			return nil, err
		}
	}
	os.Setenv("PATH", os.Getenv("PATH")+string(os.PathListSeparator)+ctx.bindir)

	// make sure the db is initialized
	err = ctx.initRpmDb()
	if err != nil {
		return nil, err
	}

	// yum
	err = ctx.initYum()
	if err != nil {
		return nil, err
	}

	ctx.yum = NewClient(ctx.siteroot)
	if ctx.yum == nil {
		return nil, err
	}

	// defining structures and checking if all needed tools are available
	ctx.extstatus = make(map[string]External)
	ctx.reqext = []string{"rpm"}
	ctx.extfix = make(map[string]FixFct)
	err = ctx.checkPreRequisites()
	if err != nil {
		return nil, err
	}

	err = ctx.checkRepository()
	if err != nil {
		return nil, err
	}

	return &ctx, err
}

func (ctx *Context) Exit(rc int) {
	os.Exit(rc)
}

// initRpmDb initializes the RPM database
func (ctx *Context) initRpmDb() error {
	var err error
	msg := ctx.msg
	msg.Infof("RPM DB in %q\n", ctx.dbpath)
	err = os.MkdirAll(ctx.dbpath, 0644)
	if err != nil {
		msg.Errorf(
			"could not create directory %q for RPM DB: %v\n",
			ctx.dbpath,
			err,
		)
		return err
	}

	pkgdir := filepath.Join(ctx.dbpath, "Packages")
	if !path_exists(pkgdir) {
		msg.Infof("Initializing RPM db\n")
		cmd := exec.Command(
			"rpm",
			"--dbpath", ctx.dbpath,
			"--initdb",
		)
		out, err := cmd.CombinedOutput()
		msg.Debugf(string(out))
		if err != nil {
			return fmt.Errorf("error initializing RPM DB: %v", err)
		}
	}
	return err
}

func (ctx *Context) initYum() error {
	var err error
	err = os.MkdirAll(ctx.etcdir, 0644)
	if err != nil {
		return fmt.Errorf("could not create dir %q: %v", ctx.etcdir, err)
	}

	if !path_exists(ctx.yumconf) {
		yum, err := os.Create(ctx.yumconf)
		if err != nil {
			return err
		}
		defer yum.Close()
		err = ctx.writeYumConf(yum)
		if err != nil {
			return err
		}
		err = yum.Sync()
		if err != nil {
			return err
		}
		err = yum.Close()
		if err != nil {
			return err
		}
	}
	err = ctx.cfg.initYum()
	return err
}

// checkPreRequisites makes sure that all external tools required by
// this tool to perform the installation are present
func (ctx *Context) checkPreRequisites() error {
	var err error
	extmissing := false
	missing := make([]string, 0)

	for _, ext := range ctx.reqext {
		cmd, err := exec.LookPath(ext)
		ctx.extstatus[ext] = External{
			cmd: cmd,
			err: err,
		}
	}

	for k, ext := range ctx.extstatus {
		if ext.err == nil {
			ctx.msg.Infof("%s: Found %q\n", k, ext.cmd)
			continue
		}
		ctx.msg.Infof("%s: Missing - trying compensatory measure\n", k)
		fix, ok := ctx.extfix[k]
		if !ok {
			extmissing = true
			missing = append(missing, k)
			continue
		}

		err = fix(ctx)
		if err != nil {
			return err
		}

		cmd, err := exec.LookPath(k)
		ctx.extstatus[k] = External{
			cmd: cmd,
			err: err,
		}
		if err == nil {
			ctx.msg.Infof("%s: Found %q\n", k, cmd)
			continue
		}
		ctx.msg.Infof("%s: Missing\n", k)
		extmissing = true
		missing = append(missing, k)
	}

	if extmissing {
		err = fmt.Errorf("missing external(s): %v", missing)
	}
	return err
}

func (ctx *Context) checkRepository() error {
	var err error
	if !path_exists(ctx.initfile) {
		fini, err := os.Create(ctx.initfile)
		if err != nil {
			return err
		}
		defer fini.Close()
		_, err = fini.WriteString(time.Now().Format(time.RFC3339) + "\n")
		if err != nil {
			return err
		}
		err = fini.Sync()
		if err != nil {
			return err
		}
		return fini.Close()
	}
	err = ctx.checkUpdates()
	return err
}

func (ctx *Context) writeYumConf(w io.Writer) error {
	var err error
	const tmpl = `
[main]
#CONFVERSION 0001
cachedir=/var/cache/yum
debuglevel=2
logfile=/var/log/yum.log
pkgpolicy=newest
distroverpkg=redhat-release
tolerant=1
exactarch=1
obsoletes=1
plugins=1
gpgcheck=0
installroot=%s
reposdir=/etc/yum.repos.d
`
	_, err = fmt.Fprintf(w, tmpl, ctx.siteroot)
	return err
}

// checkUpdates checks whether packages could be updated in the repository
func (ctx *Context) checkUpdates() error {
	var err error
	return err
}

// install performs the whole download/install procedure (eq. yum install)
func (ctx *Context) install(project, version, cmtconfig string) error {
	var err error
	ctx.msg.Infof("Installing %s/%s/%s\n", project, version, cmtconfig)
	return err
}

// EOF
