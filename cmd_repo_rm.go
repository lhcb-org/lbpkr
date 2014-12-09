package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_repo_rm() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_repo_rm,
		UsageLine: "repo-rm [options] <repo-name>",
		Short:     "remove a repository",
		Long: `repo-rm removes a repository source named <repo-name>.

ex:
 $ lbpkr repo-rm lcg
`,
		Flag: *flag.NewFlagSet("lbpkr-repo-rm", flag.ExitOnError),
	}
	add_default_options(cmd)
	cmd.Flag.Int("maxdepth", -1, "maximum depth level of dependency graph (-1: all)")
	return cmd
}

func lbpkr_run_cmd_repo_rm(cmd *commander.Command, args []string) error {
	var err error

	debug := cmd.Flag.Lookup("v").Value.Get().(bool)
	siteroot := cmd.Flag.Lookup("siteroot").Value.Get().(string)
	//dmax := cmd.Flag.Lookup("maxdepth").Value.Get().(int)

	name := ""

	switch len(args) {
	case 1:
		name = args[0]
	default:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments. expected n=1. got=%d (%v)",
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

	err = ctx.RemoveRepository(name)
	return err
}
