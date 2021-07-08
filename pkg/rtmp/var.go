// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

// TODO chef
// 一些更专业的配置项，暂时只在该源码文件中配置，不提供外部配置接口
var (
	readBufSize                   = 4096  // client/server session connection读缓冲的大小
	writeBufSize                  = 4096  // client/server session connection写缓冲的大小
	wChanSize                     = 1024  // client/server session 发送数据时，channel 的大小
	serverSessionReadAvTimeoutMs  = 10000 // server pub session，读音视频数据超时
	serverSessionWriteAvTimeoutMs = 10000 // server sub session，写音视频数据超时

	LocalChunkSize            = 4096 // 本端设置的 chunk size
	windowAcknowledgementSize = 5000000
	peerBandwidth             = 5000000
)
