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

const ConfVersion = "v0.2.3"

const (
	defaultHlsCleanupMode    = hls.CleanupModeInTheEnd
	defaultHttpflvUrlPattern = "/live/"
	defaultHttptsUrlPattern  = "/live/"
	defaultHlsUrlPattern     = "/hls/"
)

type Config struct {
	ConfVersion       string            `json:"conf_version"`
	RtmpConfig        RtmpConfig        `json:"rtmp"`
	DefaultHttpConfig DefaultHttpConfig `json:"default_http"`
	HttpflvConfig     HttpflvConfig     `json:"httpflv"`
	HlsConfig         HlsConfig         `json:"hls"`
	HttptsConfig      HttptsConfig      `json:"httpts"`
	RtspConfig        RtspConfig        `json:"rtsp"`
	RecordConfig      RecordConfig      `json:"record"`
	RelayPushConfig   RelayPushConfig   `json:"relay_push"`
	RelayPullConfig   RelayPullConfig   `json:"relay_pull"`

	HttpApiConfig    HttpApiConfig    `json:"http_api"`
	ServerId         string           `json:"server_id"`
	HttpNotifyConfig HttpNotifyConfig `json:"http_notify"`
	PprofConfig      PprofConfig      `json:"pprof"`
	LogConfig        nazalog.Option   `json:"log"`
}

type RtmpConfig struct {
	Enable                   bool   `json:"enable"`
	Addr                     string `json:"addr"`
	GopNum                   int    `json:"gop_num"`
	MergeWriteSize           int    `json:"merge_write_size"`
	AddDummyAudioEnable      bool   `json:"add_dummy_audio_enable"`
	AddDummyAudioWaitAudioMs int    `json:"add_dummy_audio_wait_audio_ms"`
}

type DefaultHttpConfig struct {
	CommonHttpAddrConfig
}

type HttpflvConfig struct {
	CommonHttpServerConfig

	GopNum int `json:"gop_num"`
}

type HttptsConfig struct {
	CommonHttpServerConfig
}

type HlsConfig struct {
	CommonHttpServerConfig

	UseMemoryAsDiskFlag bool `json:"use_memory_as_disk_flag"`
	hls.MuxerConfig
}

type RtspConfig struct {
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
}

type RecordConfig struct {
	EnableFlv     bool   `json:"enable_flv"`
	FlvOutPath    string `json:"flv_out_path"`
	EnableMpegts  bool   `json:"enable_mpegts"`
	MpegtsOutPath string `json:"mpegts_out_path"`
}

type RelayPushConfig struct {
	Enable   bool     `json:"enable"`
	AddrList []string `json:"addr_list"`
}

type RelayPullConfig struct {
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
}

type HttpApiConfig struct {
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
}

type HttpNotifyConfig struct {
	Enable            bool   `json:"enable"`
	UpdateIntervalSec int    `json:"update_interval_sec"`
	OnServerStart     string `json:"on_server_start"`
	OnUpdate          string `json:"on_update"`
	OnPubStart        string `json:"on_pub_start"`
	OnPubStop         string `json:"on_pub_stop"`
	OnSubStart        string `json:"on_sub_start"`
	OnSubStop         string `json:"on_sub_stop"`
	OnRtmpConnect     string `json:"on_rtmp_connect"`
}

type PprofConfig struct {
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
}

type CommonHttpServerConfig struct {
	CommonHttpAddrConfig

	Enable      bool   `json:"enable"`
	EnableHttps bool   `json:"enable_https"`
	UrlPattern  string `json:"url_pattern"`
}

type CommonHttpAddrConfig struct {
	HttpListenAddr  string `json:"http_listen_addr"`
	HttpsListenAddr string `json:"https_listen_addr"`
	HttpsCertFile   string `json:"https_cert_file"`
	HttpsKeyFile    string `json:"https_key_file"`
}

func LoadConfAndInitLog(confFile string) *Config {
	// 读取配置文件并解析原始内容
	rawContent, err := ioutil.ReadFile(confFile)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "read conf file failed. file=%s err=%+v", confFile, err)
		base.OsExitAndWaitPressIfWindows(1)
	}
	if err = json.Unmarshal(rawContent, &config); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "unmarshal conf file failed. file=%s err=%+v", confFile, err)
		base.OsExitAndWaitPressIfWindows(1)
	}
	j, err := nazajson.New(rawContent)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "nazajson unmarshal conf file failed. file=%s err=%+v", confFile, err)
		base.OsExitAndWaitPressIfWindows(1)
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
		config.LogConfig.TimestampWithMsFlag = true
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
		base.OsExitAndWaitPressIfWindows(1)
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
	mergeCommonHttpAddrConfig(&config.HttpflvConfig.CommonHttpAddrConfig, &config.DefaultHttpConfig.CommonHttpAddrConfig)
	mergeCommonHttpAddrConfig(&config.HttptsConfig.CommonHttpAddrConfig, &config.DefaultHttpConfig.CommonHttpAddrConfig)
	mergeCommonHttpAddrConfig(&config.HlsConfig.CommonHttpAddrConfig, &config.DefaultHttpConfig.CommonHttpAddrConfig)

	// 配置不存在时，设置默认值
	if (config.HlsConfig.Enable || config.HlsConfig.EnableHttps) && !j.Exist("hls.cleanup_mode") {
		nazalog.Warnf("config hls.cleanup_mode not exist. set to default which is %d", defaultHlsCleanupMode)
		config.HlsConfig.CleanupMode = defaultHlsCleanupMode
	}
	if (config.HttpflvConfig.Enable || config.HttpflvConfig.EnableHttps) && !j.Exist("httpflv.url_pattern") {
		nazalog.Warnf("config httpflv.url_pattern not exist. set to default wchich is %s", defaultHttpflvUrlPattern)
		config.HttpflvConfig.UrlPattern = defaultHttpflvUrlPattern
	}
	if (config.HttptsConfig.Enable || config.HttptsConfig.EnableHttps) && !j.Exist("httpts.url_pattern") {
		nazalog.Warnf("config httpts.url_pattern not exist. set to default wchich is %s", defaultHttptsUrlPattern)
		config.HttptsConfig.UrlPattern = defaultHttptsUrlPattern
	}
	if (config.HlsConfig.Enable || config.HlsConfig.EnableHttps) && !j.Exist("hls.url_pattern") {
		nazalog.Warnf("config hls.url_pattern not exist. set to default wchich is %s", defaultHlsUrlPattern)
		config.HttpflvConfig.UrlPattern = defaultHlsUrlPattern
	}

	// 对一些常见的格式错误做修复
	// 确保url pattern以`/`开始，并以`/`结束
	if urlPattern, changed := ensureStartAndEndWithSlash(config.HttpflvConfig.UrlPattern); changed {
		nazalog.Warnf("fix config. httpflv.url_pattern %s -> %s", config.HttpflvConfig.UrlPattern, urlPattern)
		config.HttpflvConfig.UrlPattern = urlPattern
	}
	if urlPattern, changed := ensureStartAndEndWithSlash(config.HttptsConfig.UrlPattern); changed {
		nazalog.Warnf("fix config. httpts.url_pattern %s -> %s", config.HttptsConfig.UrlPattern, urlPattern)
		config.HttpflvConfig.UrlPattern = urlPattern
	}
	if urlPattern, changed := ensureStartAndEndWithSlash(config.HlsConfig.UrlPattern); changed {
		nazalog.Warnf("fix config. hls.url_pattern %s -> %s", config.HlsConfig.UrlPattern, urlPattern)
		config.HttpflvConfig.UrlPattern = urlPattern
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
func mergeCommonHttpAddrConfig(dst, src *CommonHttpAddrConfig) {
	if dst.HttpListenAddr == "" && src.HttpListenAddr != "" {
		dst.HttpListenAddr = src.HttpListenAddr
	}
	if dst.HttpsListenAddr == "" && src.HttpsListenAddr != "" {
		dst.HttpsListenAddr = src.HttpsListenAddr
	}
	if dst.HttpsCertFile == "" && src.HttpsCertFile != "" {
		dst.HttpsCertFile = src.HttpsCertFile
	}
	if dst.HttpsKeyFile == "" && src.HttpsKeyFile != "" {
		dst.HttpsKeyFile = src.HttpsKeyFile
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
