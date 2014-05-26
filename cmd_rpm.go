package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func pkr_make_cmd_rpm() *commander.Command {
	cmd := &commander.Command{
		Run:       pkr_run_cmd_rpm,
		UsageLine: "rpm [options] <rpm-command-args>",
		Short:     "rpm passes through command-args to the RPM binary",
		Long: `
rpm passes through command-args to the RPM binary.

ex:
 $ pkr rpm --version
`,
		Flag: *flag.NewFlagSet("pkr-rpm", flag.ExitOnError),
	}
	cmd.Flag.Bool("v", false, "enable verbose mode")
	cmd.Flag.String("type", "lhcb", "config type (lhcb|atlas)")
	return cmd
}

func pkr_run_cmd_rpm(cmd *commander.Command, args []string) error {
	var err error

	cfgtype := cmd.Flag.Lookup("type").Value.Get().(string)
	debug := cmd.Flag.Lookup("v").Value.Get().(bool)

	switch len(args) {
	case 0:
		cmd.Usage()
		return fmt.Errorf("pkr: invalid number of arguments. expected at least one argument. got=%d (%v)",
			len(args),
			args,
		)
	}

	cfg := NewConfig(cfgtype)
	ctx, err := New(cfg, debug)
	if err != nil {
		return err
	}
	defer ctx.Close()

	err = ctx.Rpm(args...)
	return err
}
