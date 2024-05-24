// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import "path/filepath"

// Config 服务启动前，设置好一些配置
type Config struct {
	// 本服务HTTP监听端口，用于接收各lal节点的HTTP Notify
	ListenAddr string `json:"listen_addr"`

	// ServerId2Server `json:"server_id2server"`
	// 配置向本服务汇报的节点信息。
	// 如果没有配置的serverId向本服务汇报，本服务讲认为该汇报信息无效。
	ServerId2Server map[string]Server `json:"servers"`

	// 级联拉流时，携带该Url参数，使得我们可以区分是级联拉流还是用户拉流
	PullSecretParam string `json:"pull_secret_param"`

	// 检测lal节点update报活的超时时间
	ServerTimeoutSec int `json:"server_timeout_sec"`

	// MaxSubSessionPerIp
	// 当一个client IP对应的sub session大于这个阈值时，我们认为是恶意行为，踢掉这个IP的所有sub session。
	// 如果设置为0或-1，表示不限制。
	MaxSubSessionPerIp int `json:"max_sub_session_per_ip"`

	// MaxSubDurationSec
	// 当一个sub session的持续时间超过这个阈值时，我们认为是恶意行为，踢掉这个sub session。
	// 如果设置为0或-1，表示不限制。
	MaxSubDurationSec int `json:"max_sub_duration_sec"`
}

// Server lal节点静态配置信息
type Server struct {
	// 可用于级联拉流的RTMP地址
	RtmpAddr string `json:"rtmp_addr"`
	// HTTP API接口地址，比如向节点发送kick_session时使用
	ApiAddr string `json:"api_addr"`
}

var config Config

var DefaultConfFilenameList = []string{
	filepath.FromSlash("dispatch.conf.json"),
	filepath.FromSlash("./conf/dispatch.conf.json"),
	filepath.FromSlash("../dispatch.conf.json"),
	filepath.FromSlash("../conf/dispatch.conf.json"),
	filepath.FromSlash("../../dispatch.conf.json"),
	filepath.FromSlash("../../conf/dispatch.conf.json"),
	filepath.FromSlash("../../../dispatch.conf.json"),
	filepath.FromSlash("../../../conf/dispatch.conf.json"),
	filepath.FromSlash("lal/conf/dispatch.conf.json"),
}
