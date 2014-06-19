package yum

import (
	"fmt"
	"strconv"
	"strings"
)

type RPM interface {
	Name() string
	Version() string
	Release() string
	Epoch() string
	Flags() string
	StandardVersion() []string

	RpmName() string
	RpmFileName() string
	//Url() string

	// ID returns the unique identifier of this RPM
	ID() string
}

type rpmBase struct {
	name    string
	version string
	release string
	epoch   string
	flags   string
}

func (rpm *rpmBase) Name() string {
	return rpm.name
}

func (rpm *rpmBase) Version() string {
	return rpm.version
}

func (rpm *rpmBase) Release() string {
	return rpm.release
}

func (rpm *rpmBase) Epoch() string {
	return rpm.epoch
}

func (rpm *rpmBase) Flags() string {
	return rpm.flags
}

func (rpm *rpmBase) StandardVersion() []string {
	return strings.Split(rpm.version, ".")
}

func (rpm *rpmBase) RpmName() string {
	return fmt.Sprintf("%s-%s-%s", rpm.name, rpm.version, rpm.release)
}

func (rpm *rpmBase) RpmFileName() string {
	return fmt.Sprintf("%s-%s-%s.rpm", rpm.name, rpm.version, rpm.release)
}

func (rpm *rpmBase) ID() string {
	str := func(s string) string {
		switch s {
		case "":
			return "*"
		default:
			return s
		}
	}
	return fmt.Sprintf("%s-%s-%s-%s", str(rpm.name), str(rpm.version), str(rpm.release), str(rpm.epoch))
}

func (rpm *rpmBase) ProvideMatches(p RPM) bool {

	if p.Name() != rpm.Name() {
		return false
	}

	if rpm.Version() == "" {
		return true
	}

	switch rpm.Flags() {
	case "EQ", "eq", "==":
		return RpmEqual(p, rpm)
	case "LT", "lt", "<":
		return RpmLessThan(p, rpm)
	case "GT", "gt", ">":
		return !(RpmEqual(p, rpm) || RpmLessThan(p, rpm))
	case "LE", "le", "<=":
		return RpmEqual(p, rpm) || RpmLessThan(p, rpm)
	case "GE", "ge", ">=":
		return !RpmLessThan(p, rpm)
	default:
		panic(fmt.Errorf("invalid Flags %q (package=%v %T)", rpm.Flags(), rpm.Name(), rpm))
	}

	return false
}

func RpmEqual(i, j RPM) bool {
	if i.Name() != j.Name() {
		return false
	}
	if i.Version() != j.Version() {
		return false
	}

	// if i or j misses a releases number, ignore release number
	if i.Release() == "" || j.Release() == "" {
		return true
	}

	return i.Release() == j.Release()
}

func RpmLessThan(i, j RPM) bool {
	if i.Name() != j.Name() {
		return i.Name() < j.Name()
	}

	if i.Version() != j.Version() {
		ii := i.StandardVersion()
		jj := j.StandardVersion()
		n := len(ii)
		if n > len(jj) {
			n = len(jj)
		}
		for k := 0; k < n; k++ {
			iiv, ierr := strconv.Atoi(ii[k])
			jjv, jerr := strconv.Atoi(jj[k])
			if ierr == nil && jerr == nil {
				if iiv != jjv {
					return iiv < jjv
				}
			} else {
				if ii[k] != jj[k] {
					return ii[k] < jj[k]
				}
			}
		}
		return i.Version() < j.Version()
	}

	// if i or j misses a releases number, ignore release number
	if i.Release() == "" || j.Release() == "" {
		return i.Version() < j.Version()
	}
	return i.Release() < j.Release()
}

// Provides represents a functionality provided by a RPM package
type Provides struct {
	rpmBase
	Package *Package // pkg is the package Provides provides for.
}

func NewProvides(name, version, release, epoch, flags string, pkg *Package) *Provides {
	return &Provides{
		rpmBase: rpmBase{
			name:    name,
			version: version,
			release: release,
			epoch:   epoch,
			flags:   flags,
		},
		Package: pkg,
	}
}

// Requires represents a functionality required by a RPM package
type Requires struct {
	rpmBase
	pre string // pre is the prequisite required by a RPM package
}

func NewRequires(name, version, release, epoch, flags string, pre string) *Requires {
	return &Requires{
		rpmBase: rpmBase{
			name:    name,
			version: version,
			release: release,
			epoch:   epoch,
			flags:   flags,
		},
		pre: pre,
	}
}

// Package represents a RPM package in a YUM repository
type Package struct {
	rpmBase

	group      string
	arch       string
	location   string
	requires   []*Requires
	provides   []*Provides
	repository *Repository
}

// NewPackage creates a new RPM package
func NewPackage(name, version, release, epoch string) *Package {
	pkg := Package{
		rpmBase: rpmBase{
			name:    name,
			version: version,
			release: release,
			epoch:   epoch,
		},
		requires: make([]*Requires, 0),
		provides: make([]*Provides, 0),
	}

	return &pkg
}

func (pkg *Package) String() string {
	str := []string{
		fmt.Sprintf(
			"Package: %s-%s-%s\t%s",
			pkg.Name(),
			pkg.Version(),
			pkg.Release(),
			pkg.Group(),
		),
	}

	if len(pkg.provides) > 0 {
		str = append(str, "Provides:")
		for _, p := range pkg.provides {
			str = append(str, fmt.Sprintf("\t%s-%s-%s", p.Name(), p.Version(), p.Release()))
		}
	}

	if len(pkg.requires) > 0 {
		str = append(str, "Requires:")
		for _, p := range pkg.requires {
			str = append(str, fmt.Sprintf("\t%s-%s-%s\t%s", p.Name(), p.Version(), p.Release(), p.Flags()))
		}
	}

	return strings.Join(str, "\n")
}

func (pkg *Package) Group() string {
	return pkg.group
}

func (pkg *Package) Arch() string {
	return pkg.arch
}

func (pkg *Package) Location() string {
	return pkg.location
}

func (pkg *Package) Requires() []*Requires {
	return pkg.requires
}

func (pkg *Package) Provides() []*Provides {
	return pkg.provides
}

func (pkg *Package) Repository() *Repository {
	return pkg.repository
}

func (pkg *Package) Url() string {
	return pkg.repository.RepoUrl + "/" + pkg.location
}

type Packages []*Package

func (p Packages) Len() int {
	return len(p)
}

func (p Packages) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p Packages) Less(i, j int) bool {
	pi := p[i]
	pj := p[j]

	return RpmLessThan(pi, pj)
}

type RPMSlice []RPM

func (p RPMSlice) Len() int {
	return len(p)
}

func (p RPMSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func (p RPMSlice) Less(i, j int) bool {
	pi := p[i]
	pj := p[j]

	return RpmLessThan(pi, pj)
}
