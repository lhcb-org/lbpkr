package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type lhcbConfig struct {
	ConfigBase
}

func newLHCbConfig(siteroot string) *lhcbConfig {

	cfg := &lhcbConfig{
		ConfigBase: ConfigBase{
			siteroot: siteroot,
			repourl:  "http://cern.ch/lhcbproject/dist/rpm",
		},
	}
	if siteroot == "" {
		cfg.siteroot = cfg.DefaultSiteroot()
	}
	return cfg
}

func (cfg *lhcbConfig) Name() string {
	return "lhcb"
}

func (cfg *lhcbConfig) DefaultSiteroot() string {
	return "/opt/LHCbSoft"
}

// RelocateArgs returns the arguments to be passed to RPM for the repositories
func (cfg *lhcbConfig) RelocateArgs(siteroot string) []string {
	return []string{
		"--relocate", fmt.Sprintf("%s=%s", "/opt/lcg/external", filepath.Join(siteroot, "lcg", "external")),
		"--relocate", fmt.Sprintf("%s=%s", "/opt/lcg", filepath.Join(siteroot, "lcg", "releases")),
		"--relocate", fmt.Sprintf("%s=%s", "/opt/LHCbSoft", siteroot),
		"--badreloc",
	}
}

// RelocateFile returns the relocated file path
func (cfg *lhcbConfig) RelocateFile(fname, siteroot string) string {
	fname = strings.Replace(fname, "/opt/lcg", filepath.Join(siteroot, "lcg", "releases"), 1)
	fname = strings.Replace(fname, "/opt/LHCbSoft", siteroot, 1)
	return fname
}

func (cfg *lhcbConfig) InitYum(ctx *Context) error {
	var err error
	repourl := cfg.RepoUrl()
	if repourl[len(repourl)-1] == '/' {
		repourl = repourl[:len(repourl)-1]
	}
	repodir := ctx.yumreposd
	err = os.MkdirAll(repodir, 0644)
	if err != nil {
		return err
	}

	// lcg stuff
	{
		repo := filepath.Join(repodir, "lcg.repo")
		f, err := os.Create(repo)
		if err != nil {
			return err
		}
		defer f.Close()

		err = ctx.writeYumRepo(f, map[string]string{
			"name": "lcg",
			"url":  "http://cern.ch/service-spi/external/rpms/lcg",
		})
		if err != nil {
			return err
		}

		err = f.Sync()
		if err != nil {
			return err
		}

		err = f.Close()
		if err != nil {
			return err
		}
	}

	// lhcb stuff
	{
		repo := filepath.Join(repodir, "lhcb.repo")
		f, err := os.Create(repo)
		if err != nil {
			return err
		}
		defer f.Close()

		err = ctx.writeYumRepo(f, map[string]string{
			"name": "lhcb",
			"url":  repourl + "/lhcb",
		})
		if err != nil {
			return err
		}

		err = f.Sync()
		if err != nil {
			return err
		}

		err = f.Close()
		if err != nil {
			return err
		}
	}


	// lhcb ext stuff
	{
		repo := filepath.Join(repodir, "lhcbext.repo")
		f, err := os.Create(repo)
		if err != nil {
			return err
		}
		defer f.Close()

		err = ctx.writeYumRepo(f, map[string]string{
			"name": "lhcbext",
			"url":  repourl + "/lcg",
		})
		if err != nil {
			return err
		}

		err = f.Sync()
		if err != nil {
			return err
		}

		err = f.Close()
		if err != nil {
			return err
		}
	}

	return err
}
