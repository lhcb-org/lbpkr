package main

import (
	"github.com/lhcb-org/lbpkr/yum"
)

type Package struct {
	*yum.Package
	Mode Mode
}

type PackagesByDepGraph struct {
	ctx  *Context
	pkgs []Package
}

func (p PackagesByDepGraph) Len() int {
	return len(p.pkgs)
}

func (p PackagesByDepGraph) Swap(i, j int) {
	p.pkgs[i], p.pkgs[j] = p.pkgs[j], p.pkgs[i]
}

func (p PackagesByDepGraph) Less(i, j int) bool {
	pi := p.pkgs[i]
	pj := p.pkgs[j]

	// check whether package 'i' (or one of its deps) needs 'j'
	di, err := p.ctx.yum.RequiredPackages(pi.Package, -1)
	if err != nil {
		panic(err)
	}

	for _, ii := range di {
		if ii.RPMName() == pj.RPMName() {
			// i (or one of its deps) needs package 'j'
			// so correct order is: j <= i
			return false
		}
	}

	if true {
		return true
	}

	pir, err := p.ctx.getNotInstalledPackageDeps(pi)
	if err != nil {
		panic(err)
	}

	pjr, err := p.ctx.getNotInstalledPackageDeps(pj)
	if err != nil {
		panic(err)
	}

	// i deps do not overlap with j.
	// order by un-installed dependencies
	return len(pir) < len(pjr)
}
