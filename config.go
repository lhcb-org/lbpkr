package main

type Config interface {
	Siteroot() string
	RepoUrl() string
	Name() string
	Debug() bool
	RpmUpdate() bool

	// RelocateArgs returns the arguments to be passed to RPM for the repositories
	RelocateArgs(siteroot string) []string

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
		return AtlasConfig
	case "lhcb":
		return LHCbConfig
	default:
		panic("lbpkr: unknown config [" + cfgtype + "]")
	}
	panic("unreachable")
}

// EOF
