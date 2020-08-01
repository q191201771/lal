// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import (
	"errors"
)

var ErrRTMP = errors.New("lal.rtmp: fxxk")

const (
	CSIDAMF   = 5
	CSIDAudio = 6
	CSIDVideo = 7

	csidProtocolControl = 2
	csidOverConnection  = 3
	csidOverStream      = 5

	//minCSID = 2
	//maxCSID = 65599
)

const (
	tidClientConnect      = 1
	tidClientCreateStream = 2
	tidClientPlay         = 3
	tidClientPublish      = 3
)

// basic header 3 | message header 11 | extended ts 4
const maxHeaderSize = 18

// rtmp头中3字节时间戳的最大值
const maxTimestampInMessageHeader uint32 = 0xFFFFFF

const defaultChunkSize = 128 // 未收到对端设置chunk size时的默认值

const (
	//MSID0 = 0 // 所有除 publish、play、onStatus 之外的信令
	MSID1 = 1 // publish、play、onStatus 以及 音视频数据
)
