package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_repo_add() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_repo_add,
		UsageLine: "repo-add [options] <repo-name> <repo-url>",
		Short:     "add a repository",
		Long: `repo-add adds a repository source named <repo-name> and located at <repo-url>.

ex:
 $ lbpkr repo-add my-test /some/where
 $ lbpkr repo-add extra http://example.com/rpm
`,
		Flag: *flag.NewFlagSet("lbpkr-repo-add", flag.ExitOnError),
	}
	add_default_options(cmd)
	cmd.Flag.Int("maxdepth", -1, "maximum depth level of dependency graph (-1: all)")
	return cmd
}

func lbpkr_run_cmd_repo_add(cmd *commander.Command, args []string) error {
	var err error

	debug := cmd.Flag.Lookup("v").Value.Get().(bool)
	siteroot := cmd.Flag.Lookup("siteroot").Value.Get().(string)
	//dmax := cmd.Flag.Lookup("maxdepth").Value.Get().(int)

	reponame := ""
	repourl := ""

	switch len(args) {
	case 2:
		reponame = args[0]
		repourl = args[1]
	default:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments. expected n=2. got=%d (%v)",
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

	err = ctx.AddRepository(reponame, repourl)
	return err
}
