package rtmp

import (
	"errors"
)

var rtmpErr = errors.New("rtmp: fxxk")

const (
	CSIDAMF             = 5
	CSIDAudio           = 6
	CSIDVideo           = 7
	csidProtocolControl = 2
	csidOverConnection  = 3
	csidOverStream      = 5

	minCSID = 2
	maxCSID = 65599
)

const (
	TypeidAudio              = 8
	TypeidVideo              = 9
	TypeidDataMessageAMF0    = 18 // meta
	typeidSetChunkSize       = 1
	typeidUserControl        = 4
	typeidWinAckSize         = 5
	typeidBandwidth          = 6
	typeidCommandMessageAMF0 = 20
)

const (
	tidClientConnect      = 1
	tidClientCreateStream = 2
	tidClientPlay         = 3
	tidClientPublish      = 3
)

// basic header 3 | message header 11 | extended ts 4
const maxHeaderSize = 18

const maxTimestampInMessageHeader = 0xFFFFFF

const defaultChunkSize = 128 // 未收到对端设置chunk size时的默认值

const (
	MSID0 = 0 // 所有除 publish、play、onStatus 之外的信令
	MSID1 = 1 // publish、play、onStatus 以及 音视频数据
)

// TODO chef
var (
	readBufSize  = 4096
	writeBufSize = 4096
	wChanSize = 1024
)

var windowAcknowledgementSize = 5000000
var peerBandwidth = 5000000
var LocalChunkSize = 4096 // 本端设置的chunk size

// 接收到音视频类型数据时的回调函数。目前被PullSession以及PubSession使用。
type AVMsgObserver interface {
	// @param header:
	// @param timestampAbs: 绝对时间戳
	// @param message: 不包含头内容。回调结束后，PullSession会继续使用这块内存。
	ReadRTMPAVMsgCB(header Header, timestampAbs int, message []byte)
}
