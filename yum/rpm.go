package yum

import (
	"fmt"
	"strings"
)

type RPM interface {
	Name() string
	Version() string
	Release() int
	Epoch() int
	Flags() string
	StandardVersion() []string

	RpmName() string
	RpmFileName() string
	//Url() string
}

type rpmBase struct {
	name    string
	version string
	release int
	epoch   int
	flags   string
}

func (rpm *rpmBase) Name() string {
	return rpm.name
}

func (rpm *rpmBase) Version() string {
	return rpm.version
}

func (rpm *rpmBase) Release() int {
	return rpm.release
}

func (rpm *rpmBase) Epoch() int {
	return rpm.epoch
}

func (rpm *rpmBase) Flags() string {
	return rpm.flags
}

func (rpm *rpmBase) StandardVersion() []string {
	return strings.Split(rpm.version, ".")
}

func (rpm *rpmBase) RpmName() string {
	return fmt.Sprintf("%s-%s-%d", rpm.name, rpm.version, rpm.release)
}

func (rpm *rpmBase) RpmFileName() string {
	return fmt.Sprintf("%s-%s-%d.rpm", rpm.name, rpm.version, rpm.release)
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
		return RpmEqual(rpm, p)
	case "LT", "lt", "<":
		return RpmLessThan(rpm, p)
	case "GT", "gt", ">":
		return !(RpmEqual(rpm, p) || RpmLessThan(rpm, p))
	case "LE", "le", "<=":
		return RpmEqual(rpm, p) || RpmLessThan(rpm, p)
	case "GE", "ge", ">=":
		return !RpmLessThan(rpm, p)
	default:
		panic(fmt.Errorf("invalid Flags %q (package=%v)", rpm.Flags(), rpm.Name()))
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
	if i.Release() == 0 || j.Release() == 0 {
		return true
	}

	return i.Release() == j.Release()
}

func RpmLessThan(i, j RPM) bool {
	if i.Name() != j.Name() {
		return i.Name() < j.Name()
	}

	if i.Version() != j.Version() {
		//FIXME: more thorough ?
		return i.Version() < j.Version()
	}

	// if i or j misses a releases number, ignore release number
	if i.Release() == 0 || j.Release() == 0 {
		return i.Version() < j.Version()
	}
	return i.Release() < j.Release()
}

// Provides represents a functionality provided by a RPM package
type Provides struct {
	rpmBase
	Package *Package // pkg is the package Provides provides for.
}

func NewProvides(name, version string, release, epoch int, flags string, pkg *Package) *Provides {
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

func NewRequires(name, version string, release, epoch int, flags string, pre string) *Requires {
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
	requires   []RPM
	provides   []RPM
	repository *Repository
}

// NewPackage creates a new RPM package
func NewPackage(name, version string, release, epoch int) *Package {
	pkg := Package{
		rpmBase: rpmBase{
			name:    name,
			version: version,
			release: release,
			epoch:   epoch,
		},
		requires: make([]RPM, 0),
		provides: make([]RPM, 0),
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

func (pkg *Package) Requires() []RPM {
	return pkg.requires
}

func (pkg *Package) Provides() []RPM {
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

func (p Packages) Less(i,j int) bool {
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

func (p RPMSlice) Less(i,j int) bool {
	pi := p[i]
	pj := p[j]

	return RpmLessThan(pi, pj)
}
