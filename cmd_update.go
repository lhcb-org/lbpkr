package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_update() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_update,
		UsageLine: "update [options]",
		Short:     "update RPMs from the yum repository",
		Long: `
update updates RPMs from the yum repository.

ex:
 $ lbpkr update
`,
		Flag: *flag.NewFlagSet("lbpkr-update", flag.ExitOnError),
	}
	add_default_options(cmd)
	cmd.Flag.Bool("dry-run", false, "dry run. do not actually run the command")
	return cmd
}

func lbpkr_run_cmd_update(cmd *commander.Command, args []string) error {
	var err error

	siteroot := cmd.Flag.Lookup("siteroot").Value.Get().(string)
	debug := cmd.Flag.Lookup("v").Value.Get().(bool)
	dry := cmd.Flag.Lookup("dry-run").Value.Get().(bool)

	switch len(args) {
	case 0:
		// no-op
	default:
		return fmt.Errorf("lbpkr: invalid number of arguments. expected none. got=%d (%v)",
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
	ctx.setDry(dry)

	ctx.msg.Infof("updating RPMs\n")
	checkOnly := false
	err = ctx.Update(checkOnly)
	return err
}
