package yum

import (
	"path/filepath"
	"testing"

	"github.com/gonuts/logger"
)

func getTestClient(t *testing.T) (*Client, error) {
	const siteroot = "testdata/mysiteroot"
	client := &Client{
		msg: logger.New("yum"),
		siteroot: siteroot,
		etcdir:      filepath.Join(siteroot, "etc"),
		lbyumcache:  filepath.Join(siteroot, "var", "cache", "lbyum"),
		yumconf:     filepath.Join(siteroot, "etc", "yum.conf"),
		yumreposdir: filepath.Join(siteroot, "etc", "yum.repos.d"),
		configured:  false,
		repos:       make(map[string]*Repository),
		repourls:    make(map[string]string),
	}
	setupBackend := false
	checkForUpdates := true
	repo, err := NewRepository("testrepo", "http://dummy-url.org", "testdata/cachedir.tmp",
		[]string{"RepositoryXMLBackend"},
		setupBackend,
		checkForUpdates,
	)
	if err != nil {
		return nil, err
	}

	backend, err := NewRepositoryXMLBackend(repo)
	if err != nil {
		return nil, err
	}
	backend.Primary = "testdata/repo.xml"

	repo.Backend = backend
	err = repo.Backend.LoadDB()
	if err != nil {
		return nil, err
	}

	client.repos[repo.Name] = repo
	client.configured = true
	return client, err
}

func TestPackageMatching(t *testing.T) {

	yum, err := getTestClient(t)
	if err != nil {
		t.Fatalf("could not create test repo: %v\n", err)
	}

	p := NewRequires("TestPackage", "1.0.0", 1, 0, "EQ", "")
	pkg, err := yum.FindLatestMatchingRequire(p)
	if err != nil {
		t.Fatalf("could not find match: %v\n", err)
	}

	if pkg == nil {
		t.Fatalf("could not find match: nil package\n")
	}

	if pkg.Version() != "1.0.0" {
		t.Fatalf("expected version=%q. got=%q\n", "1.0.0", pkg.Version())
	}
}

func TestPackageByNameWithRelease(t *testing.T) {

	yum, err := getTestClient(t)
	if err != nil {
		t.Fatalf("could not create test repo: %v\n", err)
	}

	pkg, err := yum.FindLatestMatchingName("TP2", "1.2.5", "1")
	if err != nil {
		t.Fatalf("could not find latest matching name: %v\n", err)
	}

	if pkg == nil {
		t.Fatalf("could not find latest matching name: nil package\n")
	}

	if pkg.Version() != "1.2.5" {
		t.Fatalf("expected version=%q. got=%q\n", "1.2.5", pkg.Version())
	}

	if pkg.Release() != 1 {
		t.Fatalf("expected release=1. got=%d\n", 1, pkg.Release())
	}
}

func TestPackageByNameWithoutRelease(t *testing.T) {

	yum, err := getTestClient(t)
	if err != nil {
		t.Fatalf("could not create test repo: %v\n", err)
	}

	pkg, err := yum.FindLatestMatchingName("TP2", "1.2.5", "")
	if err != nil {
		t.Fatalf("could not find latest matching name: %v\n", err)
	}

	if pkg == nil {
		t.Fatalf("could not find latest matching name: nil package\n")
	}

	if pkg.Version() != "1.2.5" {
		t.Fatalf("expected version=%q. got=%q\n", "1.2.5", pkg.Version())
	}

	if pkg.Release() != 2 {
		t.Fatalf("expected release=1. got=%d\n", 2, pkg.Release())
	}
}

func TestPackageByNameWithoutVersion(t *testing.T) {

	yum, err := getTestClient(t)
	if err != nil {
		t.Fatalf("could not create test repo: %v\n", err)
	}

	pkg, err := yum.FindLatestMatchingName("TP2", "", "")
	if err != nil {
		t.Fatalf("could not find latest matching name: %v\n", err)
	}

	if pkg == nil {
		t.Fatalf("could not find latest matching name: nil package\n")
	}

	if pkg.Version() != "1.2.5" {
		t.Fatalf("expected version=%q. got=%q\n", "1.2.5", pkg.Version())
	}

	if pkg.Release() != 2 {
		t.Fatalf("expected release=1. got=%d\n", 2, pkg.Release())
	}
}
