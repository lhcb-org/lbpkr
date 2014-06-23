package main

import (
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_self() *commander.Command {
	cmd := &commander.Command{
		UsageLine: "self [options]",
		Short:     "admin/internal operations for lbpkr",
		Subcommands: []*commander.Command{
			lbpkr_make_cmd_self_bdist(),
			lbpkr_make_cmd_self_bdist_rpm(),
			lbpkr_make_cmd_self_upload_rpm(),
		},
		Flag: *flag.NewFlagSet("lbpkr-self", flag.ExitOnError),
	}
	return cmd
}

// EOF
