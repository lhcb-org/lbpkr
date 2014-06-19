package yum

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	gocfg "github.com/gonuts/config"
	"github.com/gonuts/logger"
)

type Client struct {
	msg         *logger.Logger
	siteroot    string
	etcdir      string
	lbyumcache  string
	yumconf     string
	yumreposdir string
	configured  bool
	repos       map[string]*Repository
	repourls    map[string]string
}

// newClient returns a Client from siteroot and backends.
// manualConfig is just for internal tests
func newClient(siteroot string, backends []string, checkForUpdates, manualConfig bool) (*Client, error) {
	client := &Client{
		msg:         logger.NewLogger("yum", logger.INFO, os.Stdout),
		siteroot:    siteroot,
		etcdir:      filepath.Join(siteroot, "etc"),
		lbyumcache:  filepath.Join(siteroot, "var", "cache", "lbyum"),
		yumconf:     filepath.Join(siteroot, "etc", "yum.conf"),
		yumreposdir: filepath.Join(siteroot, "etc", "yum.repos.d"),
		configured:  false,
		repos:       make(map[string]*Repository),
		repourls:    make(map[string]string),
	}

	if manualConfig {
		return client, nil
	}

	// load the config and set the URLs accordingly
	urls, err := client.loadConfig()
	if err != nil {
		return nil, err
	}

	// At this point we have the repo names and URLs in self.repourls
	// we know connect to them to get the best method to get the appropriate files
	err = client.initRepositories(urls, checkForUpdates, backends)
	if err != nil {
		return nil, err
	}

	return client, err
}

// New returns a new YUM Client, rooted at siteroot.
func New(siteroot string) (*Client, error) {
	checkForUpdates := true
	manualConfig := false
	backends := []string{
		"RepositorySQLiteBackend",
		"RepositoryXMLBackend",
	}
	return newClient(siteroot, backends, checkForUpdates, manualConfig)
}

// Close cleans up after use
func (yum *Client) Close() error {
	var err error
	for name, repo := range yum.repos {
		e := repo.Close()
		if e != nil {
			yum.msg.Errorf("error closing repo [%s]: %v\n", name, e)
			e = err
		} else {
			yum.msg.Debugf("closed repo [%s]\n", name)
		}
	}
	return err
}

// SetLevel sets the verbosity level of Client
func (yum *Client) SetLevel(lvl logger.Level) {
	yum.msg.SetLevel(lvl)
	for _, repo := range yum.repos {
		repo.msg.SetLevel(lvl)
	}
}

// FindLatestMatchingName locates a package by name and returns the latest available version
func (yum *Client) FindLatestMatchingName(name, version, release string) (*Package, error) {
	var err error
	var pkg *Package
	found := make(Packages, 0)
	errors := make([]error, 0, len(yum.repos))

	for _, repo := range yum.repos {
		p, err := repo.FindLatestMatchingName(name, version, release)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		found = append(found, p)
	}

	if len(found) > 0 {
		sort.Sort(found)
		pkg = found[len(found)-1]
		return pkg, err
	}

	if len(errors) == len(yum.repos) && len(errors) > 0 {
		return nil, errors[0]
	}

	return pkg, err
}

// FindLatestMatchingRequire locates a package providing a given functionality.
func (yum *Client) FindLatestMatchingRequire(requirement *Requires) (*Package, error) {
	var err error
	var pkg *Package
	found := make(Packages, 0)
	errors := make([]error, 0, len(yum.repos))

	for _, repo := range yum.repos {
		p, err := repo.FindLatestMatchingRequire(requirement)
		if err != nil {
			errors = append(errors, err)
			yum.msg.Debugf("no match for req=%s.%s-%s (repo=%s)\n",
				requirement.Name(), requirement.Version(), requirement.Release(),
				repo.RepoUrl,
			)
			continue
		}
		found = append(found, p)
	}

	if len(found) > 0 {
		sort.Sort(found)
		pkg = found[len(found)-1]
		return pkg, err
	}

	if len(errors) == len(yum.repos) && len(errors) > 0 {
		return nil, errors[0]
	}

	return pkg, err
}

// ListPackages lists all packages satisfying pattern (a regexp)
func (yum *Client) ListPackages(name, version, release string) ([]*Package, error) {
	var err error
	re_name := regexp.MustCompile(name)
	re_vers := regexp.MustCompile(version)
	re_rel := regexp.MustCompile(release)
	pkgs := make([]*Package, 0)
	for _, repo := range yum.repos {
		for _, pkg := range repo.GetPackages() {
			if re_name.MatchString(pkg.Name()) &&
				re_vers.MatchString(pkg.Version()) &&
				// FIXME: sprintf is ugly
				re_rel.MatchString(fmt.Sprintf("%d", pkg.Release())) {
				pkgs = append(pkgs, pkg)
			}
		}
	}
	return pkgs, err
}

// RequiredPackages returns the list of all required packages for pkg (including pkg itself)
func (yum *Client) RequiredPackages(pkg *Package) ([]*Package, error) {
	pkgs, err := yum.PackageDeps(pkg)
	if err != nil {
		return nil, err
	}
	pkgs = append(pkgs, pkg)
	return pkgs, err
}

// PackageDeps returns all dependencies for the package (excluding the package itself)
func (yum *Client) PackageDeps(pkg *Package) ([]*Package, error) {
	var err error
	processed := make(map[string]*Package)
	deps, err := yum.pkgDeps(pkg, processed)
	if err != nil {
		return nil, err
	}

	pkgs := make([]*Package, 0, len(deps))
	for _, p := range deps {
		pkgs = append(pkgs, p)
	}
	return pkgs, err
}

// pkgDeps returns all dependencies for the package (excluding the package itself)
func (yum *Client) pkgDeps(pkg *Package, processed map[string]*Package) (map[string]*Package, error) {
	var err error
	var lasterr error
	msg := yum.msg

	processed[pkg.RpmName()] = pkg
	required := make(map[string]*Package)

	nreqs := len(pkg.Requires())
	msg.Verbosef(">>> pkg %s.%s-%s (req=%d)\n", pkg.Name(), pkg.Version(), pkg.Release(), nreqs)
	for ireq, req := range pkg.Requires() {
		msg.Verbosef("[%03d/%03d] processing deps for %s.%s-%s\n", ireq, nreqs, req.Name(), req.Version(), req.Release())
		if str_in_slice(req.Name(), g_IGNORED_PACKAGES) {
			msg.Verbosef("[%03d/%03d] processing deps for %s.%s-%s [IGNORE]\n", ireq, nreqs, req.Name(), req.Version(), req.Release())
			continue
		}
		p, err := yum.FindLatestMatchingRequire(req)
		if err != nil {
			lasterr = err
			msg.Debugf("could not find match for %s.%s-%s\n", req.Name(), req.Version(), req.Release())
			continue
		}
		if _, dup := processed[p.RpmName()]; dup {
			msg.Warnf("cyclic dependency in repository with package: %s.%s-%s\n", p.Name(), p.Version(), p.Release())
			continue
		}
		if p == nil {
			msg.Errorf("package %s.%s-%s not found!\n", req.Name(), req.Version(), req.Release())
			lasterr = fmt.Errorf("package %s.%s-%s not found", req.Name(), req.Version(), req.Release())
			continue
			//return nil, fmt.Errorf("package %s.%s-%s not found", req.Name(), req.Version(), req.Release())
		}
		msg.Verbosef("--> adding dep %s.%s-%s\n", p.Name(), p.Version(), p.Release())
		required[p.RpmName()] = p
		sdeps, err := yum.pkgDeps(p, processed)
		if err != nil {
			lasterr = err
			continue
			//return nil, err
		}
		for _, sdep := range sdeps {
			required[sdep.RpmName()] = sdep
		}
	}

	if lasterr != nil {
		err = lasterr
	}
	return required, err
}

// loadConfig looks up the location of the yum repository
func (yum *Client) loadConfig() (map[string]string, error) {
	fis, err := ioutil.ReadDir(yum.yumreposdir)
	if err != nil {
		return nil, err
	}
	pattern := regexp.MustCompile(`(.*)\.repo$`)
	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}
		if !pattern.MatchString(fi.Name()) {
			continue
		}
		fname := filepath.Join(yum.yumreposdir, fi.Name())
		repos, err := yum.parseRepoConfigFile(fname)
		if err != nil {
			return nil, err
		}
		for k, v := range repos {
			yum.repourls[k] = v
		}
	}

	yum.configured = true
	if len(yum.repourls) <= 0 {
		return nil, fmt.Errorf("could not find repository config file in [%s]", yum.yumreposdir)
	}
	return yum.repourls, err
}

// parseRepoConfigFile parses the xyz.repo file and returns a map of reponame/repourl
func (yum *Client) parseRepoConfigFile(fname string) (map[string]string, error) {
	var err error
	repos := make(map[string]string)

	cfg, err := gocfg.ReadDefault(fname)
	if err != nil {
		return nil, err
	}

	for _, section := range cfg.Sections() {
		if !cfg.HasOption(section, "baseurl") {
			continue
		}
		repourl, err := cfg.String(section, "baseurl")
		if err != nil {
			return nil, err
		}
		yum.msg.Debugf("adding repo=%q url=%q from file [%s]\n", section, repourl, fname)
		repos[section] = repourl
	}
	return repos, err
}

func (yum *Client) initRepositories(urls map[string]string, checkForUpdates bool, backends []string) error {
	var err error

	const setupBackend = true

	// setup the repositories
	for repo, repourl := range urls {
		cachedir := filepath.Join(yum.lbyumcache, repo)
		err = os.MkdirAll(cachedir, 0755)
		if err != nil {
			return err
		}
		r, err := NewRepository(
			repo, repourl, cachedir,
			backends, setupBackend, checkForUpdates,
		)
		if err != nil {
			return err
		}
		r.msg = yum.msg
		yum.repos[repo] = r
	}

	yum.repourls = urls
	return err
}

// EOF
