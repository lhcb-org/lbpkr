package yum

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	gocfg "github.com/gonuts/config"
)

type Client struct {
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
func (yum *Client) FindLatestMatchingName(name, version, release string) (string, error) {
	var err error
	if version == "" {
		version = "0.0.1"
	}
	if release == "" {
		release = "1"
	}
	pkg := name + "-" + version + "-" + release
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
