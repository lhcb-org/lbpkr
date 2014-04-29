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

func New(siteroot string) *Client {
	return &Client{
		siteroot:    siteroot,
		etcdir:      filepath.Join(siteroot, "etc"),
		lbyumcache:  filepath.Join(siteroot, "var", "cache", "lbyum"),
		yumconf:     filepath.Join(siteroot, "etc", "yum.conf"),
		yumreposdir: filepath.Join(siteroot, "etc", "yum.repos.d"),
		configured:  false,
		repos:       make(map[string]*Repository),
		repourls:    make(map[string]string),
	}
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
