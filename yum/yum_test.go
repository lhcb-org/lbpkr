package yum

import (
	"testing"
)

func testRepo(t *testing.T) (*Repository, error) {
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
	backend.(*RepositoryXMLBackend).Primary = "testdata/repo.xml"

	repo.Backend = backend
	err = repo.Backend.LoadDB()
	if err != nil {
		return nil, err
	}

	return repo, err
}

func TestPackageMatching(t *testing.T) {
	t.Skip("not ready yet")

	repo, err := testRepo(t)
	if err != nil {
		t.Fatalf("could not create test repo: %v\n", err)
	}
	
	p := NewRequires("TestPackage", "1.0.0", 1, 0, "EQ", "")
	pkg, err := repo.FindLatestMatchingRequire(p)
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
	t.Skip("not ready yet")

	repo, err := testRepo(t)
	if err != nil {
		t.Fatalf("could not create test repo: %v\n", err)
	}

	pkg, err := repo.FindLatestMatchingName("TP2", "1.2.5", "1")
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
