// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/hls"

	"github.com/q191201771/naza/pkg/bininfo"
	"github.com/q191201771/naza/pkg/nazalog"
	//"github.com/felixge/fgprof"
)

var (
	config *Config
	sm     *ServerManager
)

// TODO(chef) 临时供innertest使用，后面应该重构
func GetConfig() *Config {
	return config
}

func Init(confFile string) {
	LoadConfAndInitLog(confFile)

	dir, _ := os.Getwd()
	nazalog.Infof("wd: %s", dir)
	nazalog.Infof("args: %s", strings.Join(os.Args, " "))
	nazalog.Infof("bininfo: %s", bininfo.StringifySingleLine())
	nazalog.Infof("version: %s", base.LalFullInfo)
	nazalog.Infof("github: %s", base.LalGithubSite)
	nazalog.Infof("doc: %s", base.LalDocSite)

	if config.HlsConfig.Enable && config.HlsConfig.UseMemoryAsDiskFlag {
		nazalog.Infof("hls use memory as disk.")
		hls.SetUseMemoryAsDiskFlag(true)
	}

	if config.RecordConfig.EnableFlv {
		if err := os.MkdirAll(config.RecordConfig.FlvOutPath, 0777); err != nil {
			nazalog.Errorf("record flv mkdir error. path=%s, err=%+v", config.RecordConfig.FlvOutPath, err)
		}
		if err := os.MkdirAll(config.RecordConfig.MpegtsOutPath, 0777); err != nil {
			nazalog.Errorf("record mpegts mkdir error. path=%s, err=%+v", config.RecordConfig.MpegtsOutPath, err)
		}
	}
}

func RunLoop() {
	sm = NewServerManager()

	if config.PprofConfig.Enable {
		go runWebPprof(config.PprofConfig.Addr)
	}
	go runSignalHandler(func() {
		sm.Dispose()
	})

	err := sm.RunLoop()
	nazalog.Errorf("server manager loop break. err=%+v", err)
}

func Dispose() {
	sm.Dispose()
}

func runWebPprof(addr string) {
	nazalog.Infof("start web pprof listen. addr=%s", addr)

	//nazalog.Warn("start fgprof.")
	//http.DefaultServeMux.Handle("/debug/fgprof", fgprof.Handler())

	if err := http.ListenAndServe(addr, nil); err != nil {
		nazalog.Error(err)
		return
	}
}
