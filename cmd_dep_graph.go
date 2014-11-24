package main

import (
	"fmt"
	"io/ioutil"
	"strconv"

	graph "code.google.com/p/gographviz"
	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/lhcb-org/lbpkr/yum"
)

func lbpkr_make_cmd_dep_graph() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_dep_graph,
		UsageLine: "dep-graph [options] [<name-pattern> [<version-pattern> [<release-pattern>]]]",
		Short:     "dump the DOT graph of installed RPM packages [<name-pattern> [<version-pattern> [<release-pattern>]]]",
		Long: `
dep-graph dumps the DAG of the installed RPM package(s) satisfying [<name-pattern> [<version-pattern> [<release-pattern>]]].

ex:
 $ lbpkr dep-graph -o graph.dot
 $ lbpkr dep-graph GAUDI
`,
		Flag: *flag.NewFlagSet("lbpkr-dep-graph", flag.ExitOnError),
	}
	add_default_options(cmd)
	cmd.Flag.String("o", "graph.dot", "generate a DOT file holding the dependency graph")
	cmd.Flag.Int("rec-lvl", 1, "recursive-level (-1 to display all the graph)")
	return cmd
}

func lbpkr_run_cmd_dep_graph(cmd *commander.Command, args []string) error {
	var err error

	siteroot := cmd.Flag.Lookup("siteroot").Value.Get().(string)
	debug := cmd.Flag.Lookup("v").Value.Get().(bool)
	dotfname := cmd.Flag.Lookup("o").Value.Get().(string)
	reclvl := cmd.Flag.Lookup("rec-lvl").Value.Get().(int)

	name := ""
	vers := ""
	release := ""

	switch len(args) {
	case 0:
		name = ""
	case 1:
		name = args[0]
	case 2:
		name = args[0]
		vers = args[1]
	case 3:
		name = args[0]
		vers = args[1]
		release = args[2]
	default:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments. expected n=0|1|2|3. got=%d (%v)",
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

	g := graph.NewGraph()
	g.SetName("rpms")
	g.SetDir(true)

	decorate := func(pkg yum.RPM) map[string]string {
		return map[string]string{
			"name":    strconv.Quote(pkg.Name()),
			"version": strconv.Quote(pkg.Version()),
			"release": strconv.Quote(pkg.Release()),
			"epoch":   strconv.Quote(pkg.Epoch()),
		}
	}

	pkgs, err := ctx.ListInstalledPackages(name, vers, release)
	if err != nil {
		return err
	}

	str_in_slice := func(str string, slice []string) bool {
		for _, v := range slice {
			if str == v {
				return true
			}
		}
		return false
	}

	var process func(pkg *yum.Package, lvl int) error

	process = func(pkg *yum.Package, lvl int) error {
		var err error
		root := strconv.Quote(pkg.ID())
		g.AddNode("rpms", root, decorate(pkg))
		reqs := pkg.Requires()
		for _, req := range reqs {
			if str_in_slice(req.Name(), yum.IGNORED_PACKAGES) {
				continue
			}
			dep, err := ctx.Client().FindLatestMatchingRequire(req)
			if err != nil {
				ctx.msg.Infof("no package providing name=%q version=%q release=%q\n",
					req.Name(),
					req.Version(),
					req.Release(),
				)
				continue
			}
			g.AddNode("rpms", strconv.Quote(dep.ID()), decorate(dep))
			g.AddEdge(root, strconv.Quote(dep.ID()), true, nil)
			if lvl < reclvl || reclvl < 0 {
				err = process(dep, lvl+1)
				if err != nil {
					return err
				}
			}
		}
		return err
	}

	for _, pkg := range pkgs {
		err = process(pkg, 1)
		if err != nil {
			return err
		}
	}

	err = ioutil.WriteFile(dotfname, []byte(g.String()), 0644)
	if err != nil {
		return err
	}

	return err
}
