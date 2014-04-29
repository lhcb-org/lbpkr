package yum

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// RepositoryXMLBackend is a Backend querying YUM XML repositories
type RepositoryXMLBackend struct {
	Name       string
	Packages   map[string]*Package
	Provides   map[string]*Provides
	DBName     string
	Primary    string
	Repository *Repository
}

func NewRepositoryXMLBackend(repo *Repository) *RepositoryXMLBackend {
	const dbname = "primary.xml.gz"
	return &RepositoryXMLBackend{
		Name:       "RepositoryXMLBackend",
		Packages:   make(map[string]*Package),
		Provides:   make(map[string]*Provides),
		DBName:     dbname,
		Primary:    filepath.Join(repo.CacheDir, dbname),
		Repository: repo,
	}
}

// YumDataType returns the ID for the data type as used in the repomd.xml file
func (repo *RepositoryXMLBackend) YumDataType() string {
	return "primary"
}

// Download the DB from server
func (repo *RepositoryXMLBackend) GetLatestDB(url string) error {
	var err error
	out, err := os.Create(repo.Primary)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

// Check whether the DB is there
func (repo *RepositoryXMLBackend) HasDB() bool {
	return path_exists(repo.Primary)
}

// Load loads the DB
func (repo *RepositoryXMLBackend) LoadDB() error {
	var err error

	return err
}

// FindLatestMatchingName locats a package by name, returns the latest available version.
func (repo *RepositoryXMLBackend) FindLatestMatchingName(name, version, release string) (*Package, error) {
	var pkg *Package
	var err error

	return pkg, err
}

// FindLatestMatchingRequire locates a package providing a given functionality.
func (repo *RepositoryXMLBackend) FindLatestMatchingRequire(requirement string) (*Package, error) {
	var pkg *Package
	var err error

	return pkg, err
}

// GetPackages returns all the packages known by a YUM repository
func (repo *RepositoryXMLBackend) GetPackages() []*Package {
	pkgs := make([]*Package, 0, len(repo.Packages))
	for _, pkg := range repo.Packages {
		pkgs = append(pkgs, pkg)
	}
	return pkgs
}
