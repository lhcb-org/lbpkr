package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gonuts/logger"
	"github.com/lhcb-org/lbpkr/yum"
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

	dryrun    bool
	extstatus map[string]External
	reqext    []string
	extfix    map[string]FixFct

	ndls int // number of concurrent downloads

	sigch   chan os.Signal
	submux  sync.RWMutex // mutex on subcommands
	subcmds []*exec.Cmd  // list of subcommands launched by lbpkr
	atexit  []func()     // functions to run at-exit
}

func New(cfg Config, dbg bool) (*Context, error) {
	var err error
	siteroot := cfg.Siteroot()
	if siteroot == "" {
		siteroot = "/opt/cern-sw"
	}

	ctx := Context{
		cfg:       cfg,
		msg:       logger.NewLogger("lbpkr", logger.INFO, os.Stdout),
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
		ndls:      runtime.NumCPU(),
		sigch:     make(chan os.Signal),
		subcmds:   make([]*exec.Cmd, 0),
		atexit:    make([]func(), 0),
	}
	if dbg {
		ctx.msg.SetLevel(logger.DEBUG)
	}
	for _, dir := range []string{
		siteroot,
		ctx.dbpath,
		ctx.etcdir,
		ctx.yumreposd,
		ctx.tmpdir,
		ctx.bindir,
		ctx.libdir,
	} {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			ctx.msg.Errorf("could not create directory %q: %v\n", dir, err)
			return nil, err
		}
	}
	os.Setenv("PATH", os.Getenv("PATH")+string(os.PathListSeparator)+ctx.bindir)

	ctx.initSignalHandler()

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

func (ctx *Context) setDry(dry bool) {
	if dry != ctx.dryrun {
		ctx.msg.Debugf("changing dry-run from [%v] to [%v]...\n", ctx.dryrun, dry)
		ctx.dryrun = dry
	}
}

func (ctx *Context) Exit(rc int) {
	err := ctx.Close()
	if err != nil {
		ctx.msg.Errorf("error closing context: %v\n", err)
	}
	for _, fct := range ctx.atexit {
		fct()
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

func (ctx *Context) Client() *yum.Client {
	return ctx.yum
}

func (ctx *Context) SetLevel(lvl logger.Level) {
	ctx.msg.SetLevel(lvl)
	ctx.yum.SetLevel(lvl)
}

// initSignalHandler initalizes the signal handler
func (ctx *Context) initSignalHandler() {
	// initialize signal handler
	go func() {
		ch := make(chan os.Signal)
		signal.Notify(ch, os.Interrupt, os.Kill)
		for {
			select {
			case sig := <-ch:
				// fmt.Fprintf(os.Stderr, "\n>>>>>>>>>\ncaught %#v\n", sig)
				ctx.sigch <- sig
				// fmt.Fprintf(os.Stderr, "subcmds: %d %#v\n", len(ctx.subcmds), ctx.subcmds)
				ctx.submux.RLock()
				for _, cmd := range ctx.subcmds {
					// fmt.Fprintf(os.Stderr, ">>> icmd %d... (%v)\n", icmd, cmd.Args)
					if cmd == nil {
						// fmt.Fprintf(os.Stderr, ">>> cmd nil\n")
						continue
					}
					// fmt.Fprintf(os.Stderr, ">>> sync-ing\n")
					if stdout, ok := cmd.Stdout.(interface {
						Sync() error
					}); ok {
						stdout.Sync()
					}
					if stderr, ok := cmd.Stderr.(interface {
						Sync() error
					}); ok {
						stderr.Sync()
					}
					proc := cmd.Process
					if proc == nil {
						// fmt.Fprintf(os.Stderr, ">>> nil Process\n")
						continue
					}
					pstate := cmd.ProcessState
					if pstate != nil && pstate.Exited() {
						// fmt.Fprintf(os.Stderr, ">>> process already exited\n")
						continue
					}
					// fmt.Fprintf(os.Stderr, ">>> signaling...\n")
					_ = proc.Signal(sig)
					// fmt.Fprintf(os.Stderr, ">>> signaling... [done]\n")
					ps, pserr := proc.Wait()
					if pserr != nil {
						ctx.msg.Errorf("waited and got: %#v (%v)\n", pserr, pserr.Error())
					} else {
						if !ps.Exited() {
							// fmt.Fprintf(os.Stderr, ">>> killing...\n")
							proc.Kill()
							// fmt.Fprintf(os.Stderr, ">>> killing... [done]\n")
						}
					}
					if stdout, ok := cmd.Stdout.(interface {
						Sync() error
					}); ok {
						stdout.Sync()
					}
					if stderr, ok := cmd.Stderr.(interface {
						Sync() error
					}); ok {
						stderr.Sync()
					}
					// fmt.Fprintf(os.Stderr, ">>> re-sync-ing... [done]\n")
				}
				ctx.submux.RUnlock()
				// fmt.Fprintf(os.Stderr, "flushing\n")
				_ = os.Stderr.Sync()
				_ = os.Stdout.Sync()
				// fmt.Fprintf(os.Stderr, "flushed\n")
				ctx.Exit(1)
				return
			}
		}
	}()
}

// initRpmDb initializes the RPM database
func (ctx *Context) initRpmDb() error {
	var err error
	msg := ctx.msg
	msg.Debugf("RPM DB in %q\n", ctx.dbpath)
	err = os.MkdirAll(ctx.dbpath, 0755)
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
		msg.Debugf("Initializing RPM db\n")
		cmd := exec.Command(
			"rpm",
			"--dbpath", ctx.dbpath,
			"--initdb",
		)
		ctx.submux.Lock()
		defer ctx.submux.Unlock()
		ctx.subcmds = append(ctx.subcmds, cmd)
		out, err := cmd.CombinedOutput()
		ctx.subcmds = ctx.subcmds[:len(ctx.subcmds)-1]
		msg.Debugf(string(out))
		if err != nil {
			return fmt.Errorf("error initializing RPM DB: %v", err)
		}
	}
	return err
}

func (ctx *Context) initYum() error {
	var err error
	err = os.MkdirAll(ctx.etcdir, 0755)
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
			ctx.msg.Debugf("%s: Found %q\n", k, ext.cmd)
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
			ctx.msg.Debugf("%s: Found %q\n", k, cmd)
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
func (ctx *Context) checkUpdates(checkOnly bool) error {
	var err error
	pkgs, err := ctx.listInstalledPackages()
	if err != nil {
		return err
	}

	if !checkOnly && ctx.dryrun {
		checkOnly = true
	}

	pkglist := make(map[string]yum.RPMSlice)
	// group by key/version to make sure we only try to update the newest installed
	for _, pkg := range pkgs {
		prov := yum.NewProvides(pkg[0], pkg[1], pkg[2], "", "EQ", nil)
		key := pkg[0] + "-" + pkg[1]
		pkglist[key] = append(pkglist[key], prov)
	}

	nupdates := 0
	for _, rpms := range pkglist {
		sort.Sort(rpms)
		pkg := rpms[len(rpms)-1]
		update, err := ctx.yum.FindLatestProvider(pkg.Name(), "", "")
		if err != nil {
			return err
		}
		if yum.RpmLessThan(pkg, update) {
			nupdates += 1
			if checkOnly {
				ctx.msg.Infof("%s-%s-%s could be updated to %s\n",
					pkg.Name(), pkg.Version(), pkg.Release(),
					update.RpmName(),
				)
			} else {
				ctx.msg.Infof("updating %s-%s-%s to %s\n",
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

	if checkOnly {
		ctx.msg.Infof("packages to update: %d\n", nupdates)
	} else {
		ctx.msg.Infof("packages updated: %d\n", nupdates)
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

	pkg, err := ctx.yum.FindLatestProvider(name, version, release)
	if err != nil {
		return err
	}

	// FIXME: this is because even though lbpkr is statically compiled, it grabs
	//        a dependency against glibc through cgo+SQLite.
	//        ==> generate the RPM with the proper deps ?
	if name == "lbpkr" {
		forceInstall = true
	}
	err = ctx.InstallPackage(pkg, forceInstall, update)
	return err
}

// InstallPackage installs a specific RPM, checking if not already installed
func (ctx *Context) InstallPackage(pkg *yum.Package, forceInstall, update bool) error {
	var err error
	ctx.msg.Infof("installing %s and dependencies\n", pkg.Name())
	var pkgs []*yum.Package
	if pkg.Name() != "lbpkr" {
		pkgs, err = ctx.yum.RequiredPackages(pkg)
		if err != nil {
			ctx.msg.Errorf("required-packages error: %v\n", err)
			return err
		}
	} else {
		pkgs = append(pkgs, pkg)
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
	pkgnames := make([]string, 0, npkgs)
	for _, p := range pkgs {
		pkgnames = append(pkgnames, p.RpmName())
	}
	sort.Strings(pkgnames)
	for i, rpm := range pkgnames {
		ctx.msg.Infof("\t[%03d/%03d] %s\n", i+1, npkgs, rpm)
	}

	if len(pkgs) <= 0 {
		return fmt.Errorf("no RPM to install")
	}

	if ctx.dryrun {
		ctx.msg.Infof("no RPM installed (dry-run)\n")
		return nil
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
func (ctx *Context) ListPackages(name, version, release string) ([]*yum.Package, error) {
	var err error

	pkgs, err := ctx.yum.ListPackages(name, version, release)
	if err != nil {
		return nil, err
	}

	sort.Sort(yum.Packages(pkgs))
	for _, pkg := range pkgs {
		fmt.Printf("%s\n", pkg.ID())
	}

	ctx.msg.Infof("Total matching: %d\n", len(pkgs))
	return pkgs, err
}

// Update checks whether updates are available and installs them if requested
func (ctx *Context) Update(checkOnly bool) error {
	return ctx.checkUpdates(checkOnly)
}

// ListInstalledPackages lists all installed packages satisfying the name/vers/release patterns
func (ctx *Context) ListInstalledPackages(name, version, release string) ([]*yum.Package, error) {
	var err error
	installed, err := ctx.listInstalledPackages()
	if err != nil {
		return nil, err
	}
	filter := func(pkg [3]string) bool { return true }
	if release != "" && version != "" && name != "" {
		re_name := regexp.MustCompile(name)
		re_vers := regexp.MustCompile(version)
		re_release := regexp.MustCompile(release)
		filter = func(pkg [3]string) bool {
			return re_name.MatchString(pkg[0]) &&
				re_vers.MatchString(pkg[1]) &&
				re_release.MatchString(pkg[2])
		}
	} else if version != "" && name != "" {
		re_name := regexp.MustCompile(name)
		re_vers := regexp.MustCompile(version)
		filter = func(pkg [3]string) bool {
			return re_name.MatchString(pkg[0]) &&
				re_vers.MatchString(pkg[1])
		}

	} else if name != "" {
		re_name := regexp.MustCompile(name)
		filter = func(pkg [3]string) bool {
			return re_name.MatchString(pkg[0])
		}
	}

	pkgs := make([]*yum.Package, 0, len(installed))
	for _, pkg := range installed {
		if !filter(pkg) {
			continue
		}
		p, err := ctx.yum.FindLatestProvider(pkg[0], pkg[1], pkg[2])
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, p)
	}
	if len(pkgs) <= 0 {
		fmt.Printf("** No Match found **\n")
		return nil, err
	}

	sort.Sort(yum.Packages(pkgs))
	for _, pkg := range pkgs {
		fmt.Printf("%s\n", pkg.ID())
	}
	return pkgs, err
}

// Provides lists all installed packages providing filename
func (ctx *Context) Provides(filename string) ([]*yum.Package, error) {
	var err error
	re_file, err := regexp.Compile(filename)
	if err != nil {
		return nil, err
	}

	installed, err := ctx.listInstalledPackages()
	if err != nil {
		return nil, err
	}
	rpms := make([]*yum.Package, 0, len(installed))
	for i := range installed {
		ipkg := installed[i]
		pkg, err := ctx.yum.FindLatestProvider(ipkg[0], ipkg[1], ipkg[2])
		if err != nil {
			return nil, err
		}
		rpms = append(rpms, pkg)
	}

	type pair struct {
		pkg  *yum.Package
		file string
	}
	list := make([]pair, 0)
	for _, rpm := range rpms {
		rpmfile := filepath.Join(ctx.tmpdir, rpm.RpmFileName())
		if _, errstat := os.Stat(rpmfile); errstat != nil {
			err = fmt.Errorf("lbpkr: no such file [%s] (%v)", rpmfile, errstat)
			return nil, err
		}
		out, err := ctx.rpm(false, "-qlp", rpmfile)
		if err != nil {
			err = fmt.Errorf("lbpkr: error querying rpm-db: %v", err)
			return nil, err
		}
		scan := bufio.NewScanner(bytes.NewBuffer(out))
		for scan.Scan() {
			file := scan.Text()
			if re_file.MatchString(file) {
				list = append(list, pair{
					pkg:  rpm,
					file: ctx.cfg.RelocateFile(file, ctx.siteroot),
				})
				break
			}
		}
		err = scan.Err()
		if err != nil {
			err = fmt.Errorf("lbpkr: error scaning rpm-output: %v", err)
			return nil, err
		}
	}
	if len(list) <= 0 {
		fmt.Printf("** No Match found **\n")
		return nil, err
	}

	pkgs := make([]string, 0, len(list))
	for _, p := range list {
		pkgs = append(pkgs,
			fmt.Sprintf("%s (%s)", p.pkg.ID(), p.file),
		)
	}

	sort.Strings(pkgs)
	for _, p := range pkgs {
		fmt.Printf("%s\n", p)
	}
	return rpms, err
}

// ListPackageDeps lists all the dependencies of the given RPM package
func (ctx *Context) ListPackageDeps(name, version, release string) ([]*yum.Package, error) {
	var err error
	pkg, err := ctx.yum.FindLatestProvider(name, version, release)
	if err != nil {
		return nil, fmt.Errorf("lbpkr: no such package name=%q version=%q release=%q (%v)", name, version, release, err)
	}

	deps, err := ctx.yum.PackageDeps(pkg)
	if err != nil {
		return nil, fmt.Errorf("lbpkr: could not find dependencies for package=%q (%v)", pkg.ID(), err)
	}

	sort.Sort(yum.Packages(deps))
	for _, pkg := range deps {
		fmt.Printf("%s\n", pkg.ID())
	}
	return deps, err
}

// RemoveRPM removes a (set of) RPM(s) by name
func (ctx *Context) RemoveRPM(rpms [][3]string, force bool) error {
	var err error
	var required []*yum.Requires

	args := []string{"-e"}
	if force {
		args = append(args, "--nodeps")
	}

	if ctx.dryrun {
		args = append(args, "--test")
	}

	for _, id := range rpms {
		pkg, err := ctx.yum.FindLatestProvider(id[0], id[1], id[2])
		if err != nil {
			return err
		}

		required = append(required, pkg.Requires()...)
		args = append(args, pkg.Name())
	}

	_, err = ctx.rpm(true, args...)
	if err != nil {
		//ctx.msg.Errorf("could not remove package:\n%v", string(out))
		return err
	}

	if len(required) > 0 {
		reqs := make([]*yum.Package, 0, len(required))
		for _, req := range required {
			p, err := ctx.yum.FindLatestProvider(req.Name(), req.Version(), req.Release())
			if err != nil {
				continue
			}
			reqs = append(reqs, p)
		}
		sort.Sort(yum.Packages(reqs))
		installed, err := ctx.listInstalledPackages()
		if err != nil {
			return err
		}

		still_req := make(map[string]struct{})
		for _, pp := range installed {
			p, err := ctx.yum.FindLatestProvider(pp[0], pp[1], pp[2])
			if err != nil {
				continue
			}
			for _, r := range p.Requires() {
				pp, err := ctx.yum.FindLatestProvider(r.Name(), r.Version(), r.Release())
				if err != nil {
					continue
				}
				still_req[pp.ID()] = struct{}{}
			}
		}
		remove := make([]string, 0, len(reqs))
		// loop over installed package, if none requires one of the required package, flag it
		for _, req := range reqs {
			id := req.ID()
			if _, dup := still_req[id]; !dup {
				remove = append(remove, req.ID())
			}
		}
		if len(remove) > 0 {
			ctx.msg.Infof("packages no longer required: %v\n", strings.Join(remove, " "))
		}
	}
	return err
}

// Rpm runs the rpm command.
func (ctx *Context) Rpm(args ...string) error {
	_, err := ctx.rpm(true, args...)
	return err
}

// rpm wraps the invocation of the rpm command
func (ctx *Context) rpm(display bool, args ...string) ([]byte, error) {
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
		rpmargs = append(rpmargs, ctx.cfg.RelocateArgs(ctx.siteroot)...)
	}
	rpmargs = append(rpmargs, args...)

	ctx.msg.Debugf("RPM command: rpm %v\n", rpmargs)
	cmd := exec.Command("rpm", rpmargs...)
	ctx.submux.Lock()
	ctx.subcmds = append(ctx.subcmds, cmd)
	ctx.submux.Unlock()
	// cmd.SysProcAttr = &syscall.SysProcAttr{
	// 	Pdeathsig: syscall.SIGINT,
	// }

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	defer stdout.Close()

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	defer stderr.Close()

	var out bytes.Buffer
	if display {
		tee := io.MultiWriter(os.Stdout, &out)

		go io.Copy(tee, stdout)
		go io.Copy(tee, stderr)
	} else {
		go io.Copy(&out, stdout)
		go io.Copy(&out, stderr)
	}
	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	errch := make(chan error)
	go func() {
		errch <- cmd.Wait()
	}()

	select {
	case sig := <-ctx.sigch:
		ctx.msg.Errorf("caught signal [%v]...\n", sig)
		return nil, fmt.Errorf("lbpkr: subcommand killed by [%v]", sig)
	case err = <-errch:
	}

	ctx.submux.Lock()
	ctx.subcmds = ctx.subcmds[:len(ctx.subcmds)-1]
	ctx.submux.Unlock()
	ctx.msg.Debugf(string(out.Bytes()))
	return out.Bytes(), err
}

// filterURLs filters out RPMs already installed
func (ctx *Context) filterURLs(pkgs []*yum.Package) ([]*yum.Package, error) {
	var err error
	filtered := make([]*yum.Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		installed, err2 := ctx.filterURL(pkg)
		err = err2
		if installed {
			ctx.msg.Debugf("already installed: %s\n", pkg.RpmName())
			continue
		}
		filtered = append(filtered, pkg)
	}
	return filtered, err
}

// filterURL returns true if a RPM was already installed
func (ctx *Context) filterURL(pkg *yum.Package) (bool, error) {
	name := pkg.RpmName()
	version := ""
	ctx.msg.Debugf("checking for installation of [%s]...\n", name)
	return ctx.isRpmInstalled(name, version), nil
}

// isRpmInstalled checks whether a given RPM package is already installed
func (ctx *Context) isRpmInstalled(name, version string) bool {
	fullname := name
	if version != "" {
		fullname += "." + version
	}
	out, err := ctx.rpm(false, "-q", fullname)
	if err != nil {
		ctx.msg.Debugf("rpm installed? command failed: %v\n%v\n", err, string(out))
	}
	installed := err == nil
	return installed
}

// listInstalledPackages checks whether a given RPM package is already installed
func (ctx *Context) listInstalledPackages() ([][3]string, error) {
	list := make([][3]string, 0)
	args := []string{"-qa", "--queryformat", "%{NAME} %{VERSION} %{RELEASE}\n"}
	out, err := ctx.rpm(false, args...)
	if err != nil {
		return nil, err
	}

	scan := bufio.NewScanner(bytes.NewBuffer(out))
	for scan.Scan() {
		line := scan.Text()
		pkg := strings.Split(line, " ")
		if len(pkg) != 3 {
			err = fmt.Errorf("lbpkr: invalid line %q", line)
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

	ipkg := 0
	npkgs := len(pkgset)
	todl := 0
	errch := make(chan error, ctx.ndls)
	quit := make(chan struct{})

	var mux sync.RWMutex
	done := 0

	for _, pkg := range pkgset {
		ipkg += 1
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
			mux.Lock()
			done += 1
			mux.Unlock()
			continue
		}

		todl += 1
		go func(ipkg int, pkg *yum.Package) {
			select {
			case errch <- ctx.downloadFile(pkg, dir):
				mux.Lock()
				done += 1
				mux.Unlock()
				ctx.msg.Infof("[%03d/%03d] downloaded %s\n", done, npkgs, pkg.Url())
				return
			case <-quit:
				return
			}
		}(ipkg, pkg)
	}

	for i := 0; i < todl; i++ {
		err = <-errch
		if err != nil {
			quit <- struct{}{}
			ctx.msg.Errorf("error downloading a RPM: %v\n", err)
			return nil, err
		}
	}
	return files, err
}

// downloadFile downloads a given RPM package under dir
func (ctx *Context) downloadFile(pkg *yum.Package, dir string) error {
	var err error
	fname := pkg.RpmFileName()
	fpath := filepath.Join(dir, fname)

	f, err := os.Create(fpath)
	if err != nil {
		return err
	}
	defer f.Close()

	resp, err := http.Get(pkg.Url())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	err = f.Sync()
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	return err
}

// installFiles installs some RPM files given the location of the RPM DB
func (ctx *Context) installFiles(files []string, rpmdir string, forceInstall, update bool) error {
	var err error
	args := []string{"-ivh", "--oldpackage"}
	if update || ctx.cfg.RpmUpdate() {
		args = []string{"-Uvh"}
	}
	if forceInstall {
		args = append(args, "--nodeps")
	}
	if ctx.dryrun {
		args = append(args, "--test")
	}

	for _, fname := range files {
		args = append(args, filepath.Join(rpmdir, fname))
	}

	ctx.msg.Infof("installing [%d] RPMs...\n", len(files))
	out, err := ctx.rpm(true, args...)
	if err != nil {
		ctx.msg.Errorf("rpm install command failed: %v\n%v\n", err, string(out))
		return err
	}
	return err
}

// checkRpmFile checks the integrity of a RPM file
func (ctx *Context) checkRpmFile(fname string) bool {
	args := []string{"-K", fname}
	out, err := ctx.rpm(false, args...)
	if err != nil {
		ctx.msg.Debugf("rpm command failed: %v\n%v\n", err, string(out))
	}
	ok := err == nil
	return ok
}

// EOF
