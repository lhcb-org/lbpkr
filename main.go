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
		UsageLine: "pkr",
		Short:     "installs software in MYSITEROOT directory.",
		Subcommands: []*commander.Command{
			pkr_make_cmd_install(),
			//pkr_make_cmd_list(),
			//pkr_make_cmd_rpm(),
			pkr_make_cmd_version(),
		},
		Flag: *flag.NewFlagSet("pkr", flag.ExitOnError),
	}
}

func main() {
	err := g_cmd.Flag.Parse(os.Args[1:])
	if err != nil {

	}

	args := g_cmd.Flag.Args()
	err = g_cmd.Dispatch(args)
	handle_err(err)
}
