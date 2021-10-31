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

	"github.com/q191201771/naza/pkg/nazalog"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/naza/pkg/bininfo"
)

func main() {
	defer nazalog.Sync()

	confFile := parseFlag()
	sm := logic.NewLalServer(confFile, func(option *logic.Option) {
	})
	err := sm.RunLoop()
	nazalog.Infof("server manager done. err=%+v", err)
}

func parseFlag() string {
	binInfoFlag := flag.Bool("v", false, "show bin info")
	cf := flag.String("c", "", "specify conf file")
	flag.Parse()

	if *binInfoFlag {
		_, _ = fmt.Fprint(os.Stderr, bininfo.StringifyMultiLine())
		_, _ = fmt.Fprintln(os.Stderr, base.LalFullInfo)
		os.Exit(0)
	}

	// 运行参数中有配置文件，直接返回
	if *cf != "" {
		return *cf
	}

	// 运行参数中没有配置文件，尝试从几个默认位置读取
	nazalog.Warnf("config file did not specify in the command line, try to load it in the usual path.")
	defaultConfigFileList := []string{
		filepath.FromSlash("lalserver.conf.json"),
		filepath.FromSlash("./conf/lalserver.conf.json"),
		filepath.FromSlash("../conf/lalserver.conf.json"),
		filepath.FromSlash("../lalserver.conf.json"),
		filepath.FromSlash("../../lalserver.conf.json"),
		filepath.FromSlash("../../conf/lalserver.conf.json"),
		filepath.FromSlash("lal/conf/lalserver.conf.json"),
	}
	for _, dcf := range defaultConfigFileList {
		fi, err := os.Stat(dcf)
		if err == nil && fi.Size() > 0 && !fi.IsDir() {
			nazalog.Warnf("%s exist. using it as config file.", dcf)
			return dcf
		} else {
			nazalog.Warnf("%s not exist.", dcf)
		}
	}

	// 所有默认位置都找不到配置文件，退出程序
	flag.Usage()
	_, _ = fmt.Fprintf(os.Stderr, `
Example:
  %s -c %s

Github: %s
Doc: %s
`, os.Args[0], filepath.FromSlash("./conf/lalserver.conf.json"), base.LalGithubSite, base.LalDocSite)
	base.OsExitAndWaitPressIfWindows(1)
	return *cf
}
