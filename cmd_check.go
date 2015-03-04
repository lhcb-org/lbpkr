package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_check() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_check,
		UsageLine: "check [options]",
		Short:     "check for RPM updates from the yum repository",
		Long: `
check checks for RPM updates from the yum repository.

ex:
 $ lbpkr check
`,
		Flag: *flag.NewFlagSet("lbpkr-check", flag.ExitOnError),
	}
	add_default_options(cmd)
	return cmd
}

func lbpkr_run_cmd_check(cmd *commander.Command, args []string) error {
	var err error

	debug := cmd.Flag.Lookup("v").Value.Get().(bool)
	siteroot := cmd.Flag.Lookup("siteroot").Value.Get().(string)

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
	ctx, err := New(cfg, Debug(debug))
	if err != nil {
		return err
	}
	defer ctx.Close()

	mode := "upgrade"
	switch {
	case ctx.options.Package.Has(UpgradeMode):
		mode = "upgrade"
	case ctx.options.Package.Has(UpdateMode):
		mode = "update"
	}
	ctx.msg.Infof("checking for RPMs %ss\n", mode)
	checkOnly := true
	err = ctx.Update(checkOnly)
	return err
}
