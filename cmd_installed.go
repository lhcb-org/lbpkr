package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_installed() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_installed,
		UsageLine: "installed [options] <name-pattern> [<version-pattern> [<release-pattern>]]",
		Short:     "list all installed RPM packages satisfying <name-pattern> [<version-pattern> [<release-pattern>]]",
		Long: `
installed lists all installed RPM packages satisfying <name-pattern> [<version-pattern> [<release-pattern>]].

ex:
 $ lbpkr installed GAUDI
 $ lbpkr installed GAUDI v23r2
`,
		Flag: *flag.NewFlagSet("lbpkr-installed", flag.ExitOnError),
	}
	cmd.Flag.Bool("v", false, "enable verbose mode")
	cmd.Flag.String("type", "lhcb", "config type (lhcb|atlas)")
	return cmd
}

func lbpkr_run_cmd_installed(cmd *commander.Command, args []string) error {
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

	err = ctx.ListInstalledPackages(name, vers, release)
	return err
}
