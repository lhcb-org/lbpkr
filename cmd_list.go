package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_list() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_list,
		UsageLine: "list [options] <name-pattern> [<version-pattern> [<release-pattern>]]",
		Short:     "list all RPM packages satisfying <name-pattern> [<version-pattern> [<release-pattern>]]",
		Long: `
list lists all RPM packages satisfying <name-pattern>.

ex:
 $ lbpkr list GAUDI
 $ lbpkr list GAUDI v23r2
`,
		Flag: *flag.NewFlagSet("lbpkr-list", flag.ExitOnError),
	}
	cmd.Flag.Bool("v", false, "enable verbose mode")
	cmd.Flag.String("type", "lhcb", "config type (lhcb|atlas)")
	return cmd
}

func lbpkr_run_cmd_list(cmd *commander.Command, args []string) error {
	var err error

	cfgtype := cmd.Flag.Lookup("type").Value.Get().(string)
	debug := cmd.Flag.Lookup("v").Value.Get().(bool)

	name := ""
	vers := ""
	release := ""

	switch len(args) {
	case 0:
		name = ""
	case 1:
		name = args[0]
	case 2:
		name = args[0]
		vers = args[1]
	case 3:
		name = args[0]
		vers = args[1]
		release = args[2]
	default:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments. expected n=0|1|2|3. got=%d (%v)",
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

	_, err = ctx.ListPackages(name, vers, release)
	return err
}
