package yum

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// global registry of known backends
var g_backends map[string]func(repo *Repository) (Backend, error)

// NewBackend returns a new backend of type "backend"
func NewBackend(backend string, repo *Repository) (Backend, error) {
	factory, ok := g_backends[backend]
	if !ok {
		return nil, fmt.Errorf("yum: no such backend [%s]", backend)
	}
	return factory(repo)
}

// Backend queries a YUM DB repository
type Backend interface {

	// YumDataType returns the ID for the data type as used in the repomd.xml file
	YumDataType() string

	// Download the DB from server
	GetLatestDB(url string) error

	// Check whether the DB is there
	HasDB() bool

	// Load loads the DB
	LoadDB() error

	// FindLatestMatchingName locats a package by name, returns the latest available version.
	FindLatestMatchingName(name, version, release string) (*Package, error)

	// FindLatestMatchingRequire locates a package providing a given functionality.
	FindLatestMatchingRequire(requirement string) (*Package, error)

	// GetPackages returns all the packages known by a YUM repository
	GetPackages() []*Package
}

// Repository represents a YUM repository with all associated metadata.
type Repository struct {
	Name           string
	RepoUrl        string
	RepoMdUrl      string
	LocalRepoMdXml string
	CacheDir       string
	Backends       []string
	Backend        Backend
}

// NewRepository create a new Repository with name and from url.
func NewRepository(name, url, cachedir string, backends []string, setupBackend, checkForUpdates bool) (*Repository, error) {
	repo := Repository{
		Name:           name,
		RepoUrl:        url,
		RepoMdUrl:      url + "/repodata/repomd.xml",
		LocalRepoMdXml: filepath.Join(cachedir, "repomd.xml"),
		CacheDir:       cachedir,
		Backends:       make([]string, len(backends)),
	}
	copy(repo.Backends, backends)

	err := os.MkdirAll(cachedir, 0644)
	if err != nil {
		return nil, err
	}

	// load appropriate backend if requested
	if setupBackend {
		if checkForUpdates {
			err = repo.setupBackendFromRemote()
			if err != nil {
				return nil, err
			}
		} else {
			err = repo.setupBackendFromLocal()
			if err != nil {
				return nil, err
			}
		}
	}
	return &repo, err
}

// FindLatestMatchingName locats a package by name, returns the latest available version.
func (repo *Repository) FindLatestMatchingName(name, version, release string) (*Package, error) {
	return repo.Backend.FindLatestMatchingName(name, version, release)
}

// FindLatestMatchingRequire locates a package providing a given functionality.
func (repo *Repository) FindLatestMatchingRequire(requirement string) (*Package, error) {
	return repo.Backend.FindLatestMatchingRequire(requirement)
}

// GetPackages returns all the packages known by a YUM repository
func (repo *Repository) GetPackages() []*Package {
	return repo.Backend.GetPackages()
}

// setupBackendFromRemote checks which backend should be used and updates the DB files.
func (repo *Repository) setupBackendFromRemote() error {
	var err error
	return err
}

func (repo *Repository) setupBackendFromLocal() error {
	var err error
	return err
}

// remoteMetadata retrieves the repo metadata file content
func (repo *Repository) remoteMetadata() ([]byte, error) {
	resp, err := http.Get(repo.RepoMdUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf.Bytes(), err
}

// localMetadata retrieves the repo metadata from the repomd file
func (repo *Repository) localMetadata() ([]byte, error) {
	f, err := os.Open(repo.LocalRepoMdXml)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return buf.Bytes(), err
}

// checkRepoMD parses the Repository metadata XML content
func (repo *Repository) checkRepoMD(data []byte) (map[string]RepoMD, error) {
	db := make(map[string]RepoMD)
	var err error

	return db, err
}

type RepoMD struct {
	Checksum  string
	Timestamp time.Time
	Location  string
}

// EOF
