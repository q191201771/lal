// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import "github.com/q191201771/naza/pkg/bele"

const (
	// RtmpTypeIdAudio spec-rtmp_specification_1.0.pdf
	// 7.1. Types of Messages
	RtmpTypeIdAudio              uint8 = 8
	RtmpTypeIdVideo              uint8 = 9
	RtmpTypeIdMetadata           uint8 = 18 // RtmpTypeIdDataMessageAmf0
	RtmpTypeIdSetChunkSize       uint8 = 1
	RtmpTypeIdAck                uint8 = 3
	RtmpTypeIdUserControl        uint8 = 4
	RtmpTypeIdWinAckSize         uint8 = 5
	RtmpTypeIdBandwidth          uint8 = 6
	RtmpTypeIdCommandMessageAmf3 uint8 = 17
	RtmpTypeIdCommandMessageAmf0 uint8 = 20
	RtmpTypeIdAggregateMessage   uint8 = 22

	// RtmpUserControlStreamBegin RtmpUserControlXxx...
	//
	// user control message type
	//
	RtmpUserControlStreamBegin  uint8 = 0
	RtmpUserControlRecorded     uint8 = 4
	RtmpUserControlPingRequest  uint8 = 6
	RtmpUserControlPingResponse uint8 = 7

	// RtmpFrameTypeKey spec-video_file_format_spec_v10.pdf
	// Video tags
	//   VIDEODATA
	//     FrameType UB[4]
	//     CodecId   UB[4]
	//   AVCVIDEOPACKET
	//     AVCPacketType   UI8
	//     CompositionTime SI24
	//     Data            UI8[n]
	RtmpFrameTypeKey   uint8 = 1
	RtmpFrameTypeInter uint8 = 2

	RtmpCodecIdAvc  uint8 = 7
	RtmpCodecIdHevc uint8 = 12

	// RtmpAvcPacketTypeSeqHeader RtmpAvcPacketTypeNalu RtmpHevcPacketTypeSeqHeader RtmpHevcPacketTypeNalu
	// 注意，按照标准文档上描述，PacketType还有可能为2：
	// 2: AVC end of sequence (lower level NALU sequence ender is not required or supported)
	//
	// 我自己遇到过在流结尾时，对端发送 27 02 00 00 00的情况（比如我们的使用wontcry.flv的单元测试，最后一个包）
	//
	RtmpAvcPacketTypeSeqHeader  uint8 = 0
	RtmpAvcPacketTypeNalu       uint8 = 1
	RtmpHevcPacketTypeSeqHeader       = RtmpAvcPacketTypeSeqHeader
	RtmpHevcPacketTypeNalu            = RtmpAvcPacketTypeNalu

	RtmpAvcKeyFrame    = RtmpFrameTypeKey<<4 | RtmpCodecIdAvc
	RtmpHevcKeyFrame   = RtmpFrameTypeKey<<4 | RtmpCodecIdHevc
	RtmpAvcInterFrame  = RtmpFrameTypeInter<<4 | RtmpCodecIdAvc
	RtmpHevcInterFrame = RtmpFrameTypeInter<<4 | RtmpCodecIdHevc

	// RtmpSoundFormatAac spec-video_file_format_spec_v10.pdf
	// Audio tags
	//   AUDIODATA
	//     SoundFormat UB[4]
	//     SoundRate   UB[2]
	//     SoundSize   UB[1]
	//     SoundType   UB[1]
	//   AACAUDIODATA
	//     AACPacketType UI8
	//     Data          UI8[n]
	RtmpSoundFormatAac         uint8 = 10 // 注意，视频的CodecId是后4位，音频是前4位
	RtmpAacPacketTypeSeqHeader       = 0
	RtmpAacPacketTypeRaw             = 1
)

type RtmpHeader struct {
	Csid         int
	MsgLen       uint32 // 不包含header的大小
	MsgTypeId    uint8  // 8 audio 9 video 18 metadata
	MsgStreamId  int
	TimestampAbs uint32 // dts, 经过计算得到的流上的绝对时间戳，单位毫秒
}

type RtmpMsg struct {
	Header  RtmpHeader
	Payload []byte // Payload不包含Header内容。如果需要将RtmpMsg序列化成RTMP chunk，可调用rtmp.ChunkDivider相关的函数
}

func (msg RtmpMsg) IsAvcKeySeqHeader() bool {
	return msg.Header.MsgTypeId == RtmpTypeIdVideo && msg.Payload[0] == RtmpAvcKeyFrame && msg.Payload[1] == RtmpAvcPacketTypeSeqHeader
}

func (msg RtmpMsg) IsHevcKeySeqHeader() bool {
	return msg.Header.MsgTypeId == RtmpTypeIdVideo && msg.Payload[0] == RtmpHevcKeyFrame && msg.Payload[1] == RtmpHevcPacketTypeSeqHeader
}

func (msg RtmpMsg) IsVideoKeySeqHeader() bool {
	return msg.IsAvcKeySeqHeader() || msg.IsHevcKeySeqHeader()
}

func (msg RtmpMsg) IsAvcKeyNalu() bool {
	return msg.Header.MsgTypeId == RtmpTypeIdVideo && msg.Payload[0] == RtmpAvcKeyFrame && msg.Payload[1] == RtmpAvcPacketTypeNalu
}

func (msg RtmpMsg) IsHevcKeyNalu() bool {
	return msg.Header.MsgTypeId == RtmpTypeIdVideo && msg.Payload[0] == RtmpHevcKeyFrame && msg.Payload[1] == RtmpHevcPacketTypeNalu
}

func (msg RtmpMsg) IsVideoKeyNalu() bool {
	return msg.IsAvcKeyNalu() || msg.IsHevcKeyNalu()
}

func (msg RtmpMsg) IsAacSeqHeader() bool {
	return msg.Header.MsgTypeId == RtmpTypeIdAudio && (msg.Payload[0]>>4) == RtmpSoundFormatAac && msg.Payload[1] == RtmpAacPacketTypeSeqHeader
}

func (msg RtmpMsg) VideoCodecId() uint8 {
	return msg.Payload[0] & 0xF
}

func (msg RtmpMsg) Clone() (ret RtmpMsg) {
	ret.Header = msg.Header
	ret.Payload = make([]byte, len(msg.Payload))
	copy(ret.Payload, msg.Payload)
	return
}

func (msg RtmpMsg) Dts() uint32 {
	return msg.Header.TimestampAbs
}

// Pts
//
// 注意，只有视频才能调用该函数获取pts，音频的dts和pts都直接使用 RtmpMsg.Header.TimestampAbs
//
func (msg RtmpMsg) Pts() uint32 {
	return msg.Header.TimestampAbs + bele.BeUint24(msg.Payload[2:])
}
