package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/gonuts/logger"
)

func lbpkr_make_cmd_self_upload_rpm() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_self_upload_rpm,
		UsageLine: "upload-rpm [options] <rpm-file>",
		Short:     "upload a RPM package of lbpkr",
		Long: `
upload-rpm uploads a previously created RPM package containing lbpkr.

ex:
 $ lbpkr self upload-rpm lbpkr-0.1.20140620-0.x86_64.rpm
`,
		Flag: *flag.NewFlagSet("lbpkr-self-upload-rpm", flag.ExitOnError),
	}
	cmd.Flag.Bool("v", false, "enable verbose mode")
	cmd.Flag.String("type", "lhcb", "config type (lhcb|atlas)")
	return cmd
}

func lbpkr_run_cmd_self_upload_rpm(cmd *commander.Command, args []string) error {
	var err error

	cfgtype := cmd.Flag.Lookup("type").Value.Get().(string)
	debug := cmd.Flag.Lookup("v").Value.Get().(bool)

	fname := ""
	switch len(args) {
	case 1:
		fname = args[0]
	default:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments. expected n=1. got=%d (%v)",
			len(args),
			args,
		)
	}

	//cfg := NewConfig(cfgtype)
	msg := logger.New("lbpkr")
	if debug {
		msg.SetLevel(logger.DEBUG)
	}

	msg.Infof("uploading [%s]...\n", fname)

	switch cfgtype {
	case "lhcb":
		rpmdir := "/afs/cern.ch/lhcb/distribution/rpm"
		dst := filepath.Join(rpmdir, "lhcb", fname)
		err = bincp(dst, fname)
		if err != nil {
			msg.Errorf("could not copy [%s] into [%s] (err=%v)\n", fname, dst, err)
			return err
		}

		msg.Debugf("updating metadata...\n")
		updatecmd := filepath.Join(rpmdir, "update.sh")
		regen := exec.Command(updatecmd)
		regen.Dir = rpmdir
		regen.Stdout = os.Stdout
		regen.Stderr = os.Stderr
		regen.Stdin = os.Stdin
		err = regen.Run()
		if err != nil {
			msg.Errorf("could not regenerate metadata: %v\n", err)
			return err
		}
		msg.Debugf("updating metadata... [ok]\n")

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

		err = bincp(filepath.Join(rpmdir, "lbpkr"), lbpkr)
		if err != nil {
			msg.Errorf("could not copy 'lbpkr': %v\n", err)
			return err
		}

	default:
		return fmt.Errorf("lbpkr: config type [%s] not handled", cfgtype)
	}

	msg.Infof("uploading [%s]... [ok]\n", fname)
	return err
}
