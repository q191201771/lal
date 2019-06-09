package rtmp

import (
	"errors"
)

var rtmpErr = errors.New("rtmp error")

const (
	csidProtocolControl = 2
	csidOverConnection  = 3
	csidOverStream      = 5
)

const (
	typeidSetChunkSize       = 1
	typeidUserControl        = 4
	typeidWinAckSize         = 5
	typeidBandwidth          = 6
	typeidAudio              = 8
	typeidVideo              = 9
	typeidDataMessageAMF0    = 18 // meta
	typeidCommandMessageAMF0 = 20
)

const (
	tidClientConnect      = 1
	tidClientCreateStream = 2
	tidClientPlay         = 3
	tidClientPublish      = 3
)

const maxTimestampInMessageHeader = 0xFFFFFF

var defaultChunkSize = 128

var readBufSize = 4096
var writeBufSize = 4096

var windowAcknowledgementSize = 5000000
var peerBandwidth = 5000000
var localChunkSize = 4096

var msid = 1

// 接收到音视频类型数据时的回调函数。目前被PullSession以及PubSession使用。
type AVMessageObserver interface {
	// @param header:
	// @param timestampAbs: 绝对时间戳
	// @param message: 不包含头内容。回调结束后，PullSession会继续使用这块内存。
	ReadAVMessageCB(header Header, timestampAbs int, message []byte)
}
