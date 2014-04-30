package yum

import "path/filepath"

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
func (yum *Client) ListPackages(pattern string) ([]*Package, error) {
	var err error
	pkgs := make([]*Package, 0)

	return pkgs, err
}

// EOF
