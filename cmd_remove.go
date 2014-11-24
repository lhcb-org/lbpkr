package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_remove() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_remove,
		UsageLine: "remove [options] <rpmname> [<rpmname> [<rpmname>]]",
		Short:     "remove a RPM from the yum repository",
		Long: `
remove removes a RPM from the yum repository.

ex:
 $ lbpkr remove gcc_4.8.1_x86_64_slc6-1.0.0-1
 $ lbpkr remove gcc_4.8.1_x86_64_slc6-1.0.0-1 xrootd-3a806_3.2.7_x86_64_slc6_gcc48_opt-1.0.0-4
`,
		Flag: *flag.NewFlagSet("lbpkr-remove", flag.ExitOnError),
	}
	add_default_options(cmd)
	cmd.Flag.Bool("force", false, "force removal of RPM")
	cmd.Flag.Bool("dry-run", false, "dry run. do not actually run the command")
	return cmd
}

func lbpkr_run_cmd_remove(cmd *commander.Command, args []string) error {
	var err error

	siteroot := cmd.Flag.Lookup("siteroot").Value.Get().(string)
	debug := cmd.Flag.Lookup("v").Value.Get().(bool)
	force := cmd.Flag.Lookup("force").Value.Get().(bool)
	dry := cmd.Flag.Lookup("dry-run").Value.Get().(bool)

	rpms := make([][3]string, 0)
	switch len(args) {
	case 0:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments (got=%d)", len(args))
	default:
		re := regexp.MustCompile(`(.*)-([\d\.]+)-(\d)$`)
		for _, name := range args {
			rpmname := name
			version := ""
			release := ""

			match := re.FindAllStringSubmatch(rpmname, -1)
			if len(match) == 1 {
				m := match[0]
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
			rpms = append(rpms, [3]string{rpmname, version, release})
		}
	}

	cfg := NewConfig(siteroot)
	ctx, err := New(cfg, debug)
	if err != nil {
		return err
	}
	defer ctx.Close()
	ctx.setDry(dry)

	str := []string{}
	for _, s := range rpms {
		str = append(str, fmt.Sprintf("%s %s %s", s[0], s[1], s[2]))
	}
	plural := ""
	if len(rpms) > 1 {
		plural = "s"
	}
	ctx.msg.Infof("removing RPM%s:\n%v\n", plural, strings.Join(str, "\n"))

	err = ctx.RemoveRPM(rpms, force)
	return err
}
