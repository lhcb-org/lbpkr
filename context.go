package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/gonuts/config"
	"github.com/gonuts/logger"
	"github.com/lhcb-org/lbpkr/yum"
)

type External struct {
	cmd string
	err error
}
type FixFct func(*Context) error

type Mode int

func (m Mode) Has(o Mode) bool {
	return m&o != 0
}

func (m Mode) check() {
	if m.Has(UpdateMode) && m.Has(UpgradeMode) {
		panic("lbpkr: invalid mode (update && upgrade)")
	}
}

func (m Mode) String() string {
	return fmt.Sprintf("Mode{Install=%v, Update=%v, Upgrade=%v, value=%d}",
		m.Has(InstallMode),
		m.Has(UpdateMode),
		m.Has(UpgradeMode),
		int(m),
	)
}

const (
	InstallMode Mode = 1 << iota
	UpdateMode
	UpgradeMode
)

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

	installdb map[[3]string]struct{} // list of installed packages

	// options for the rpm binary
	options struct {
		Force   bool // force rpm installation (by-passing any check)
		DryRun  bool // dry run. do not actually run the command
		NoDeps  bool // do not install package dependencies
		JustDb  bool // update the database, but do not modify the filesystem
		Package Mode // update mode of packages (Install|Update|Upgrade)
	}

	ndls int // number of concurrent downloads

	sigch   chan os.Signal
	submux  sync.RWMutex // mutex on subcommands
	subcmds []*exec.Cmd  // list of subcommands launched by lbpkr
	atexit  []func()     // functions to run at-exit
}

// Debug enables/disables debug mode of Context
func Debug(dbg bool) func(*Context) {
	return func(ctx *Context) {
		if dbg {
			ctx.msg.SetLevel(logger.DEBUG)
		}
	}
}

// EnableForce forces rpm installation (by-passing any check)
func EnableForce(force bool) func(*Context) {
	return func(ctx *Context) {
		ctx.options.Force = force
	}
}

// EnableDryRun sets the dry-run mode. no command is actually run.
func EnableDryRun(dryrun bool) func(*Context) {
	return func(ctx *Context) {
		ctx.options.DryRun = dryrun
	}
}

// EnablePackageMode toggles between various update modes for packages (Install|Update|Upgrade)
func EnablePackageMode(mode Mode) func(*Context) {
	return func(ctx *Context) {
		ctx.options.Package |= mode
		ctx.options.Package.check()
	}
}

func EnableNoDeps(nodeps bool) func(*Context) {
	return func(ctx *Context) {
		ctx.options.NoDeps = nodeps
	}
}

func EnableJustDb(justdb bool) func(*Context) {
	return func(ctx *Context) {
		ctx.options.JustDb = justdb
	}
}

func New(cfg Config, options ...func(*Context)) (*Context, error) {
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
		installdb: nil,
		ndls:      runtime.NumCPU(),
		sigch:     make(chan os.Signal),
		subcmds:   make([]*exec.Cmd, 0),
		atexit:    make([]func(), 0),
	}

	for _, opt := range options {
		opt(&ctx)
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
	if ctx.msg.Level() < logger.INFO {
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
					pserr := killProcess(proc)
					if pserr != nil {
						ctx.msg.Errorf("waited and got: %#v (%v)\n", pserr, pserr.Error())
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
		cmd := newCommand(
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
	if err != nil {
		return err
	}

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

// checkUpdates checks whether packages could be updated/upgraded in the repository
func (ctx *Context) checkUpdates(checkOnly bool) error {
	var err error
	pkgs, err := ctx.listInstalledPackages()
	if err != nil {
		return err
	}

	if !checkOnly && ctx.options.DryRun {
		checkOnly = true
	}

	ctx.options.Force = false

	type Manifest struct {
		Old  yum.RPM
		New  yum.RPM
		Mode Mode
	}

	type cmpFunc func(i, j yum.RPM) bool
	var (
		upgradeFunc cmpFunc = yum.RPMLessThan
		updateFunc  cmpFunc = func(i, j yum.RPM) bool {
			if i.Name() != j.Name() {
				return false
			}

			if i.Version() != j.Version() {
				return false
			}
			return i.Release() < j.Release()
		}
	)

	compare := struct {
		Mode Mode
		Name string
		Func cmpFunc
	}{
		Mode: UpgradeMode,
		Name: "upgrade",
		Func: upgradeFunc,
	}
	switch {
	case ctx.options.Package.Has(UpgradeMode):
		// consider version+release
		compare.Func = upgradeFunc
		compare.Mode = UpgradeMode
		compare.Name = "upgrade"

	case ctx.options.Package.Has(UpdateMode):
		// consider only packages with same version
		compare.Func = updateFunc
		compare.Mode = UpdateMode
		compare.Name = "Update"
	}

	pkglist := make(map[string]yum.RPMSlice)
	// group by key/version to make sure we only try to update the newest installed
	for _, pkg := range pkgs {
		prov := yum.NewProvides(pkg[0], pkg[1], pkg[2], "", "EQ", nil)
		key := pkg[0] + "-" + pkg[1]
		pkglist[key] = append(pkglist[key], prov)
	}

	updateLbpkr := false
	manifest := make([]Manifest, 0, len(pkglist))
	toprocess := make([]Package, 0, len(pkglist))
	for _, rpms := range pkglist {
		sort.Sort(rpms)
		pkg := rpms[len(rpms)-1]
		update, err := ctx.yum.FindLatestProvider(pkg.Name(), "", "")
		if err != nil {
			return err
		}
		if update.Name() == "lbpkr" {
			if checkOnly {
				if yum.RPMLessThan(pkg, update) {
					manifest = append(manifest,
						Manifest{
							Old:  pkg,
							New:  update,
							Mode: UpgradeMode,
						},
					)
				}
				continue
			}

			if yum.RPMLessThan(pkg, update) {
				ctx.options.Force = true
				err = ctx.InstallPackage(Package{Package: update, Mode: UpgradeMode})
				ctx.options.Force = false
				if err != nil || len(pkglist) == 1 {
					return err
				}
				updateLbpkr = true
				continue
			}
		}
		if checkOnly {
			switch {
			case updateFunc(pkg, update):
				manifest = append(manifest,
					Manifest{
						Old:  pkg,
						New:  update,
						Mode: UpdateMode,
					},
				)
			case upgradeFunc(pkg, update):
				manifest = append(manifest,
					Manifest{
						Old:  pkg,
						New:  update,
						Mode: UpgradeMode,
					},
				)
			}
		}
		if compare.Func(pkg, update) {
			toprocess = append(toprocess, Package{update, ctx.options.Package})
		}
	}

	// if only the 'lbpkr' package was updated, then don't consider it as an error
	if updateLbpkr && len(toprocess) <= 0 {
		return err
	}

	if checkOnly {
		upgrade := 0
		update := 0
		w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
		for _, m := range manifest {
			mode := "update"
			if m.Mode == UpgradeMode {
				mode = "upgrade"
				upgrade++
			} else {
				update++
			}
			fmt.Fprintf(w, "%s\t%s-%s\t-> %s-%s\t(%v)\n",
				m.Old.Name(),
				m.Old.Version(), m.Old.Release(),
				m.New.Version(), m.New.Release(),
				mode,
			)
		}
		w.Flush()
		if upgrade > 0 {
			ctx.msg.Infof("packages to upgrade: %d\n", upgrade)
		}
		if update > 0 {
			ctx.msg.Infof("packages to update:  %d\n", update)
		}
		return err
	}

	err = ctx.InstallPackages(toprocess)
	if err != nil {
		return err
	}

	ctx.msg.Infof("packages %sd: %d\n", compare.Name, len(toprocess))
	return err
}

// getNotInstalledPackageDeps returns the list of dependencies for package pkg which have not
// yet been installed
func (ctx *Context) getNotInstalledPackageDeps(pkg Package) ([]Package, error) {
	var err error
	pkgset := make(map[string]Package)

	processed := make(map[string]Package)
	var collect func(pkg Package) ([]Package, error)

	collect = func(pkg Package) ([]Package, error) {
		if _, dup := processed[pkg.ID()]; dup {
			return nil, nil
		}
		rpkgs, err := ctx.yum.PackageDeps(pkg.Package, 1)
		if err != nil {
			return nil, err
		}
		processed[pkg.ID()] = pkg
		var pkgset = make(map[string]Package)

		mode := ctx.options.Package
		// check whether we need to update or just install
		if !mode.Has(UpdateMode) && ctx.isRPMInstalled(pkg.Name(), pkg.Version(), "") {
			mode |= UpdateMode
		}
		if !mode.Has(InstallMode) && !mode.Has(UpdateMode) &&
			!ctx.isRPMInstalled(pkg.Name(), "", "") {
			mode |= InstallMode
		}

		pkg.Mode = mode
		if !ctx.isRPMInstalled(pkg.Name(), pkg.Version(), pkg.Release()) {
			pkgset[pkg.RPMName()] = pkg
		}

		for _, rpkg := range rpkgs {
			if ctx.isRPMInstalled(rpkg.RPMName(), "", "") {
				continue
			}
			opkgs, err := collect(Package{rpkg, mode})
			if err != nil {
				return nil, err
			}
			for _, opkg := range opkgs {
				pkgset[opkg.RPMName()] = opkg
			}
		}
		pkgs := make([]Package, 0, len(pkgset))
		for _, p := range pkgset {
			pkgs = append(pkgs, p)
		}
		return pkgs, err
	}

	pkgs, err := collect(pkg)
	if err != nil {
		return nil, err
	}

	for _, p := range pkgs {
		pkgset[p.RPMName()] = p
	}

	pkgs = pkgs[:0]
	for _, p := range pkgset {
		pkgs = append(pkgs, p)
	}

	return pkgs, nil
}

// InstallRPM installs a RPM by name
func (ctx *Context) InstallRPM(name, version, release string) error {
	rpm := name
	switch {
	case version != "":
		rpm = name + "-" + version
	case version != "" && release != "":
		rpm = name + "-" + version + "-" + release
	}
	rpms := []string{rpm}
	return ctx.InstallRPMs(rpms)
}

// InstallRPMs installs a (list of) RPM(s) by name
func (ctx *Context) InstallRPMs(rpms []string) error {
	var err error

	pkgs := make([]Package, 0, len(rpms))
	for _, rpm := range rpms {
		args := splitRPM(rpm)
		name, version, release := args[0], args[1], args[2]
		pkg, err := ctx.yum.FindLatestProvider(name, version, release)
		if err != nil {
			return err
		}

		// FIXME: this is because even though lbpkr is statically compiled, it grabs
		//        a dependency against glibc through cgo+SQLite.
		//        ==> generate the RPM with the proper deps ?
		if name == "lbpkr" {
			// install/update/upgrade lbpkr first.
			force := ctx.options.Force
			ctx.options.Force = true
			err = ctx.InstallPackage(Package{pkg, InstallMode | UpgradeMode})
			ctx.options.Force = force
			if err != nil || len(rpms) == 1 {
				return err
			}
			continue
		}

		pkgs = append(pkgs, Package{pkg, InstallMode})
	}

	err = ctx.InstallPackages(pkgs)
	return err
}

// InstallProject installs a whole project by name
func (ctx *Context) InstallProject(name, version, release, platforms string) error {
	var err error

	install := make([]Package, 0, 2)
	plist := make([]Package, 0, 2)
	versions := make([]string, 0, 1)

	// find all available project versions
	switch version {
	case "":
		pname := name + `_(?P<ProjectVersion>.*?)_index`
		projs, err := ctx.yum.ListPackages(pname, "", "")
		if err != nil {
			return err
		}
		re := regexp.MustCompile(pname)
		for _, proj := range projs {
			sub := re.FindStringSubmatch(proj.Name())
			if len(sub) > 0 {
				versions = append(versions, sub[1])
			}
		}

	default:
		versions = []string{version}
	}

	// collect available projects+versions
	{
		vers := strings.Join(versions, "|")
		pname := name + "_(" + vers + `)_(?P<ProjectArch>.*?)`
		pkgs, err := ctx.yum.ListPackages(pname, "", "")
		if err != nil {
			return err
		}
		re := regexp.MustCompile(pname)
		for _, pkg := range pkgs {
			sub := re.FindStringSubmatch(pkg.Name())
			if len(sub) > 0 {
				if strings.HasSuffix(pkg.Name(), "_index") {
					continue
				}
				plist = append(plist, Package{pkg, InstallMode})
			}
		}
	}

	if len(plist) <= 0 {
		ctx.msg.Errorf("could not find a project with name=%q version=%q and archs=%q\n",
			name, version, platforms,
		)

		// get list of all projects
		re := regexp.MustCompile(`(?P<ProjectName>.*?)_(?P<ProjectVersion>.*?)_index`)
		set := make(map[string][]string)
		pkgs, err := ctx.yum.ListPackages("", "", "")
		if err != nil {
			return err
		}
		for _, pkg := range pkgs {
			sub := re.FindStringSubmatch(pkg.Name())
			if len(sub) <= 0 {
				continue
			}
			set[sub[1]] = append(set[sub[1]], sub[2])
		}
		pnames := make([]string, 0, len(set))
		for k := range set {
			pnames = append(pnames, k)
		}
		sort.Strings(pnames)
		ctx.msg.Infof("Available projects: %d\n", len(pnames))
		w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
		for _, p := range pnames {
			fmt.Fprintf(w, "%s\t%s\n", p, strings.Join(set[p], ", "))
		}
		w.Flush()

		return fmt.Errorf("could not find a project with name=%q version=%q and archs=%q",
			name, version, platforms,
		)
	}

	if platforms == "" {
		// if no CMTCONFIG defined, we'll default to "ALL"
		platforms = os.Getenv("CMTCONFIG")
	}

	archs := make([]string, 0, 2)
	switch platforms {
	case "", "ALL", "all":
		archs = nil
	default:
		for _, v := range strings.Split(platforms, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				archs = append(archs, v)
			}
		}
	}

	{
		archset := make(map[string]struct{})
		arch := `.*`
		if len(archs) > 0 {
			arch = "(" + strings.Join(archs, "|") + ")"
		}
		pname := regexp.MustCompile(name + `_(?P<ProjectVersion>.*?)_(?P<ProjectArch>` + arch + ")")
		for _, v := range plist {
			sub := pname.FindStringSubmatch(v.Name())
			if len(sub) <= 0 {
				continue
			}
			install = append(install, v)
			archset[sub[2]] = struct{}{}
		}

		if len(archset) <= 0 {
			return fmt.Errorf("could not find a project with name=%q version=%q and platforms=%v",
				name, version, platforms,
			)
		}

		archs = archs[:0]
		for k := range archset {
			archs = append(archs, k)
		}
	}

	ctx.msg.Infof("installing project name=%q version=%q for archs=%v\n",
		name, version, archs,
	)

	if len(install) <= 0 {
		ctx.msg.Errorf("found NO project matching this description\n")
		return fmt.Errorf("could not find a project with name=%q version=%q and archs=%v",
			name, version, archs,
		)
	}

	ctx.msg.Infof("found %d project(s) matching this description:\n", len(install))
	pnames := make([]string, 0, len(install))
	for _, pkg := range install {
		pnames = append(pnames, pkg.Name())
	}
	sort.Strings(pnames)
	for _, pkg := range pnames {
		fmt.Printf("%s\n", pkg)
	}

	err = ctx.InstallPackages(install)
	return err
}

// InstallPackage installs a specific RPM, checking if not already installed
func (ctx *Context) InstallPackage(pkg Package) error {
	pkgs := []Package{pkg}
	return ctx.InstallPackages(pkgs)
}

// InstallPackages installs a list of specific RPMs, checking if not already installed
func (ctx *Context) InstallPackages(packages []Package) error {
	var err error
	pkgs := make([]Package, 0, len(packages))
	pkgset := make(map[string]Package)

	for _, pkg := range packages {
		dodeps := " and dependencies"
		if ctx.options.NoDeps {
			dodeps = " (w/o dependencies)"
		}
		ctx.msg.Infof("installing %s%s\n", pkg.RPMName(), dodeps)
		if pkg.Name() == "lbpkr" {
			pkgs = append(pkgs, pkg)
			continue
		}
		if ctx.options.NoDeps {
			// user requested to NOT install dependencies
			pkgs = append(pkgs, pkg)
			continue
		}
		var opkgs []Package
		opkgs, err = ctx.getNotInstalledPackageDeps(pkg)
		if err != nil {
			ctx.msg.Errorf("required-packages error: %v\n", err)
			return err
		}
		pkgs = append(pkgs, opkgs...)
	}

	for _, p := range pkgs {
		pkgset[p.RPMName()] = p
	}
	pkgs = pkgs[:0]
	for _, p := range pkgset {
		pkgs = append(pkgs, p)
	}

	npkgs := len(pkgs)
	ctx.msg.Infof("found %d RPMs to install:\n", npkgs)
	pkgnames := make([]string, 0, npkgs)
	for _, p := range pkgs {
		pkgnames = append(pkgnames, p.RPMName())
	}
	sort.Strings(pkgnames)
	for i, rpm := range pkgnames {
		ctx.msg.Infof("\t[%03d/%03d] %s\n", i+1, npkgs, rpm)
	}

	if len(pkgs) <= 0 {
		return fmt.Errorf("no RPM to install")
	}

	if ctx.options.DryRun {
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

	// download the packages
	err = ctx.downloadPackages(filtered, ctx.tmpdir)
	if err != nil {
		return err
	}

	// install these packages
	err = ctx.installPackages(filtered, ctx.tmpdir)
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
		rpmfile := filepath.Join(ctx.tmpdir, rpm.RPMFileName())
		if _, errstat := os.Stat(rpmfile); errstat != nil {
			// try to re-download the package
			errdl := ctx.downloadPackages([]Package{{Package: rpm}}, ctx.tmpdir)
			if errdl != nil {
				err = fmt.Errorf("lbpkr: no such file [%s] (%v)", rpmfile, errstat)
				return nil, err
			}
		}
		out, err := ctx.rpm(false, "-qlp", rpmfile)
		if err != nil {
			err = fmt.Errorf("lbpkr: error querying rpm-db: %v", err)
			return nil, err
		}
		scan := bufio.NewScanner(bytes.NewBuffer(out))
		for scan.Scan() {
			file := ctx.cfg.RelocateFile(scan.Text())
			if re_file.MatchString(file) {
				list = append(list, pair{
					pkg:  rpm,
					file: ctx.cfg.RelocateFile(file),
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
func (ctx *Context) ListPackageDeps(name, version, release string, depthmax int) ([]*yum.Package, error) {
	var err error
	pkg, err := ctx.yum.FindLatestProvider(name, version, release)
	if err != nil {
		return nil, fmt.Errorf("lbpkr: no such package name=%q version=%q release=%q (%v)", name, version, release, err)
	}

	deps, depsErr := ctx.yum.PackageDeps(pkg, depthmax)
	// do not handle the depsErr error just yet.
	// printout the deps we've got so far.
	if depsErr != nil {
		err = depsErr
	}

	sort.Sort(yum.Packages(deps))
	for _, pkg := range deps {
		fmt.Printf("%s\n", pkg.ID())
	}

	if depsErr != nil {
		return nil, fmt.Errorf("lbpkr: could not find dependencies for package=%q (%v)", pkg.ID(), depsErr)
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

	if ctx.options.DryRun {
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

	// FIXME(sbinet)
	// when in query-mode, rpm --dbpath ... will print the filenames without
	// relocating them.
	// we should fix that up as that may be utterly confusing.
	// e.g. it would print:
	//  /opt/lcg/blas/20110419-e1974/x86_64-slc6-gcc48-opt/lib/libBLAS.a
	// instead of:
	//  $MYSITEROOT/lcg/releases/blas/20110419-e1974/x86_64-slc6-gcc48-opt/lib/libBLAS.a

	rpmargs := []string{"--dbpath", ctx.dbpath}
	if !query_mode && install_mode {
		rpmargs = append(rpmargs, ctx.cfg.RelocateArgs()...)
	}
	rpmargs = append(rpmargs, args...)

	ctx.msg.Debugf("RPM command: rpm %v\n", rpmargs)
	cmd := newCommand("rpm", rpmargs...)
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
func (ctx *Context) filterURLs(pkgs []Package) ([]Package, error) {
	var err error
	filtered := make([]Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		installed, err2 := ctx.filterURL(pkg.Package)
		err = err2
		if installed {
			ctx.msg.Debugf("already installed: %s\n", pkg.RPMName())
			continue
		}
		filtered = append(filtered, pkg)
	}
	return filtered, err
}

// filterURL returns true if a RPM was already installed
func (ctx *Context) filterURL(pkg yum.RPM) (bool, error) {
	name := pkg.Name()
	version := pkg.Version()
	release := pkg.Release()
	ctx.msg.Debugf("checking for installation of [%s]...\n", name)
	return ctx.isRPMInstalled(name, version, release), nil
}

// isRPMInstalled checks whether a given RPM package is already installed
func (ctx *Context) isRPMInstalled(name, version, release string) bool {
	if ctx.installdb == nil {
		ctx.initInstalledPackages()
	}
	if ctx.installdb != nil {
		_, ok := ctx.installdb[[3]string{name, version, release}]
		return ok
	}

	fullname := name
	if version != "" {
		fullname += "-" + version
		if release != "" {
			fullname += "-" + release
		}
	}
	out, err := ctx.rpm(false, "-q", fullname)
	if err != nil {
		ctx.msg.Debugf("rpm installed? command failed: %v\n%v\n", err, string(out))
	}
	installed := err == nil
	return installed
}

// initInstalledPackages populates the cache of installed packages
func (ctx *Context) initInstalledPackages() {
	installed, err := ctx.listInstalledPackages()
	if err != nil {
		ctx.msg.Errorf("lbpkr: %v\n", err)
		return
	}

	installdb := make(map[[3]string]struct{}, len(installed)*3)
	for _, v := range installed {
		installdb[[3]string{v[0], "", ""}] = struct{}{}
		installdb[[3]string{v[0], v[1], ""}] = struct{}{}
		installdb[[3]string{v[0], v[1], v[2]}] = struct{}{}
	}

	ctx.installdb = installdb
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

// downloadPackages downloads a list of packages
func (ctx *Context) downloadPackages(pkgs []Package, dir string) error {
	var err error

	pkgset := make(map[string]Package)

	for _, pkg := range pkgs {
		fname := pkg.RPMFileName()
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
		fname := pkg.RPMFileName()
		fpath := filepath.Join(dir, fname)

		needsDl := true
		if path_exists(fpath) {
			if ok := ctx.checkRpmFile(fpath); ok {
				needsDl = false
			}
		}

		if !needsDl {
			ctx.msg.Debugf("%s already downloaded\n", fname)
			mux.Lock()
			done += 1
			mux.Unlock()
			continue
		}

		todl += 1
		go func(ipkg int, pkg Package) {
			select {
			case errch <- ctx.downloadPackage(pkg, dir):
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
			close(quit)
			ctx.msg.Errorf("error downloading a RPM: %v\n", err)
			return err
		}
	}
	return err
}

// downloadPackage downloads a given RPM package under dir
func (ctx *Context) downloadPackage(pkg Package, dir string) error {
	var err error
	fname := pkg.RPMFileName()
	fpath := filepath.Join(dir, fname)

	f, err := os.Create(fpath)
	if err != nil {
		return err
	}
	defer f.Close()

	r, err := getRemoteData(pkg.Url())
	if err != nil {
		return err
	}
	defer r.Close()

	_, err = io.Copy(f, r)
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

// installPackages installs some RPM files given the location of the RPM DB
func (ctx *Context) installPackages(pkgs []Package, rpmdir string) error {

	// split the install between packages to be installed (anew) and packages to be updated
	// assume that we can safely first update the to-be-updated packages
	// and then install the other ones.
	//
	// FIXME(sbinet): we could/should instead order the input packages via their dependency
	// and install/update each one in turn.
	// but finding the correct order is very time consuming (at least with a naive algorithm)
	// finding a way to extract that order from `rpm` would be great.

	installCmd := []string{"-ivh", "--oldpackage"}
	updateCmd := []string{"-Uvh"}
	add := func(v string) {
		installCmd = append(installCmd, v)
		updateCmd = append(updateCmd, v)
	}

	if ctx.options.Force || ctx.options.NoDeps {
		add("--nodeps")
	}
	if ctx.options.JustDb {
		add("--justdb")
	}
	if ctx.options.DryRun {
		add("--test")
	}

	install := []string{}
	update := []string{}
	for _, pkg := range pkgs {
		fname := filepath.Join(rpmdir, pkg.RPMFileName())
		switch {
		case pkg.Mode.Has(UpdateMode) || pkg.Mode.Has(UpgradeMode) || ctx.cfg.RpmUpdate():
			update = append(update, fname)
		default:
			install = append(install, fname)
		}
	}

	if len(update) > 0 {
		ctx.msg.Infof("updating [%d] RPMs...\n", len(update))
		updateCmd = append(updateCmd, update...)
		out, err := ctx.rpm(true, updateCmd...)
		if err != nil {
			ctx.msg.Errorf("rpm install command failed: %v\n%v\n", err, string(out))
			return err
		}
	}

	if len(install) > 0 {
		ctx.msg.Infof("installing [%d] RPMs...\n", len(install))
		installCmd = append(installCmd, install...)
		out, err := ctx.rpm(true, installCmd...)
		if err != nil {
			ctx.msg.Errorf("rpm install command failed: %v\n%v\n", err, string(out))
			return err
		}
	}

	return nil
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

// AddRepository adds a repository named name and located at repo.
func (ctx *Context) AddRepository(name, repo string) error {
	repo, err := sanitizePathOrURL(repo)
	if err != nil {
		return err
	}

	data := map[string]string{
		"name": name,
		"url":  repo,
	}

	fname := filepath.Join(ctx.yumreposd, name+".repo")
	if path_exists(fname) {
		return fmt.Errorf("lbpkr: repo %q already exists", name)
	}

	defer func() {
		if err != nil {
			os.Remove(fname)
		}
	}()

	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer f.Close()

	err = ctx.writeYumRepo(f, data)
	if err != nil {
		return err
	}

	err = f.Close()
	if err != nil {
		return err
	}

	// make sure the new repo is correct
	ctx, err = New(ctx.cfg, Debug(false))
	if err != nil {
		return err
	}

	return err
}

// RemoveRepository removes the repository named name.
func (ctx *Context) RemoveRepository(name string) error {
	var err error
	fname := filepath.Join(ctx.yumreposd, name+".repo")
	if !path_exists(fname) {
		return fmt.Errorf("lbpkr: no such repo %q", name)
	}

	cfg, err := config.ReadDefault(fname)
	if err != nil {
		return err
	}
	v, err := cfg.String(name, "name")
	if err != nil {
		return err
	}
	if v != name {
		return fmt.Errorf("lbpkr: invalid repo name (got=%q. want=%q)", v, name)
	}
	err = os.Remove(fname)
	if err != nil {
		return err
	}
	return err
}

// ListRepositories lists all repositories.
func (ctx *Context) ListRepositories() error {
	var err error
	reposdir, err := os.Open(ctx.yumreposd)
	if err != nil {
		return err
	}
	defer reposdir.Close()

	dirs, err := reposdir.Readdir(-1)
	if err != nil {
		return err
	}

	for _, fi := range dirs {
		fname := filepath.Join(ctx.yumreposd, fi.Name())
		cfg, err := config.ReadDefault(fname)
		if err != nil {
			return err
		}
		for _, section := range cfg.Sections() {
			if section == config.DEFAULT_SECTION {
				continue
			}
			name, err := cfg.String(section, "name")
			if err != nil {
				return err
			}
			baseurl, err := cfg.String(section, "baseurl")
			if err != nil {
				return err
			}
			isEnabled, err := cfg.Bool(section, "enabled")
			if err != nil {
				return err
			}
			enabled := "disabled"
			if isEnabled {
				enabled = "enabled"
			}
			fmt.Printf("%s: %q (%s)\n", name, baseurl, enabled)
		}
	}
	return err
}

// EOF
