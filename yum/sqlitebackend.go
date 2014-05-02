package yum

import (
	"github.com/gonuts/logger"
)

// RepositorySQLiteBackend is Backend querying YUM SQLite repositories
type RepositorySQLiteBackend struct {
	Name       string
	Packages   map[string][]*Package
	Provides   map[string][]*Provides
	DBName     string
	Primary    string
	Repository *Repository
	msg        *logger.Logger
}

// EOF
