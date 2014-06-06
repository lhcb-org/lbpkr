package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func path_exists(name string) bool {
	_, err := os.Stat(name)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func handle_err(err error) {
	if err != nil {
		if g_ctx != nil {
			g_ctx.msg.Errorf("%v\n", err.Error())
		} else {
			fmt.Fprintf(os.Stderr, "**error** %v\n", err)
		}
		os.Exit(1)
	}
}

func PrintHeader(ctx *Context) {
	now := time.Now()
	ctx.msg.Infof("%s\n", strings.Repeat("=", 80))
	ctx.msg.Infof(
		"<<< %s - start of lbpkr-%s installation >>>\n",
		now, Version,
	)
	ctx.msg.Infof("%s\n", strings.Repeat("=", 80))
	ctx.msg.Debugf("cmd line args: %v\n", os.Args)
}

func PrintTrailer(ctx *Context) {
	now := time.Now()
	ctx.msg.Infof("%s\n", strings.Repeat("=", 80))
	ctx.msg.Infof(
		"<<< %s - end of lbpkr-%s installation >>>\n",
		now, Version,
	)
	ctx.msg.Infof("%s\n", strings.Repeat("=", 80))
}

// EOF
