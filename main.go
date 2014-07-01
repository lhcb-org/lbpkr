package main

import (
	"os"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

var g_cmd *commander.Command
var g_ctx *Context

func init() {
	g_cmd = &commander.Command{
		UsageLine: "lbpkr",
		Short:     "installs software in MYSITEROOT directory.",
		Subcommands: []*commander.Command{
			lbpkr_make_cmd_check(),
			lbpkr_make_cmd_deps(),
			lbpkr_make_cmd_dep_graph(),
			lbpkr_make_cmd_install(),
			lbpkr_make_cmd_installed(),
			lbpkr_make_cmd_list(),
			lbpkr_make_cmd_provides(),
			lbpkr_make_cmd_remove(),
			lbpkr_make_cmd_rpm(),
			lbpkr_make_cmd_self(),
			lbpkr_make_cmd_update(),
			lbpkr_make_cmd_version(),
		},
		Flag: *flag.NewFlagSet("lbpkr", flag.ContinueOnError),
	}
}

func main() {
	var args []string

	err := g_cmd.Flag.Parse(os.Args[1:])
	if err != nil || err == flag.ErrHelp {
		args = []string{"help"}
	} else {
		args = g_cmd.Flag.Args()
	}

	err = g_cmd.Dispatch(args)
	handle_err(err)
}
