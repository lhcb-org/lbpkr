package main

import (
	"os"
	"path/filepath"
)

var (
	LHCbConfig = &lhcbConfig{
		ConfigBase: ConfigBase{
			siteroot: os.Getenv("MYSITEROOT"),
			repourl:  "http://test-lbrpm.web.cern.ch/test-lbrpm",
			prefix:   "/opt/lhcb",
		},
	}
)

type lhcbConfig struct {
	ConfigBase
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

	// lhcb stuff
	{
		repo := filepath.Join(repodir, "lhcb.repo")
		f, err := os.Create(repo)
		if err != nil {
			return err
		}
		defer f.Close()

		err = ctx.writeYumRepo(f, map[string]string{
			"name": "lhcbold",
			"url":  repourl + "/rpm",
		})
		if err != nil {
			return err
		}

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

	return err
}
