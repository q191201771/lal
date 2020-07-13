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
	TypeidAudio           = uint8(8)
	TypeidVideo           = uint8(9)
	TypeidDataMessageAMF0 = uint8(18) // meta

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

type AVMsg struct {
	Header  Header
	Payload []byte // 不包含 rtmp 头
}

// 这部分内容，和httpflv中的类似
const (
	frameTypeKey   uint8 = 1
	frameTypeInter uint8 = 2
	SoundFormatAAC uint8 = 10

	codecIDAVC  uint8 = 7
	codecIDHEVC uint8 = 12

	AVCKeyFrame   = frameTypeKey<<4 | codecIDAVC
	AVCInterFrame = frameTypeInter<<4 | codecIDAVC

	HEVCKeyFrame   = frameTypeKey<<4 | codecIDHEVC
	HEVCInterFrame = frameTypeInter<<4 | codecIDHEVC

	AVCPacketTypeSeqHeader uint8 = 0
	AVCPacketTypeNALU      uint8 = 1

	HEVCPacketTypeSeqHeader uint8 = 0
	HEVCPacketTypeNALU      uint8 = 1

	AACPacketTypeSeqHeader uint8 = 0
	AACPacketTypeRaw       uint8 = 1
)

type OnReadRTMPAVMsg func(msg AVMsg)

func (msg AVMsg) IsAVCKeySeqHeader() bool {
	return msg.Header.MsgTypeID == TypeidVideo && msg.Payload[0] == AVCKeyFrame && msg.Payload[1] == AVCPacketTypeSeqHeader
}

func (msg AVMsg) IsHEVCKeySeqHeader() bool {
	return msg.Header.MsgTypeID == TypeidVideo && msg.Payload[0] == HEVCKeyFrame && msg.Payload[1] == HEVCPacketTypeSeqHeader
}

func (msg AVMsg) IsVideoKeySeqHeader() bool {
	return msg.IsAVCKeySeqHeader() || msg.IsHEVCKeySeqHeader()
}

func (msg AVMsg) IsAVCKeyNALU() bool {
	return msg.Header.MsgTypeID == TypeidVideo && msg.Payload[0] == AVCKeyFrame && msg.Payload[1] == AVCPacketTypeNALU
}

func (msg AVMsg) IsHEVCKeyNALU() bool {
	return msg.Header.MsgTypeID == TypeidVideo && msg.Payload[0] == HEVCKeyFrame && msg.Payload[1] == HEVCPacketTypeNALU
}

func (msg AVMsg) IsVideoKeyNALU() bool {
	return msg.IsAVCKeyNALU() || msg.IsHEVCKeyNALU()
}

func (msg AVMsg) IsAACSeqHeader() bool {
	return msg.Header.MsgTypeID == TypeidAudio && (msg.Payload[0]>>4) == SoundFormatAAC && msg.Payload[1] == AACPacketTypeSeqHeader
}

func ParseMetadata(b []byte) (ObjectPairArray, error) {
	_, l, err := AMF0.ReadString(b)
	if err != nil {
		return nil, err
	}
	opa, _, err := AMF0.ReadObjectOrArray(b[l:])
	return opa, err
}
