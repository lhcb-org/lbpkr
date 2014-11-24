package main

import (
	"fmt"
	"regexp"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_install() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_install,
		UsageLine: "install [options] <rpmname> [<version> [<release>]]",
		Short:     "install a RPM from the yum repository",
		Long: `
install installs a RPM from the yum repository.

ex:
 $ lbpkr install LHCb
`,
		Flag: *flag.NewFlagSet("lbpkr-install", flag.ExitOnError),
	}
	add_default_options(cmd)
	cmd.Flag.Bool("force", false, "force RPM installation (by-passing any check)")
	cmd.Flag.Bool("dry-run", false, "dry run. do not actually run the command")
	return cmd
}

func lbpkr_run_cmd_install(cmd *commander.Command, args []string) error {
	var err error

	siteroot := cmd.Flag.Lookup("siteroot").Value.Get().(string)
	debug := cmd.Flag.Lookup("v").Value.Get().(bool)
	force := cmd.Flag.Lookup("force").Value.Get().(bool)
	dry := cmd.Flag.Lookup("dry-run").Value.Get().(bool)

	rpmname := ""
	version := ""
	release := ""
	switch len(args) {
	case 0:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments (got=%d)", len(args))
	case 1:
		rpmname = args[0]
	case 2:
		rpmname = args[0]
		version = args[1]
	case 3:
		rpmname = args[0]
		version = args[1]
		release = args[2]
	default:
		return fmt.Errorf("lbpkr: invalid number of arguments. expected n=1|2|3. got=%d (%v)",
			len(args),
			args,
		)
	}

	re := regexp.MustCompile(`(.*)-([\d\.]+)-(\d)$`).FindAllStringSubmatch(rpmname, -1)
	if len(re) == 1 {
		m := re[0]
		switch len(m) {
		case 2:
			rpmname = m[1]
		case 3:
			rpmname = m[1]
			version = m[2]
		case 4:
			rpmname = m[1]
			version = m[2]
			release = m[3]
		}
	}

	cfg := NewConfig(siteroot)
	ctx, err := New(cfg, debug)
	if err != nil {
		return err
	}
	defer ctx.Close()
	ctx.setDry(dry)

	ctx.msg.Infof("installing RPM %s %s %s\n", rpmname, version, release)

	update := false
	err = ctx.InstallRPM(rpmname, version, release, force, update)
	return err
}
