// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/bininfo"
	"github.com/q191201771/naza/pkg/nazalog"
)

var (
	config *Config
)

func Entry(confFile string) {
	config = loadConf(confFile)
	initLog(config.LogConfig)
	nazalog.Infof("bininfo: %s", bininfo.StringifySingleLine())
	nazalog.Infof("version: %s", base.LALFullInfo)

	sm := NewServerManager()

	if config.PProfConfig.Enable {
		go runWebPProf(config.PProfConfig.Addr)
	}
	go runSignalHandler(func() {
		sm.Dispose()
	})

	sm.RunLoop()
}

func loadConf(confFile string) *Config {
	config, err := LoadConf(confFile)
	if err != nil {
		nazalog.Errorf("load conf failed. file=%s err=%+v", confFile, err)
		os.Exit(1)
	}
	nazalog.Infof("load conf file succ. file=%s content=%+v", confFile, config)
	return config
}

func initLog(opt nazalog.Option) {
	if err := nazalog.Init(func(option *nazalog.Option) {
		*option = opt
	}); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "initial log failed. err=%+v\n", err)
		os.Exit(1)
	}
	nazalog.Info("initial log succ.")
}

func runWebPProf(addr string) {
	nazalog.Infof("start web pprof listen. addr=%s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		nazalog.Error(err)
		return
	}
}
