// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import "errors"

var ErrRTMP = errors.New("lal.rtmp: fxxk")

const (
	CSIDAMF             = 5
	CSIDAudio           = 6
	CSIDVideo           = 7
	csidProtocolControl = 2
	csidOverConnection  = 3
	csidOverStream      = 5

	//minCSID = 2
	//maxCSID = 65599
)

const (
	TypeidAudio              = uint8(8)
	TypeidVideo              = uint8(9)
	TypeidDataMessageAMF0    = uint8(18) // meta
	typeidSetChunkSize       = uint8(1)
	typeidAck                = uint8(3)
	typeidUserControl        = uint8(4)
	typeidWinAckSize         = uint8(5)
	typeidBandwidth          = uint8(6)
	typeidCommandMessageAMF0 = uint8(20)
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

// 接收到音视频类型数据时的回调函数。目前被PullSession以及PubSession使用。
type AVMsgObserver interface {
	// @param header:
	// @param timestampAbs: 绝对时间戳
	// @param message: 不包含头内容。回调结束后，PubSession 会继续使用这块内存。
	ReadRTMPAVMsgCB(header Header, timestampAbs uint32, message []byte)
}
