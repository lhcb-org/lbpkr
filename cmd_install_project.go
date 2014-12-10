package main

import (
	"fmt"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_install_project() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_install_project,
		UsageLine: "install-project [options] <project-name> [<version> [<release>]]",
		Short:     "install-project a whole project from the yum repository",
		Long: `
install-project installs a whole project from the yum repository.

ex:
 $ lbpkr install-project GAUDI
 $ lbpkr install-project GAUDI v42
 $ lbpkr install-project -platforms=all GAUDI v42
`,
		Flag: *flag.NewFlagSet("lbpkr-install-project", flag.ExitOnError),
	}
	add_default_options(cmd)
	cmd.Flag.Bool("force", false, "force RPM installation (by-passing any check)")
	cmd.Flag.Bool("dry-run", false, "dry run. do not actually run the command")
	cmd.Flag.String("platforms", "", "comma-separated list of (regex) platforms to install")
	cmd.Flag.Bool("nodeps", false, "do not verify package dependencies")
	cmd.Flag.Bool("justdb", false, "update the database, but do not modify the filesystem")
	return cmd
}

func lbpkr_run_cmd_install_project(cmd *commander.Command, args []string) error {
	var err error

	siteroot := cmd.Flag.Lookup("siteroot").Value.Get().(string)
	debug := cmd.Flag.Lookup("v").Value.Get().(bool)
	force := cmd.Flag.Lookup("force").Value.Get().(bool)
	dry := cmd.Flag.Lookup("dry-run").Value.Get().(bool)
	archs := cmd.Flag.Lookup("platforms").Value.Get().(string)
	nodeps := cmd.Flag.Lookup("nodeps").Value.Get().(bool)
	justdb := cmd.Flag.Lookup("justdb").Value.Get().(bool)

	projname := ""
	version := ""
	release := ""
	switch len(args) {
	case 0:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments (got=%d)", len(args))
	case 1:
		projname = args[0]
	case 2:
		projname = args[0]
		version = args[1]
	case 3:
		projname = args[0]
		version = args[1]
		release = args[2]
	default:
		return fmt.Errorf("lbpkr: invalid number of arguments. expected n=1|2|3. got=%d (%v)",
			len(args),
			args,
		)
	}

	cfg := NewConfig(siteroot)
	ctx, err := New(
		cfg,
		Debug(debug),
		EnableForce(force), EnableDryRun(dry), EnableNoDeps(nodeps),
		EnableJustDb(justdb),
	)
	if err != nil {
		return err
	}
	defer ctx.Close()

	ctx.msg.Infof("installing project %s %s %s\n", projname, version, release)

	update := false
	err = ctx.InstallProject(projname, version, release, archs, force, update)
	return err
}
