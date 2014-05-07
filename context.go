package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gonuts/logger"
	"github.com/lhcb-org/pkr/yum"
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
	dbpath    string
	etcdir    string
	yumconf   string
	yumreposd string
	yum       *yum.Client
	tmpdir    string
	bindir    string
	libdir    string
	initfile  string

	extstatus map[string]External
	reqext    []string
	extfix    map[string]FixFct
}

func New(cfg Config, dbg bool) (*Context, error) {
	var err error
	siteroot := cfg.Siteroot()
	if siteroot == "" {
		siteroot = "/opt/cern-sw"
	}

	ctx := Context{
		cfg:       cfg,
		msg:       logger.NewLogger("pkr", logger.INFO, os.Stdout),
		siteroot:  siteroot,
		repourl:   cfg.RepoUrl(),
		dbpath:    filepath.Join(siteroot, "var", "lib", "rpm"),
		etcdir:    filepath.Join(siteroot, "etc"),
		yumconf:   filepath.Join(siteroot, "etc", "yum.conf"),
		yumreposd: filepath.Join(siteroot, "etc", "yum.repos.d"),
		tmpdir:    filepath.Join(siteroot, "tmp"),
		bindir:    filepath.Join(siteroot, "usr", "bin"),
		libdir:    filepath.Join(siteroot, "lib"),
		initfile:  filepath.Join(siteroot, "etc", "repoinit"),
	}
	if dbg {
		ctx.msg.SetLevel(logger.DEBUG)
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

	ctx.yum, err = yum.New(ctx.siteroot)
	if err != nil {
		return nil, err
	}
	if dbg {
		ctx.yum.SetLevel(logger.DEBUG)
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
	err := ctx.Close()
	if err != nil {
		ctx.msg.Errorf("error closing context: %v\n", err)
	}
	os.Exit(rc)
}

// Close cleans up resources used by the Context
func (ctx *Context) Close() error {
	if ctx == nil {
		return nil
	}

	return ctx.yum.Close()
}

func (ctx *Context) SetLevel(lvl logger.Level) {
	ctx.msg.SetLevel(lvl)
	ctx.yum.SetLevel(lvl)
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
	err = ctx.cfg.InitYum(ctx)
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

func (ctx *Context) writeYumRepo(w io.Writer, data map[string]string) error {
	var err error
	const tmpl = `
[%s]
#REPOVERSION 0001
name=%s
baseurl=%s
enabled=1
`
	_, err = fmt.Fprintf(w, tmpl,
		data["name"],
		data["name"],
		data["url"],
	)
	return err
}

// checkUpdates checks whether packages could be updated in the repository
func (ctx *Context) checkUpdates() error {
	var err error
	pkgs, err := ctx.listInstalledPackages()
	if err != nil {
		return err
	}
	pkglist := make(map[string]yum.RPMSlice)
	// group by key/version to make sure we only try to update the newest installed
	for _, pkg := range pkgs {
		prov := yum.NewProvides(pkg[0], pkg[1], pkg[2], "", "EQ", nil)
		key := pkg[0] + "-" + pkg[1]
		pkglist[key] = append(pkglist[key], prov)
	}

	for _, rpms := range pkglist {
		sort.Sort(rpms)
		pkg := rpms[len(rpms)-1]
		// create a RPM requirement and check whether we have a match
		req := yum.NewRequires(pkg.Name(), pkg.Version(), pkg.Release(), "", "EQ", "")
		update, err := ctx.yum.FindLatestMatchingRequire(req)
		if err != nil {
			return err
		}
		if yum.RpmLessThan(pkg, update) {
			if ctx.cfg.NoAutoUpdate() {
				ctx.msg.Warnf("%s.%s-%s could be updated to %s but update disabled\n",
					pkg.Name(), pkg.Version(), pkg.Release(),
					update.RpmName(),
				)
			} else {
				ctx.msg.Warnf("updating %s.%s-%s to %s\n",
					pkg.Name(), pkg.Version(), pkg.Release(),
					update.RpmName(),
				)
				forceInstall := false
				doUpdate := true
				err = ctx.InstallRPM(update.Name(), update.Version(), update.Release(), forceInstall, doUpdate)
				if err != nil {
					return err
				}
			}
		}
	}
	return err
}

// install performs the whole download/install procedure (eq. yum install)
func (ctx *Context) install(project, version, cmtconfig string) error {
	var err error
	ctx.msg.Infof("Installing %s/%s/%s\n", project, version, cmtconfig)
	return err
}

// InstallRPM installs a RPM by name
func (ctx *Context) InstallRPM(name, version, release string, forceInstall, update bool) error {
	var err error
	pkg, err := ctx.yum.FindLatestMatchingName(name, version, release)
	if err != nil {
		return err
	}
	err = ctx.InstallPackage(pkg, forceInstall, update)
	return err
}

// InstallPackage installs a specific RPM, checking if not already installed
func (ctx *Context) InstallPackage(pkg *yum.Package, forceInstall, update bool) error {
	var err error
	ctx.msg.Infof("installing %s and dependencies\n", pkg.Name())
	pkgs, err := ctx.yum.RequiredPackages(pkg)
	if err != nil {
		ctx.msg.Errorf("required-packages error: %v\n", err)
		return err
	}
	pkgset := make(map[string]*yum.Package)
	for _, p := range pkgs {
		pkgset[p.RpmName()] = p
	}
	pkgs = pkgs[:0]
	for _, p := range pkgset {
		pkgs = append(pkgs, p)
	}

	npkgs := len(pkgs)
	ctx.msg.Infof("found %d RPMs to install:\n", npkgs)
	for i, p := range pkgs {
		ctx.msg.Infof("\t[%03d/%03d] %s\n", i+1, npkgs, p.RpmName())
	}

	if len(pkgs) <= 0 {
		return fmt.Errorf("no RPM to install")
	}

	// filtering urls to only keep the ones not already installed
	filtered, err := ctx.filterURLs(pkgs)
	if err != nil {
		return err
	}

	if len(filtered) <= 0 {
		ctx.msg.Infof("all packages already installed\n")
		return nil
	}

	// download the files
	files, err := ctx.downloadFiles(filtered, ctx.tmpdir)
	if err != nil {
		return err
	}

	// install these files
	err = ctx.installFiles(files, ctx.tmpdir, forceInstall, update)
	return err
}

// ListPackages lists all packages satisfying pattern (a regexp)
func (ctx *Context) ListPackages(name, version, release string) error {
	var err error
	total := 0
	pkgs, err := ctx.yum.ListPackages(name, version, release)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		fmt.Printf("%s\n", pkg.RpmName())
		total += 1
	}
	ctx.msg.Infof("Total matching: %d\n", total)
	return err
}

// rpm wraps the invocation of the rpm command
func (ctx *Context) rpm(args ...string) ([]byte, error) {
	install_mode := false
	query_mode := false
	for _, arg := range args {
		if len(arg) < 2 {
			continue
		}
		if arg[:2] == "-i" || arg[:2] == "-U" {
			install_mode = true
			continue
		}
		if arg[:2] == "-q" {
			query_mode = true
		}
	}

	rpmargs := []string{"--dbpath", ctx.dbpath}
	if !query_mode && install_mode {
		rpmargs = append(rpmargs, "--prefix", ctx.siteroot)
	}
	rpmargs = append(rpmargs, args...)

	ctx.msg.Debugf("RPM command: rpm %v\n", rpmargs)
	cmd := exec.Command("rpm", rpmargs...)
	out, err := cmd.CombinedOutput()
	ctx.msg.Debugf(string(out))
	return out, err
}

// filterURLs filters out RPMs already installed
func (ctx *Context) filterURLs(pkgs []*yum.Package) ([]*yum.Package, error) {
	var err error
	filtered := make([]*yum.Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		name := pkg.RpmName()
		version := ""
		ctx.msg.Debugf("checking for installation of [%s]...\n", name)
		if ctx.isRpmInstalled(name, version) {
			ctx.msg.Debugf("already installed: %s\n", name)
			continue
		}
		filtered = append(filtered, pkg)
	}
	return filtered, err
}

// isRpmInstalled checks whether a given RPM package is already installed
func (ctx *Context) isRpmInstalled(name, version string) bool {
	fullname := name
	if version != "" {
		fullname += "." + version
	}
	out, err := ctx.rpm("-q", fullname)
	if err != nil {
		ctx.msg.Debugf("rpm installed? command failed: %v\n%v\n", err, string(out))
	}
	installed := err == nil
	return installed
}

// listInstalledPackages checks whether a given RPM package is already installed
func (ctx *Context) listInstalledPackages() ([][3]string, error) {
	list := make([][3]string, 0)
	args := []string{"--dbpath", ctx.dbpath, "-qa", "--queryformat", "%{NAME} %{VERSION} %{RELEASE}\n"}
	out, err := ctx.rpm(args...)
	if err != nil {
		return nil, err
	}

	scan := bufio.NewScanner(bytes.NewBuffer(out))
	for scan.Scan() {
		line := scan.Text()
		pkg := strings.Split(line, " ")
		if len(pkg) != 3 {
			err = fmt.Errorf("pkr: invalid line %q", line)
			return nil, err
		}
		for i, p := range pkg {
			pkg[i] = strings.Trim(p, " \n\r\t")
		}
		list = append(list, [3]string{pkg[0], pkg[1], pkg[2]})
	}
	err = scan.Err()
	if err != nil {
		return nil, err
	}
	return list, err
}

// downloadFiles downloads a list of packages
func (ctx *Context) downloadFiles(pkgs []*yum.Package, dir string) ([]string, error) {
	files := make([]string, 0, len(pkgs))
	var err error

	pkgset := make(map[string]*yum.Package)

	for _, pkg := range pkgs {
		fname := pkg.RpmFileName()
		pkgset[fname] = pkg
	}

	for _, pkg := range pkgset {
		fname := pkg.RpmFileName()
		fpath := filepath.Join(dir, fname)
		files = append(files, fname)

		needs_dl := true
		if path_exists(fpath) {
			if ok := ctx.checkRpmFile(fpath); ok {
				needs_dl = false
			}
		}

		if !needs_dl {
			ctx.msg.Debugf("%s already downloaded\n", fname)
			continue
		}

		ctx.msg.Infof("downloading %s to %s\n", pkg.Url(), fpath)
		f, err := os.Create(fpath)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		resp, err := http.Get(pkg.Url())
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		_, err = io.Copy(f, resp.Body)
		if err != nil {
			return nil, err
		}
		err = f.Sync()
		if err != nil {
			return nil, err
		}
		err = f.Close()
		if err != nil {
			return nil, err
		}
	}
	return files, err
}

// installFiles installs some RPM files given the location of the RPM DB
func (ctx *Context) installFiles(files []string, rpmdir string, forceInstall, update bool) error {
	var err error
	args := []string{"-ivh", "--oldpackage"}
	if update || ctx.cfg.RpmUpdate() {
		args = []string{"-Uvh"}
	}
	if forceInstall {
		args = append(args, "--force")
	}
	for _, fname := range files {
		args = append(args, filepath.Join(rpmdir, fname))
	}

	out, err := ctx.rpm(args...)
	if err != nil {
		ctx.msg.Errorf("rpm install command failed: %v\n%v\n", err, string(out))
		return err
	}
	return err
}

// checkRpmFile checks the integrity of a RPM file
func (ctx *Context) checkRpmFile(fname string) bool {
	args := []string{"-K", fname}
	out, err := ctx.rpm(args...)
	if err != nil {
		ctx.msg.Debugf("rpm command failed: %v\n%v\n", err, string(out))
	}
	ok := err == nil
	return ok
}

// EOF
