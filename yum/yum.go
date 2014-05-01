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
	msg *logger.Logger
	siteroot    string
	etcdir      string
	lbyumcache  string
	yumconf     string
	yumreposdir string
	configured  bool
	repos       map[string]*Repository
	repourls    map[string]string
}

func New(siteroot string) (*Client, error) {
	client := &Client{
		msg: logger.New("yum"),
		siteroot:    siteroot,
		etcdir:      filepath.Join(siteroot, "etc"),
		lbyumcache:  filepath.Join(siteroot, "var", "cache", "lbyum"),
		yumconf:     filepath.Join(siteroot, "etc", "yum.conf"),
		yumreposdir: filepath.Join(siteroot, "etc", "yum.repos.d"),
		configured:  false,
		repos:       make(map[string]*Repository),
		repourls:    make(map[string]string),
	}

	// load the config and set the URLs accordingly
	urls, err := client.loadConfig()
	if err != nil {
		return nil, err
	}

	// At this point we have the repo names and URLs in self.repourls
	// we know connect to them to get the best method to get the appropriate files
	checkForUpdates := true
	backends := []string{
		//"RepositorySQLiteBackend",
		"RepositoryXMLBackend",
	}
	err = client.initRepositories(urls, checkForUpdates, backends)
	if err != nil {
		return nil, err
	}

	return client, err
}

// FindLatestMatchingName locates a package by name and returns the latest available version
func (yum *Client) FindLatestMatchingName(name, version, release string) (*Package, error) {
	var err error
	var pkg *Package
	found := make(Packages, 0)
	
	for _, repo := range yum.repos {
		p, err := repo.FindLatestMatchingName(name, version, release)
		if err != nil {
			return nil, err
		}
		found = append(found, p)
	}

	if len(found) > 0 {
		sort.Sort(found)
		pkg = found[len(found)-1]
	}

	return pkg, err
}

// FindLatestMatchingRequire locates a package providing a given functionality.
func (yum *Client) FindLatestMatchingRequire(requirement *Requires) (*Package, error) {
	var err error
	var pkg *Package
	found := make(Packages, 0)
	
	for _, repo := range yum.repos {
		p, err := repo.FindLatestMatchingRequire(requirement)
		if err != nil {
			return nil, err
		}
		found = append(found, p)
	}

	if len(found) > 0 {
		sort.Sort(found)
		pkg = found[len(found)-1]
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

// PackageDeps returns all dependencies for the package (excluding the package itself)
func (yum *Client) PackageDeps(pkg *Package) ([]*Package, error) {
	var err error
	processed := make(map[*Package]struct{})
	deps, err := yum.pkgDeps(pkg, processed)
	if err != nil {
		return nil, err
	}
	pkgs := make([]*Package, 0, len(deps))
	for p := range deps {
		pkgs = append(pkgs, p)
	}
	return pkgs, err
}

// pkgDeps returns all dependencies for the package (excluding the package itself)
func (yum *Client) pkgDeps(pkg *Package, processed map[*Package]struct{}) (map[*Package]struct{}, error) {
	var err error
	msg := yum.msg

	processed[pkg] = struct{}{}
	required := make(map[*Package]struct{})

	msg.Debugf(">>> pkg %s.%s-%d\n", pkg.Name(), pkg.Version(), pkg.Release())
	for _, req := range pkg.Requires() {
		msg.Debugf("processing deps for %s.%s-%d\n", req.Name(), req.Version(), req.Release())
		if str_in_slice(req.Name(), g_IGNORED_PACKAGES) {
			msg.Debugf("processing deps for %s.%s-%d [IGNORE]\n", req.Name(), req.Version(), req.Release())
			continue
		}
		p, err := yum.FindLatestMatchingRequire(req)
		if err != nil {
			return nil, err
		}
		if _, dup := processed[p]; dup {
			msg.Warnf("cyclic dependency in repository with package: %s.%s-%d\n", p.Name(), p.Version(), p.Release())
			continue
		}
		if p == nil {
			msg.Errorf("package %s.%s-%d not found!\n", req.Name(), req.Version(), req.Release())
			return nil, fmt.Errorf("package %s.%s-%d not found", req.Name(), req.Version(), req.Release())
		}
		msg.Debugf("--> adding dep %s.%s-%d\n", p.Name(), p.Version(), p.Release())
		required[p] = struct{}{}
		sdeps, err := yum.pkgDeps(p, processed)
		if err != nil {
			return nil, err
		}
		for sdep := range sdeps {
			required[sdep] = struct{}{}
		}
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
		repos[section] = repourl
		//fmt.Printf(">>> [%s] repo=%q url=%q\n", fname, section, repourl)
	}
	return repos, err
}

func (yum *Client) initRepositories(urls map[string]string, checkForUpdates bool, backends []string) error {
	var err error

	const setupBackend = true

	// setup the repositories
	for repo, repourl := range urls {
		cachedir := filepath.Join(yum.lbyumcache, repo)
		err = os.MkdirAll(cachedir, 0644)
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
		yum.repos[repo] = r
	}

	yum.repourls = urls
	return err
}

// EOF
