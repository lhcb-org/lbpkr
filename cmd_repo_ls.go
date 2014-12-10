package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_repo_ls() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_repo_ls,
		UsageLine: "repo-ls [options]",
		Short:     "list repositories",
		Long: `repo-ls lists repositories.

ex:
 $ lbpkr repo-ls
`,
		Flag: *flag.NewFlagSet("lbpkr-repo-ls", flag.ExitOnError),
	}
	add_default_options(cmd)
	return cmd
}

func lbpkr_run_cmd_repo_ls(cmd *commander.Command, args []string) error {
	var err error

	debug := cmd.Flag.Lookup("v").Value.Get().(bool)
	siteroot := cmd.Flag.Lookup("siteroot").Value.Get().(string)

	switch len(args) {
	case 0:
	default:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments. expected n=0. got=%d (%v)",
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

	err = ctx.ListRepositories()
	return err
}
