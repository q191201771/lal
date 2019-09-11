package main

import (
	"encoding/json"
	"github.com/q191201771/nezha/pkg/log"
	"io/ioutil"
)

type Config struct {
	RTMP RTMP `json:"rtmp"`
	Log log.Config

	SubIdleTimeout int64   `json:"sub_idle_timeout"`
	GOPCacheNum    int     `json:"gop_cache_number"`
	HTTPFlv        HTTPFlv `json:"httpflv"`
	Pull           Pull    `json:"pull"`
}

type HTTPFlv struct {
	SubListenAddr string `json:"sub_listen_addr"`
}

type RTMP struct {
	Addr string `json:"addr"`
}

type Pull struct {
	Type                      string `json:"type"`
	Addr                      string `json:"addr"`
	ConnectTimeout            int64  `json:"connect_timeout"`
	ReadTimeout               int64  `json:"read_timeout"`
	StopPullWhileNoSubTimeout int64  `json:"stop_pull_while_no_sub_timeout"`
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

	// TODO chef: check item valid.

	return &config, nil
}
