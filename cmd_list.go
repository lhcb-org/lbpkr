package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func pkr_make_cmd_list() *commander.Command {
	cmd := &commander.Command{
		Run:       pkr_run_cmd_list,
		UsageLine: "list [options] <name-pattern>",
		Short:     "list all RPM packages satisfying <name-pattern>",
		Long: `
list lists all RPM packages satisfying <name-pattern>.

ex:
 $ pkr list LHCb
`,
		Flag: *flag.NewFlagSet("pkr-list", flag.ExitOnError),
	}
	cmd.Flag.Bool("v", false, "enable verbose mode")
	cmd.Flag.String("type", "lhcb", "config type (lhcb|atlas)")
	return cmd
}

func pkr_run_cmd_list(cmd *commander.Command, args []string) error {
	var err error

	cfgtype := cmd.Flag.Lookup("type").Value.Get().(string)
	debug := cmd.Flag.Lookup("v").Value.Get().(bool)

	cfg := NewConfig(cfgtype)
	ctx, err := New(cfg, debug)
	if err != nil {
		return err
	}

	ctx.msg.Infof("hello: %v\n", cfg.Prefix())

	rpmname := ""
	switch len(args) {
	case 0:
		cmd.Usage()
		return fmt.Errorf("pkr: invalid number of arguments (got=%d)", len(args))
	case 1:
		rpmname = args[0]
	default:
		return fmt.Errorf("pkr: invalid number of arguments. expected n=1. got=%d (%v)",
			len(args),
			args,
		)
	}

	err = ctx.ListPackages(rpmname)
	return err
}
