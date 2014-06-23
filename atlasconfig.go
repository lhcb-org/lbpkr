package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type atlasConfig struct {
	ConfigBase
}

func newAtlasConfig() *atlasConfig {
	return &atlasConfig{
		ConfigBase: ConfigBase{
			siteroot: os.Getenv("MYSITEROOT"),
			repourl:  "http://atlas-computing.web.cern.ch/atlas-computing/links/reposDirectory/lcg/slc6/yum/",
		},
	}
}

func (cfg *atlasConfig) Name() string {
	return "atlas"
}

func (cfg *atlasConfig) DefaultSiteroot() string {
	return "/opt/atlas"
}

// RelocateArgs returns the arguments to be passed to RPM for the repositories
func (cfg *atlasConfig) RelocateArgs(siteroot string) []string {
	return []string{
		"--relocate", fmt.Sprintf("%s=%s", "/opt/lcg", filepath.Join(siteroot, "lcg", "releases")),
		"--relocate", fmt.Sprintf("%s=%s", "/opt/atlas", siteroot),
		"--badreloc",
	}
}

// RelocateFile returns the relocated file path
func (cfg *atlasConfig) RelocateFile(fname, siteroot string) string {
	fname = strings.Replace(fname, "/opt/lcg", filepath.Join(siteroot, "lcg", "releases"), 1)
	fname = strings.Replace(fname, "/opt/atlas", siteroot, 1)
	return fname
}

func (cfg *atlasConfig) InitYum(ctx *Context) error {
	var err error
	repodir := ctx.yumreposd
	err = os.MkdirAll(repodir, 0644)
	if err != nil {
		return err
	}

	repo := filepath.Join(repodir, "atlas.repo")
	f, err := os.Create(repo)
	if err != nil {
		return err
	}
	defer f.Close()

	data := map[string]string{
		"name": "repo",
		"url":  cfg.RepoUrl(),
	}

	err = ctx.writeYumRepo(f, data)
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

	return err
}
