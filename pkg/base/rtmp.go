// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

const (
	// spec-rtmp_specification_1.0.pdf
	// 7.1. Types of Messages
	RTMPTypeIDAudio              uint8 = 8
	RTMPTypeIDVideo              uint8 = 9
	RTMPTypeIDMetadata           uint8 = 18 // RTMPTypeIDDataMessageAMF0
	RTMPTypeIDSetChunkSize       uint8 = 1
	RTMPTypeIDAck                uint8 = 3
	RTMPTypeIDUserControl        uint8 = 4
	RTMPTypeIDWinAckSize         uint8 = 5
	RTMPTypeIDBandwidth          uint8 = 6
	RTMPTypeIDCommandMessageAMF3 uint8 = 17
	RTMPTypeIDCommandMessageAMF0 uint8 = 20

	// user control message type
	RTMPUserControlStreamBegin uint8 = 0
	RTMPUserControlRecorded    uint8 = 4

	// spec-video_file_format_spec_v10.pdf
	// Video tags
	//   VIDEODATA
	//     FrameType UB[4]
	//     CodecID   UB[4]
	//   AVCVIDEOPACKET
	//     AVCPacketType   UI8
	//     CompositionTime SI24
	//     Data            UI8[n]
	RTMPFrameTypeKey   uint8 = 1
	RTMPFrameTypeInter uint8 = 2

	RTMPCodecIDAVC  uint8 = 7
	RTMPCodecIDHEVC uint8 = 12

	RTMPAVCPacketTypeSeqHeader  uint8 = 0
	RTMPAVCPacketTypeNALU       uint8 = 1
	RTMPHEVCPacketTypeSeqHeader       = RTMPAVCPacketTypeSeqHeader
	RTMPHEVCPacketTypeNALU            = RTMPAVCPacketTypeNALU

	RTMPAVCKeyFrame   = RTMPFrameTypeKey<<4 | RTMPCodecIDAVC
	RTMPHEVCKeyFrame  = RTMPFrameTypeKey<<4 | RTMPCodecIDHEVC
	RTMPAVCInterFrame = RTMPFrameTypeInter<<4 | RTMPCodecIDAVC

	// spec-video_file_format_spec_v10.pdf
	// Audio tags
	//   AUDIODATA
	//     SoundFormat UB[4]
	//     SoundRate   UB[2]
	//     SoundSize   UB[1]
	//     SoundType   UB[1]
	//   AACAUDIODATA
	//     AACPacketType UI8
	//     Data          UI8[n]
	RTMPSoundFormatAAC         uint8 = 10
	RTMPAACPacketTypeSeqHeader       = 0
	RTMPAACPacketTypeRaw             = 1
)

type RTMPHeader struct {
	CSID         int
	MsgLen       uint32 // 不包含header的大小
	MsgTypeID    uint8  // 8 audio 9 video 18 metadata
	MsgStreamID  int
	TimestampAbs uint32 // 经过计算得到的流上的绝对时间戳
}

type RTMPMsg struct {
	Header  RTMPHeader
	Payload []byte // 不包含 rtmp 头
}

func (msg RTMPMsg) IsAVCKeySeqHeader() bool {
	return msg.Header.MsgTypeID == RTMPTypeIDVideo && msg.Payload[0] == RTMPAVCKeyFrame && msg.Payload[1] == RTMPAVCPacketTypeSeqHeader
}

func (msg RTMPMsg) IsHEVCKeySeqHeader() bool {
	return msg.Header.MsgTypeID == RTMPTypeIDVideo && msg.Payload[0] == RTMPHEVCKeyFrame && msg.Payload[1] == RTMPHEVCPacketTypeSeqHeader
}

func (msg RTMPMsg) IsVideoKeySeqHeader() bool {
	return msg.IsAVCKeySeqHeader() || msg.IsHEVCKeySeqHeader()
}

func (msg RTMPMsg) IsAVCKeyNALU() bool {
	return msg.Header.MsgTypeID == RTMPTypeIDVideo && msg.Payload[0] == RTMPAVCKeyFrame && msg.Payload[1] == RTMPAVCPacketTypeNALU
}

func (msg RTMPMsg) IsHEVCKeyNALU() bool {
	return msg.Header.MsgTypeID == RTMPTypeIDVideo && msg.Payload[0] == RTMPHEVCKeyFrame && msg.Payload[1] == RTMPHEVCPacketTypeNALU
}

func (msg RTMPMsg) IsVideoKeyNALU() bool {
	return msg.IsAVCKeyNALU() || msg.IsHEVCKeyNALU()
}

func (msg RTMPMsg) IsAACSeqHeader() bool {
	return msg.Header.MsgTypeID == RTMPTypeIDAudio && (msg.Payload[0]>>4) == RTMPSoundFormatAAC && msg.Payload[1] == RTMPAACPacketTypeSeqHeader
}
