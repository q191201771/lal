// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/naza/pkg/nazalog"
)

const ConfVersion = "v0.2.0"

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
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
	GOPNum int    `json:"gop_num"`
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

	Enable      bool `json:"enable"`
	EnableHTTPS bool `json:"enable_https"`
}

type CommonHTTPAddrConfig struct {
	HTTPListenAddr  string `json:"http_listen_addr"`
	HTTPSListenAddr string `json:"https_listen_addr"`
	HTTPSCertFile   string `json:"https_cert_file"`
	HTTPSKeyFile    string `json:"https_key_file"`
}
