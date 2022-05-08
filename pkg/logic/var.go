// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import "github.com/q191201771/naza/pkg/nazalog"

var Log = nazalog.GetGlobalLogger()

var (
	relayPushTimeoutMs        = 5000
	relayPushWriteAvTimeoutMs = 5000
	relayPullTimeoutMs        = 5000 // 回源拉流的超时时间，rtmp和rtsp都使用它

	// calcSessionStatIntervalSec 计算所有session收发码率的时间间隔
	//
	calcSessionStatIntervalSec uint32 = 5

	// checkSessionAliveIntervalSec
	//
	// - 对于输入型session，检查一定时间内，是否没有收到数据
	// - 对于输出型session，检查一定时间内，是否没有发送数据
	//   注意，这里既检查socket发送阻塞，又检查上层没有给session喂数据
	//
	checkSessionAliveIntervalSec uint32 = 10
)
