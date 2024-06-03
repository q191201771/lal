// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import "github.com/q191201771/naza/pkg/nazalog"

var Log = nazalog.GetGlobalLogger()

// ----- hls --------------------
var (
	// AddCors2HlsFlag 是否为hls增加跨域相关的http header
	AddCors2HlsFlag = true
)

// ----- rtmp --------------------
var (
	// RtmpServerSessionReadAvTimeoutMs rtmp server pub session，读音视频数据超时
	RtmpServerSessionReadAvTimeoutMs = 120000
)

// ----- logic --------------------
var (
	// LogicCheckSessionAliveIntervalSec
	//
	// 检查session是否有数据传输的时间间隔，该间隔内没有数据传输的session将被关闭。
	//
	// 对于输入型session，检查一定时间内，是否没有收到数据。
	//
	// 对于输出型session，检查一定时间内，是否没有发送数据。
	// 注意，socket阻塞无法发送和上层没有向该session喂入数据都算没有发送数据。
	//
	LogicCheckSessionAliveIntervalSec uint32 = 120
)
