package main

import (
	"os"
	"path/filepath"
)

var (
	AtlasConfig = &atlasConfig{
		ConfigBase: ConfigBase{
			siteroot: os.Getenv("MYSITEROOT"),
			repourl:  "http://atlas-computing.web.cern.ch/atlas-computing/links/reposDirectory/lcg/slc6/yum/",
			prefix:   "/opt/atlas",
		},
	}
)

type atlasConfig struct {
	ConfigBase
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
