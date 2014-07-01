package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/gonuts/logger"
)

func lbpkr_make_cmd_self_bdist() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_self_bdist,
		UsageLine: "bdist [options]",
		Short:     "create a tarball package of lbpkr",
		Long: `
bdist creates a tarball package containing lbpkr.

ex:
 $ lbpkr self bdist
 $ lbpkr self bdist -name=lbpkr
 $ lbpkr self bdist -name=lbpkr -version=0.1.20140619
`,
		Flag: *flag.NewFlagSet("lbpkr-self-bdist", flag.ExitOnError),
	}
	cmd.Flag.Bool("v", false, "enable verbose mode")
	cmd.Flag.String("name", "lbpkr", "name of the tarball to generate")
	cmd.Flag.String("version", Version, "version of the tarball to generate")
	return cmd
}

func lbpkr_run_cmd_self_bdist(cmd *commander.Command, args []string) error {
	var err error

	debug := cmd.Flag.Lookup("v").Value.Get().(bool)
	name := cmd.Flag.Lookup("name").Value.Get().(string)
	vers := cmd.Flag.Lookup("version").Value.Get().(string)

	switch len(args) {
	case 0:
	default:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments. expected n=0. got=%d (%v)",
			len(args),
			args,
		)
	}

	tmpdir, err := ioutil.TempDir("", "lbpkr-self-bdist-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)
	//fmt.Printf(">>> [%s]\n", tmpdir)

	msg := logger.New("lbpkr")
	if debug {
		msg.SetLevel(logger.DEBUG)
	}

	lbpkr, err := exec.LookPath(os.Args[0])
	if err != nil {
		msg.Errorf("could not locate '%s': %v\n", err, os.Args[0])
		return err
	}

	lbpkr, err = filepath.EvalSymlinks(lbpkr)
	if err != nil {
		msg.Errorf("could not find '%s' executable: %v\n", lbpkr, err)
		return err
	}

	data := struct {
		Name    string
		Version string
		Arch    string
	}{
		Name:    name,
		Version: vers,
		Arch:    "x86_64",
	}

	switch runtime.GOARCH {
	case "amd64":
		data.Arch = "x86_64"
	case "386":
		data.Arch = "i686"
	}

	bdist_fname := fmt.Sprintf("%s-%s.%s.tar.gz", data.Name, data.Version, data.Arch)
	bdist_fname, err = filepath.Abs(bdist_fname)
	if err != nil {
		return err
	}
	msg.Infof("creating [%s]...\n", bdist_fname)

	// prepare a tarball with the lbpkr binary.
	dirname := fmt.Sprintf("%s-%s", data.Name, data.Version)

	//
	top := filepath.Join(tmpdir, dirname)

	// create hierarchy of dirs for bdist
	for _, dir := range []string{
		filepath.Join("usr", "bin"),
	} {
		err = os.MkdirAll(filepath.Join(top, dirname, dir), 0755)
		if err != nil {
			return err
		}
	}

	// install under /bin
	dst_lbpkr := filepath.Join(top, dirname, "usr", "bin", "lbpkr")
	dst, err := os.OpenFile(dst_lbpkr, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	defer func(dst *os.File) error {
		err := dst.Sync()
		if err != nil {
			return err
		}
		err = dst.Close()
		return err
	}(dst)

	src, err := os.Open(lbpkr)
	if err != nil {
		return err
	}
	defer func(src *os.File) error {
		return src.Close()
	}(src)

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	// create tarball
	err = _tar_gz(bdist_fname, top)
	if err != nil {
		return err
	}

	msg.Infof("creating [%s]... [ok]\n", bdist_fname)
	return err
}
