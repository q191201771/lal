// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

// 服务启动前，设置好一些配置
type Config struct {
	// 本服务HTTP监听端口，用于接收各lal节点的HTTP Notify
	ListenAddr string

	// 配置向本服务汇报的节点信息
	ServerId2Server map[string]Server

	// 级联拉流时，携带该Url参数，使得我们可以区分是级联拉流还是用户拉流
	PullSecretParam string

	// 检测lal节点update报活的超时时间
	ServerTimeoutSec int
}

// lal节点静态配置信息
type Server struct {
	RtmpAddr string // 可用于级联拉流的RTMP地址
	ApiAddr  string // HTTP API接口地址
}
