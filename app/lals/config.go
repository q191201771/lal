// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/q191201771/lal/pkg/logic"

	"github.com/q191201771/naza/pkg/nazajson"
	log "github.com/q191201771/naza/pkg/nazalog"
)

type Config struct {
	logic.Config
	Log   log.Option  `json:"log"`
	PProf PProfConfig `json:"pprof"`
}

type PProfConfig struct {
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
	if !j.Exist("log.assert_behavior") {
		config.Log.AssertBehavior = log.AssertError
	}

	return &config, nil
}
