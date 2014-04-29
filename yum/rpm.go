package yum

import (
	"strings"
)

type RPM struct {
	Name    string
	Version string
	Release string
	Epoch   string
	Flags   string
}

func (rpm *RPM) StandardVersion() []string {
	return strings.Split(rpm.Version, ".")
}

type Package struct {
	Name    string
	Version string
	Release string
	Epoch   string
	Flags   string
}
