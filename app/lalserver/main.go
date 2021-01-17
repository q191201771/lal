// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/naza/pkg/bininfo"
)

var sm *logic.ServerManager

func main() {
	confFile := parseFlag()
	logic.Entry(confFile)
}

func parseFlag() string {
	binInfoFlag := flag.Bool("v", false, "show bin info")
	cf := flag.String("c", "", "specify conf file")
	flag.Parse()
	if *binInfoFlag {
		_, _ = fmt.Fprint(os.Stderr, bininfo.StringifyMultiLine())
		_, _ = fmt.Fprintln(os.Stderr, base.LALFullInfo)
		os.Exit(0)
	}
	if *cf == "" {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `
Example:
  %s -c %s
`, os.Args[0], filepath.FromSlash("./conf/lalserver.conf.json"))
		base.OSExitAndWaitPressIfWindows(1)
	}
	return *cf
}
