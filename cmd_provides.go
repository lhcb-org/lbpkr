package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_provides() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_provides,
		UsageLine: "provides [options] <file>",
		Short:     "list all installed RPM packages providing the given file",
		Long: `
provides lists all installed RPM packages providing the given file.

ex:
 $ lbpkr provides gaudirun.py
 GAUDI_v25r1_x86_64_slc6_gcc48_opt-1.0.0-1 (/opt/cern-sw/lhcb/GAUDI/GAUDI_v25r1/InstallArea/x86_64-slc6-gcc48-opt/scripts/gaudirun.py)
`,
		Flag: *flag.NewFlagSet("lbpkr-provides", flag.ExitOnError),
	}
	cmd.Flag.Bool("v", false, "enable verbose mode")
	cmd.Flag.String("type", "lhcb", "config type (lhcb|atlas)")
	return cmd
}

func lbpkr_run_cmd_provides(cmd *commander.Command, args []string) error {
	var err error

	cfgtype := cmd.Flag.Lookup("type").Value.Get().(string)
	debug := cmd.Flag.Lookup("v").Value.Get().(bool)

	filename := ""

	switch len(args) {
	case 1:
		filename = args[0]
	default:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments. expected n=1. got=%d (%v)",
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

	err = ctx.Provides(filename)
	return err
}
