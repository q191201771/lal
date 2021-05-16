// Copyright 2019, Chef.  All rights reserved.
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
	"os"
	"strings"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/naza/pkg/nazajson"
	"github.com/q191201771/naza/pkg/nazalog"
)

const ConfVersion = "v0.2.2"

const (
	defaultHLSCleanupMode    = hls.CleanupModeInTheEnd
	defaultHTTPFLVURLPattern = "/live/"
	defaultHTTPTSURLPattern  = "/live/"
	defaultHLSURLPattern     = "/hls/"
)

type Config struct {
	ConfVersion       string            `json:"conf_version"`
	RTMPConfig        RTMPConfig        `json:"rtmp"`
	DefaultHTTPConfig DefaultHTTPConfig `json:"default_http"`
	HTTPFLVConfig     HTTPFLVConfig     `json:"httpflv"`
	HLSConfig         HLSConfig         `json:"hls"`
	HTTPTSConfig      HTTPTSConfig      `json:"httpts"`
	RTSPConfig        RTSPConfig        `json:"rtsp"`
	RecordConfig      RecordConfig      `json:"record"`
	RelayPushConfig   RelayPushConfig   `json:"relay_push"`
	RelayPullConfig   RelayPullConfig   `json:"relay_pull"`

	HTTPAPIConfig    HTTPAPIConfig    `json:"http_api"`
	ServerID         string           `json:"server_id"`
	HTTPNotifyConfig HTTPNotifyConfig `json:"http_notify"`
	PProfConfig      PProfConfig      `json:"pprof"`
	LogConfig        nazalog.Option   `json:"log"`
}

type RTMPConfig struct {
	Enable         bool   `json:"enable"`
	Addr           string `json:"addr"`
	GOPNum         int    `json:"gop_num"`
	MergeWriteSize int    `json:"merge_write_size"`
}

type DefaultHTTPConfig struct {
	CommonHTTPAddrConfig
}

type HTTPFLVConfig struct {
	CommonHTTPServerConfig

	GOPNum int `json:"gop_num"`
}

type HTTPTSConfig struct {
	CommonHTTPServerConfig
}

type HLSConfig struct {
	CommonHTTPServerConfig

	UseMemoryAsDiskFlag bool `json:"use_memory_as_disk_flag"`
	hls.MuxerConfig
}

type RTSPConfig struct {
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
}

type RecordConfig struct {
	EnableFLV     bool   `json:"enable_flv"`
	FLVOutPath    string `json:"flv_out_path"`
	EnableMPEGTS  bool   `json:"enable_mpegts"`
	MPEGTSOutPath string `json:"mpegts_out_path"`
}

type RelayPushConfig struct {
	Enable   bool     `json:"enable"`
	AddrList []string `json:"addr_list"`
}

type RelayPullConfig struct {
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
}

type HTTPAPIConfig struct {
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
}

type HTTPNotifyConfig struct {
	Enable            bool   `json:"enable"`
	UpdateIntervalSec int    `json:"update_interval_sec"`
	OnServerStart     string `json:"on_server_start"`
	OnUpdate          string `json:"on_update"`
	OnPubStart        string `json:"on_pub_start"`
	OnPubStop         string `json:"on_pub_stop"`
	OnSubStart        string `json:"on_sub_start"`
	OnSubStop         string `json:"on_sub_stop"`
	OnRTMPConnect     string `json:"on_rtmp_connect"`
}

type PProfConfig struct {
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
}

type CommonHTTPServerConfig struct {
	CommonHTTPAddrConfig

	Enable      bool   `json:"enable"`
	EnableHTTPS bool   `json:"enable_https"`
	URLPattern  string `json:"url_pattern"`
}

type CommonHTTPAddrConfig struct {
	HTTPListenAddr  string `json:"http_listen_addr"`
	HTTPSListenAddr string `json:"https_listen_addr"`
	HTTPSCertFile   string `json:"https_cert_file"`
	HTTPSKeyFile    string `json:"https_key_file"`
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
		"record",
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

	// 如果具体的HTTP应用没有设置HTTP监听相关的配置，则尝试使用全局配置
	mergeCommonHTTPAddrConfig(&config.HTTPFLVConfig.CommonHTTPAddrConfig, &config.DefaultHTTPConfig.CommonHTTPAddrConfig)
	mergeCommonHTTPAddrConfig(&config.HTTPTSConfig.CommonHTTPAddrConfig, &config.DefaultHTTPConfig.CommonHTTPAddrConfig)
	mergeCommonHTTPAddrConfig(&config.HLSConfig.CommonHTTPAddrConfig, &config.DefaultHTTPConfig.CommonHTTPAddrConfig)

	// 配置不存在时，设置默认值
	if (config.HLSConfig.Enable || config.HLSConfig.EnableHTTPS) && !j.Exist("hls.cleanup_mode") {
		nazalog.Warnf("config hls.cleanup_mode not exist. set to default which is %d", defaultHLSCleanupMode)
		config.HLSConfig.CleanupMode = defaultHLSCleanupMode
	}
	if (config.HTTPFLVConfig.Enable || config.HTTPFLVConfig.EnableHTTPS) && !j.Exist("httpflv.url_pattern") {
		nazalog.Warnf("config httpflv.url_pattern not exist. set to default wchich is %s", defaultHTTPFLVURLPattern)
		config.HTTPFLVConfig.URLPattern = defaultHTTPFLVURLPattern
	}
	if (config.HTTPTSConfig.Enable || config.HTTPTSConfig.EnableHTTPS) && !j.Exist("httpts.url_pattern") {
		nazalog.Warnf("config httpts.url_pattern not exist. set to default wchich is %s", defaultHTTPTSURLPattern)
		config.HTTPTSConfig.URLPattern = defaultHTTPTSURLPattern
	}
	if (config.HLSConfig.Enable || config.HLSConfig.EnableHTTPS) && !j.Exist("hls.url_pattern") {
		nazalog.Warnf("config hls.url_pattern not exist. set to default wchich is %s", defaultHLSURLPattern)
		config.HTTPFLVConfig.URLPattern = defaultHLSURLPattern
	}

	// 对一些常见的格式错误做修复
	// 确保url pattern以`/`开始，并以`/`结束
	if urlPattern, changed := ensureStartAndEndWithSlash(config.HTTPFLVConfig.URLPattern); changed {
		nazalog.Warnf("fix config. httpflv.url_pattern %s -> %s", config.HTTPFLVConfig.URLPattern, urlPattern)
		config.HTTPFLVConfig.URLPattern = urlPattern
	}
	if urlPattern, changed := ensureStartAndEndWithSlash(config.HTTPTSConfig.URLPattern); changed {
		nazalog.Warnf("fix config. httpts.url_pattern %s -> %s", config.HTTPTSConfig.URLPattern, urlPattern)
		config.HTTPFLVConfig.URLPattern = urlPattern
	}
	if urlPattern, changed := ensureStartAndEndWithSlash(config.HLSConfig.URLPattern); changed {
		nazalog.Warnf("fix config. hls.url_pattern %s -> %s", config.HLSConfig.URLPattern, urlPattern)
		config.HTTPFLVConfig.URLPattern = urlPattern
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
func mergeCommonHTTPAddrConfig(dst, src *CommonHTTPAddrConfig) {
	if dst.HTTPListenAddr == "" && src.HTTPListenAddr != "" {
		dst.HTTPListenAddr = src.HTTPListenAddr
	}
	if dst.HTTPSListenAddr == "" && src.HTTPSListenAddr != "" {
		dst.HTTPSListenAddr = src.HTTPSListenAddr
	}
	if dst.HTTPSCertFile == "" && src.HTTPSCertFile != "" {
		dst.HTTPSCertFile = src.HTTPSCertFile
	}
	if dst.HTTPSKeyFile == "" && src.HTTPSKeyFile != "" {
		dst.HTTPSKeyFile = src.HTTPSKeyFile
	}
}

func ensureStartWithSlash(in string) (out string, changed bool) {
	if in == "" {
		return in, false
	}
	if in[0] == '/' {
		return in, false
	}
	return "/" + in, true
}

func ensureEndWithSlash(in string) (out string, changed bool) {
	if in == "" {
		return in, false
	}
	if in[len(in)-1] == '/' {
		return in, false
	}
	return in + "/", true
}

func ensureStartAndEndWithSlash(in string) (out string, changed bool) {
	n, c := ensureStartWithSlash(in)
	n2, c2 := ensureEndWithSlash(n)
	return n2, c || c2
}
