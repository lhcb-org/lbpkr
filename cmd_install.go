package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_install() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_install,
		UsageLine: "install [options] <rpm-1> [<rpm-2> [<rpm-3> [...]]]",
		Short:     "install a (list of) RPM(s) from the yum repository",
		Long: `
install installs a (list of) RPMs from the yum repository.

ex:
 $ lbpkr install GAUDI_v25r5
 $ lbpkr install GAUDI_v25r5 AIDA-3fe9f_3.2.1_i686_slc6_gcc48_opt
`,
		Flag: *flag.NewFlagSet("lbpkr-install", flag.ExitOnError),
	}
	add_default_options(cmd)
	cmd.Flag.Bool("force", false, "force RPM installation (by-passing any check)")
	cmd.Flag.Bool("dry-run", false, "dry run. do not actually run the command")
	cmd.Flag.Bool("nodeps", false, "do not install package dependencies")
	cmd.Flag.Bool("justdb", false, "update the database, but do not modify the filesystem")
	return cmd
}

func lbpkr_run_cmd_install(cmd *commander.Command, args []string) error {
	var err error

	siteroot := cmd.Flag.Lookup("siteroot").Value.Get().(string)
	debug := cmd.Flag.Lookup("v").Value.Get().(bool)
	force := cmd.Flag.Lookup("force").Value.Get().(bool)
	dry := cmd.Flag.Lookup("dry-run").Value.Get().(bool)
	nodeps := cmd.Flag.Lookup("nodeps").Value.Get().(bool)
	justdb := cmd.Flag.Lookup("justdb").Value.Get().(bool)

	switch len(args) {
	case 0:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments (got=%d)", len(args))
	}

	cfg := NewConfig(siteroot)
	ctx, err := New(
		cfg,
		Debug(debug),
		EnableForce(force), EnableDryRun(dry), EnableNoDeps(nodeps),
		EnableJustDb(justdb),
	)
	if err != nil {
		return err
	}
	defer ctx.Close()

	ctx.msg.Infof("installing RPMs %v\n", args)

	update := false
	err = ctx.InstallRPMs(args, force, update)
	return err
}
