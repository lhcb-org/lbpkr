package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_rpm() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_rpm,
		UsageLine: "rpm [options] -- <rpm-command-args>",
		Short:     "rpm passes through command-args to the RPM binary",
		Long: `
rpm passes through command-args to the RPM binary.

ex:
 $ lbpkr rpm -- --version
`,
		Flag: *flag.NewFlagSet("lbpkr-rpm", flag.ExitOnError),
	}
	add_default_options(cmd)
	return cmd
}

func lbpkr_run_cmd_rpm(cmd *commander.Command, args []string) error {
	var err error

	siteroot := cmd.Flag.Lookup("siteroot").Value.Get().(string)
	debug := cmd.Flag.Lookup("v").Value.Get().(bool)

	switch len(args) {
	case 0:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments. expected at least one argument. got=%d (%v)",
			len(args),
			args,
		)
	}

	cfg := NewConfig(siteroot)
	ctx, err := New(cfg, debug)
	if err != nil {
		return err
	}
	defer ctx.Close()

	err = ctx.Rpm(args...)
	return err
}
