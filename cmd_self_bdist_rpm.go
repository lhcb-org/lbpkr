package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/gonuts/logger"
)

func lbpkr_make_cmd_self_bdist_rpm() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_self_bdist_rpm,
		UsageLine: "bdist-rpm [options]",
		Short:     "create a RPM package of lbpkr",
		Long: `
bdist-rpm creates a RPM package containing lbpkr.

ex:
 $ lbpkr self bdist-rpm
 $ lbpkr self bdist-rpm -name=lbpkr
 $ lbpkr self bdist-rpm -name=lbpkr -version=0.1.20140619
 $ lbpkr self bdist-rpm -name=lbpkr -version=0.1.20140619 -release=1
`,
		Flag: *flag.NewFlagSet("lbpkr-self-bdist-rpm", flag.ExitOnError),
	}
	cmd.Flag.Bool("v", false, "enable verbose mode")
	cmd.Flag.String("type", "lhcb", "config type (lhcb|atlas)")
	cmd.Flag.String("name", "lbpkr", "name of the RPM to generate")
	cmd.Flag.String("version", Version, "version of the RPM to generate")
	cmd.Flag.Int("release", 0, "release number of the RPM to generate")
	return cmd
}

func lbpkr_run_cmd_self_bdist_rpm(cmd *commander.Command, args []string) error {
	var err error

	cfgtype := cmd.Flag.Lookup("type").Value.Get().(string)
	debug := cmd.Flag.Lookup("v").Value.Get().(bool)
	name := cmd.Flag.Lookup("name").Value.Get().(string)
	vers := cmd.Flag.Lookup("version").Value.Get().(string)
	release := cmd.Flag.Lookup("release").Value.Get().(int)

	switch len(args) {
	case 0:
	default:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments. expected n=0. got=%d (%v)",
			len(args),
			args,
		)
	}

	tmpdir, err := ioutil.TempDir("", "lbpkr-self-bdist-rpm-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)
	//fmt.Printf(">>> [%s]\n", tmpdir)

	rpmbuildroot := filepath.Join(tmpdir, "rpmbuild")

	siteroot := ""
	cfg := NewConfig(cfgtype, siteroot)
	msg := logger.New("lbpkr")
	if debug {
		msg.SetLevel(logger.DEBUG)
	}

	rpmbuild, err := exec.LookPath("rpmbuild")
	if err != nil {
		msg.Errorf("could not locate 'rpmbuild': %v\n", err)
		return err
	}

	tarcmd := exec.Command("lbpkr",
		"self", "bdist",
		"-name="+name,
		"-version="+vers,
		fmt.Sprintf("-v=%v", debug),
	)
	tarcmd.Stdout = os.Stdout
	tarcmd.Stderr = os.Stderr
	tarcmd.Stdin = os.Stdin
	err = tarcmd.Run()
	if err != nil {
		msg.Errorf("could not create tarball: %v\n", err)
		return err
	}

	data := struct {
		Url       string
		Prefix    string
		BuildRoot string
		Name      string
		Version   string
		Release   int
	}{
		Url:       "http://github.com/lhcb-org/lbpkr",
		Prefix:    cfg.Siteroot(),
		BuildRoot: tmpdir,
		Name:      name,
		Version:   vers,
		Release:   release,
	}

	rpm_arch := "x86_64"
	switch runtime.GOARCH {
	case "amd64":
		rpm_arch = "x86_64"
	case "386":
		rpm_arch = "i686"
	}

	rpm_fname := fmt.Sprintf("%s-%s-%d.%s.rpm", data.Name, data.Version, data.Release, rpm_arch)
	msg.Infof("creating [%s]...\n", rpm_fname)

	// prepare a tarball with the lbpkr binary.
	dirname := fmt.Sprintf("%s-%s", data.Name, data.Version)
	fname := dirname + ".tar.gz"
	bdist_fname := dirname + "." + rpm_arch + ".tar.gz"

	// create hierarchy of dirs for rpmbuild
	for _, dir := range []string{"RPMS", "SRPMS", "BUILD", "SOURCES", "SPECS", "tmp"} {
		err = os.MkdirAll(filepath.Join(rpmbuildroot, dir), 0755)
		if err != nil {
			return err
		}
	}

	// copy tarball
	err = bincp(filepath.Join(rpmbuildroot, "SOURCES", fname), bdist_fname)
	if err != nil {
		return err
	}

	// create spec-file
	spec_fname := fmt.Sprintf("%s-%s-%d.spec", data.Name, data.Version, data.Release)
	spec, err := os.Create(filepath.Join(
		rpmbuildroot, "SPECS",
		spec_fname,
	))
	if err != nil {
		return err
	}
	defer spec.Close()
	t := template.Must(template.New("bdist-rpm-spec").Parse(rpm_tmpl))
	err = t.Execute(spec, &data)
	if err != nil {
		return err
	}
	spec.Sync()
	spec.Close()

	rpm := exec.Command(rpmbuild, "-ba", filepath.Join("SPECS", spec_fname))
	rpm.Dir = rpmbuildroot
	if debug {
		rpm.Stdout = os.Stdout
		rpm.Stderr = os.Stderr
	}

	err = rpm.Run()
	if err != nil {
		return err
	}

	err = bincp(rpm_fname, filepath.Join(rpmbuildroot, "RPMS", rpm_arch, rpm_fname))
	if err != nil {
		return err
	}

	msg.Infof("creating [%s]... [ok]\n", rpm_fname)
	return err
}

const rpm_tmpl = `
%define        __spec_install_post %{nil}
%define          debug_package %{nil}
%define        __os_install_post %{_dbpath}/brp-compress
%define _topdir   {{.BuildRoot}}/rpmbuild
%define _tmppath  %{_topdir}/tmp

Summary: lbpkr is a tool to install RPMs.
Name: {{.Name}}
Version: {{.Version}}
Release: {{.Release}}
License: BSD
Group: Science
SOURCE0 : %{name}-%{version}.tar.gz
URL: {{.Url}}

BuildRoot: %{_tmppath}/%{name}-%{version}-%{release}-root

%description
%{summary}

%prep
%setup -q

%build
  
%install
rm -rf %{buildroot}
mkdir -p  %{buildroot}/{{.Prefix}}
/bin/cp -r ./* %{buildroot}/{{.Prefix}}

%clean
rm -rf %{buildroot}


%files
%defattr(-,root,root,-)
/{{.Prefix}}/usr/bin/lbpkr
`
