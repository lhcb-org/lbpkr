package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_deps() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_deps,
		UsageLine: "deps [options] <name-pattern> [<version-pattern> [<release-pattern>]]",
		Short:     "list deps of RPM packages",
		Long: `
deps lists all dependencies of the RPM package satisfying <name-pattern> [<version-pattern> [<release-pattern>]].

ex:
 $ lbpkr deps GAUDI
 $ lbpkr deps GAUDI v23r2
`,
		Flag: *flag.NewFlagSet("lbpkr-deps", flag.ExitOnError),
	}
	add_default_options(cmd)
	cmd.Flag.Int("maxdepth", -1, "maximum depth level of dependency graph (-1: all)")
	return cmd
}

func lbpkr_run_cmd_deps(cmd *commander.Command, args []string) error {
	var err error

	debug := cmd.Flag.Lookup("v").Value.Get().(bool)
	siteroot := cmd.Flag.Lookup("siteroot").Value.Get().(string)
	dmax := cmd.Flag.Lookup("maxdepth").Value.Get().(int)

	name := ""
	vers := ""
	release := ""

	switch len(args) {
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
		return fmt.Errorf("lbpkr: invalid number of arguments. expected n=1|2|3. got=%d (%v)",
			len(args),
			args,
		)
	}

	cfg := NewConfig(siteroot)
	ctx, err := New(cfg, Debug(debug))
	if err != nil {
		return err
	}
	defer ctx.Close()

	pkg, err := ctx.Client().FindLatestProvider(name, vers, release)
	if err != nil {
		return err
	}

	_, err = ctx.ListPackageDeps(pkg.Name(), pkg.Version(), pkg.Release(), dmax)
	if err != nil {
		return err
	}

	return err
}
