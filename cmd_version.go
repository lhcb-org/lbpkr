package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_version() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_version,
		UsageLine: "version",
		Short:     "print out script version",
		Long: `
version prints out the script version.

ex:
 $ lbpkr version
 20140428
`,
		Flag: *flag.NewFlagSet("lbpkr-version", flag.ExitOnError),
	}
	cmd.Flag.Bool("v", false, "enable verbose mode")
	return cmd
}

func lbpkr_run_cmd_version(cmd *commander.Command, args []string) error {
	var err error
	fmt.Printf("%s\n", Version)
	return err
}
