package yum

import (
	"fmt"
)

// global registry of known backends
var g_backends = make(map[string]func(repo *Repository) (Backend, error))

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
	FindLatestMatchingName(name, version string, release int) (*Package, error)

	// FindLatestMatchingRequire locates a package providing a given functionality.
	FindLatestMatchingRequire(requirement *Requires) (*Package, error)

	// GetPackages returns all the packages known by a YUM repository
	GetPackages() []*Package
}
