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
	"errors"
	"io/ioutil"

	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/naza/pkg/nazajson"
	"github.com/q191201771/naza/pkg/nazalog"
)

type Config struct {
	RTMPConfig      RTMPConfig      `json:"rtmp"`
	HTTPFLVConfig   HTTPFLVConfig   `json:"httpflv"`
	HLSConfig       HLSConfig       `json:"hls"`
	RTSPConfig      RTSPConfig      `json:"rtsp"`
	RelayPushConfig RelayPushConfig `json:"relay_push"`
	RelayPullConfig RelayPullConfig `json:"relay_pull"`

	PProfConfig PProfConfig    `json:"pprof"`
	LogConfig   nazalog.Option `json:"log"`
}

type RTMPConfig struct {
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
	GOPNum int    `json:"gop_num"`
}

type HTTPFLVConfig struct {
	Enable        bool   `json:"enable"`
	SubListenAddr string `json:"sub_listen_addr"`
	GOPNum        int    `json:"gop_num"`
}

type HLSConfig struct {
	Enable        bool   `json:"enable"`
	SubListenAddr string `json:"sub_listen_addr"`
	hls.MuxerConfig
}

type RTSPConfig struct {
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
}

type RelayPushConfig struct {
	Enable   bool     `json:"enable"`
	AddrList []string `json:"addr_list"`
}

type RelayPullConfig struct {
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
}

type PProfConfig struct {
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
}

func LoadConf(confFile string) (*Config, error) {
	var config Config
	rawContent, err := ioutil.ReadFile(confFile)
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(rawContent, &config); err != nil {
		return nil, err
	}

	j, err := nazajson.New(rawContent)
	if err != nil {
		return nil, err
	}

	// 检查配置必须项
	if !j.Exist("rtmp") || !j.Exist("httpflv") || !j.Exist("hls") || !j.Exist("rtsp") ||
		!j.Exist("relay_push") || !j.Exist("relay_pull") ||
		!j.Exist("pprof") || !j.Exist("log") {
		return &config, errors.New("missing key field in config file")
	}

	// 配置不存在时，设置默认值
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
	if !j.Exist("log.assert_behavior") {
		config.LogConfig.AssertBehavior = nazalog.AssertError
	}

	return &config, nil
}
