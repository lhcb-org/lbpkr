package main

const Version = "20140428"

type Config interface {
	Siteroot() string
	RepoUrl() string
	Prefix() string
	Debug() bool
	RpmUpdate() bool

	InitYum(*Context) error
}

// ConfigBase holds the options and defaults for the installer
type ConfigBase struct {
	siteroot  string // where to install software, binaries, ...
	repourl   string
	prefix    string // prefix path for RPMs
	debug     bool
	rpmupdate bool // install/update switch
}

func (cfg *ConfigBase) Siteroot() string {
	return cfg.siteroot
}

func (cfg *ConfigBase) RepoUrl() string {
	return cfg.repourl
}

func (cfg *ConfigBase) Prefix() string {
	return cfg.prefix
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
		panic("pkr: unknown config [" + cfgtype + "]")
	}
	panic("unreachable")
}

// EOF
