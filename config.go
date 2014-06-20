package main

import "os"

type Config interface {
	DefaultSiteroot() string
	Siteroot() string
	RepoUrl() string
	Name() string
	Debug() bool
	RpmUpdate() bool

	// RelocateArgs returns the arguments to be passed to RPM for the repositories
	RelocateArgs(siteroot string) []string

	// RelocateFile returns the relocated file path
	RelocateFile(fname, siteroot string) string

	InitYum(*Context) error
}

// ConfigBase holds the options and defaults for the installer
type ConfigBase struct {
	siteroot  string // where to install software, binaries, ...
	repourl   string
	debug     bool
	rpmupdate bool // install/update switch
}

func (cfg *ConfigBase) Siteroot() string {
	return cfg.siteroot
}

func (cfg *ConfigBase) RepoUrl() string {
	return cfg.repourl
}

func (cfg *ConfigBase) Debug() bool {
	return cfg.debug
}

func (cfg *ConfigBase) RpmUpdate() bool {
	return cfg.rpmupdate
}

// NewConfig returns a default configuration value.
func NewConfig(cfgtype string) Config {
	switch cfgtype {
	case "atlas":
		AtlasConfig.siteroot = os.Getenv("MYSITEROOT")
		return AtlasConfig
	case "lhcb":
		LHCbConfig.siteroot = os.Getenv("MYSITEROOT")
		return LHCbConfig
	default:
		panic("lbpkr: unknown config [" + cfgtype + "]")
	}
	panic("unreachable")
}

// EOF
