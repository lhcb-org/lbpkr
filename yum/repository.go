package yum

// Backend queries a YUM DB repository
type Backend interface {
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
	Backends       []Backend
	Backend        Backend
}
