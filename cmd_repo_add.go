package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func lbpkr_make_cmd_repo_add() *commander.Command {
	cmd := &commander.Command{
		Run:       lbpkr_run_cmd_repo_add,
		UsageLine: "repo-add [options] <repo-name> <repo-url>",
		Short:     "add a repository",
		Long: `repo-add adds a repository source named <repo-name> and located at <repo-url>.

ex:
 $ lbpkr repo-add my-test /some/where
 $ lbpkr repo-add extra http://example.com/rpm

 # add lhcb-release/Fri (2015-01-20, DaVinci-v36r4p1)
 $ lbpkr repo-add lhcb-release/Fri

 # add a nightly lhcb-gaudi-head/Fri
 $ lbpkr repo-add lhcb-gaudi-head/Fri
`,
		Flag: *flag.NewFlagSet("lbpkr-repo-add", flag.ExitOnError),
	}
	add_default_options(cmd)
	cmd.Flag.Int("maxdepth", -1, "maximum depth level of dependency graph (-1: all)")
	return cmd
}

func lbpkr_run_cmd_repo_add(cmd *commander.Command, args []string) error {
	var err error

	debug := cmd.Flag.Lookup("v").Value.Get().(bool)
	siteroot := cmd.Flag.Lookup("siteroot").Value.Get().(string)
	//dmax := cmd.Flag.Lookup("maxdepth").Value.Get().(int)

	cfg := NewConfig(siteroot)
	ctx, err := New(cfg, Debug(debug))
	if err != nil {
		return err
	}
	defer ctx.Close()

	reponame := ""
	repourl := ""

	switch len(args) {
	case 1:
		reponame = strings.Replace(args[0], "/", "-", -1)
		// https://buildlhcb.cern.ch/artifacts/lhcb-gaudi-head/Fri/slot-config.json
		// https://buildlhcb.cern.ch/artifacts/release/lhcb-release/375/slot-config.json
		url := "https://buildlhcb.cern.ch/artifacts/"
		if strings.HasPrefix(args[0], "lhcb-release/") {
			url += "release/"
		}
		if !strings.HasSuffix(url, "/") {
			url += "/"
		}
		url += args[0]
		if strings.HasSuffix(url, "/") {
			url = url[:len(url)-2]
		}

		repourl = url + "/rpm"
		url += "/slot-config.json"
		f, err := getRemoteData(url)
		if err != nil {
			ctx.msg.Errorf("could not download [%s]: %v\n", url, err)
			return err
		}
		defer f.Close()

		data := struct {
			Slot    string `json:"slot"`
			BuildID int    `json:"build_id"`
			Date    string `json:"date"`
		}{}

		err = json.NewDecoder(f).Decode(&data)
		if err != nil {
			ctx.msg.Errorf("could not decode slot-config.json: %v\n", err)
			return err
		}

		ctx.msg.Infof("slot: %q\n", data.Slot)
		ctx.msg.Infof("date: %q (build-id: %v)\n", data.Date, data.BuildID)
		ctx.msg.Infof("url:  %q\n", repourl)

	case 2:
		reponame = args[0]
		repourl = args[1]
	default:
		cmd.Usage()
		return fmt.Errorf("lbpkr: invalid number of arguments. expected n=1|2. got=%d (%v)",
			len(args),
			args,
		)
	}

	err = ctx.AddRepository(reponame, repourl)
	return err
}
