package main

import (
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func pkr_make_cmd_install() *commander.Command {
	cmd := &commander.Command{
		Run:       pkr_run_cmd_install,
		UsageLine: "install [options] <rpmname> [<version> [<release>]]",
		Short:     "install a RPM from the yum repository",
		Long: `
install installs a RPM from the yum repository.

ex:
 $ pkr install LHCb
`,
		Flag: *flag.NewFlagSet("pkr-install", flag.ExitOnError),
	}
	cmd.Flag.Bool("v", false, "enable verbose mode")
	cmd.Flag.String("type", "lhcb", "config type (lhcb|atlas)")
	return cmd
}

func pkr_run_cmd_install(cmd *commander.Command, args []string) error {
	var err error
	cfg := NewConfig()
	cfg.Type = cmd.Flag.Lookup("type").Value.Get().(string)
	ctx, err := New(cfg)
	if err != nil {
		return err
	}
	ctx.msg.Infof("hello\n")
	return err
}
