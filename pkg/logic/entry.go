// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/naza/pkg/nazajson"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/bininfo"
	"github.com/q191201771/naza/pkg/nazalog"
	//"github.com/felixge/fgprof"
)

var (
	config *Config
	sm     *ServerManager
)

func Entry(confFile string) {
	LoadConfAndInitLog(confFile)
	nazalog.Infof("args: %s", strings.Join(os.Args, " "))
	nazalog.Infof("bininfo: %s", bininfo.StringifySingleLine())
	nazalog.Infof("version: %s", base.LALFullInfo)
	nazalog.Infof("github: %s", base.LALGithubSite)
	nazalog.Infof("doc: %s", base.LALDocSite)

	sm = NewServerManager()

	if config.PProfConfig.Enable {
		go runWebPProf(config.PProfConfig.Addr)
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

func LoadConfAndInitLog(confFile string) *Config {
	// 读取配置文件并解析原始内容
	rawContent, err := ioutil.ReadFile(confFile)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "read conf file failed. file=%s err=%+v", confFile, err)
		base.OSExitAndWaitPressIfWindows(1)
	}
	if err = json.Unmarshal(rawContent, &config); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "unmarshal conf file failed. file=%s err=%+v", confFile, err)
		base.OSExitAndWaitPressIfWindows(1)
	}
	j, err := nazajson.New(rawContent)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "nazajson unmarshal conf file failed. file=%s err=%+v", confFile, err)
		base.OSExitAndWaitPressIfWindows(1)
	}

	// 初始化日志，注意，这一步尽量提前，使得后续的日志内容按我们的日志配置输出
	// 日志配置项不存在时，设置默认值
	if !j.Exist("log.level") {
		config.LogConfig.Level = nazalog.LevelDebug
	}
	if !j.Exist("log.filename") {
		config.LogConfig.Filename = "./logs/lalserver.log"
	}
	if !j.Exist("log.is_to_stdout") {
		config.LogConfig.IsToStdout = true
	}
	if !j.Exist("log.is_rotate_daily") {
		config.LogConfig.IsRotateDaily = true
	}
	if !j.Exist("log.short_file_flag") {
		config.LogConfig.ShortFileFlag = true
	}
	if !j.Exist("log.timestamp_flag") {
		config.LogConfig.TimestampFlag = true
	}
	if !j.Exist("log.timestamp_with_ms_flag") {
		config.LogConfig.TimestampWithMSFlag = true
	}
	if !j.Exist("log.level_flag") {
		config.LogConfig.LevelFlag = true
	}
	if !j.Exist("log.assert_behavior") {
		config.LogConfig.AssertBehavior = nazalog.AssertError
	}
	if err := nazalog.Init(func(option *nazalog.Option) {
		*option = config.LogConfig
	}); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "initial log failed. err=%+v\n", err)
		base.OSExitAndWaitPressIfWindows(1)
	}
	nazalog.Info("initial log succ.")

	// 打印Logo
	nazalog.Info(`
    __    ___    __ 
   / /   /   |  / / 
  / /   / /| | / /  
 / /___/ ___ |/ /___
/_____/_/  |_/_____/
`)

	// 检查配置版本号是否匹配
	if config.ConfVersion != ConfVersion {
		nazalog.Warnf("config version invalid. conf version of lalserver=%s, conf version of config file=%s",
			ConfVersion, config.ConfVersion)
	}

	// 检查一级配置项
	keyFieldList := []string{
		"rtmp",
		"httpflv",
		"hls",
		"httpts",
		"rtsp",
		"relay_push",
		"relay_pull",
		"http_api",
		"http_notify",
		"pprof",
		"log",
	}
	for _, kf := range keyFieldList {
		if !j.Exist(kf) {
			nazalog.Warnf("missing config item %s", kf)
		}
	}

	// 配置不存在时，设置默认值
	if !j.Exist("hls.cleanup_mode") {
		const defaultMode = hls.CleanupModeInTheEnd
		nazalog.Warnf("config hls.cleanup_mode not exist. default is %d", defaultMode)
		config.HLSConfig.CleanupMode = defaultMode
	}

	// 把配置文件原始内容中的换行去掉，使得打印日志时紧凑一些
	lines := strings.Split(string(rawContent), "\n")
	if len(lines) == 1 {
		lines = strings.Split(string(rawContent), "\r\n")
	}
	var tlines []string
	for _, l := range lines {
		tlines = append(tlines, strings.TrimSpace(l))
	}
	compactRawContent := strings.Join(tlines, " ")
	nazalog.Infof("load conf file succ. filename=%s, raw content=%s parsed=%+v", confFile, compactRawContent, config)

	return config
}

func runWebPProf(addr string) {
	nazalog.Infof("start web pprof listen. addr=%s", addr)

	//nazalog.Warn("start fgprof.")
	//http.DefaultServeMux.Handle("/debug/fgprof", fgprof.Handler())

	if err := http.ListenAndServe(addr, nil); err != nil {
		nazalog.Error(err)
		return
	}
}
