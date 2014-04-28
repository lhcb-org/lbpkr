package main

import "os"

const Version = "20140428"

// Config holds the options and defaults for the installer
type Config struct {
	Siteroot     string // where to install software, binaries, ...
	RepoUrl      string
	Debug        bool
	NoAutoUpdate bool
	RpmUpdate    bool   // install/update switch
	Type         string // atlas|lhcb installation type
}

// NewConfig returns a default configuration value.
func NewConfig() Config {
	return Config{
		Siteroot: os.Getenv("MYSITEROOT"),
		Type:     "lhcb",
	}
}

func (cfg *Config) RpmPrefix() string {
	return ""
}

func (cfg *Config) initYum() error {
	var err error
	return err
}

// EOF
