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
	"io/ioutil"

	"github.com/q191201771/naza/pkg/nazajson"
	log "github.com/q191201771/naza/pkg/nazalog"
)

type Config struct {
	RTMP    RTMP       `json:"rtmp"`
	HTTPFLV HTTPFLV    `json:"httpflv"`
	Log     log.Option `json:"log"`
	PProf   PProf      `json:"pprof"`
}

type RTMP struct {
	Addr string `json:"addr"`
}

type HTTPFLV struct {
	SubListenAddr string `json:"sub_listen_addr"`
}

type PProf struct {
	Addr string `json:"addr"`
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
	// 暂时无

	// 配置不存在时，设置默认值
	if !j.Exist("log.level") {
		config.Log.Level = log.LevelDebug
	}
	if !j.Exist("log.filename") {
		config.Log.Filename = "./logs/lals.log"
	}
	if !j.Exist("log.is_to_stdout") {
		config.Log.IsToStdout = true
	}
	if !j.Exist("log.is_rotate_daily") {
		config.Log.IsRotateDaily = true
	}
	if !j.Exist("log.short_file_flag") {
		config.Log.ShortFileFlag = true
	}

	return &config, nil
}
