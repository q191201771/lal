// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

type AvPacketPt int

const (
	AvPacketPtUnknown AvPacketPt = -1
	AvPacketPtAvc     AvPacketPt = 96 // h264
	AvPacketPtHevc    AvPacketPt = 98 // h265
	AvPacketPtAac     AvPacketPt = 97
)

// AvPacket
//
// 不同场景使用时，字段含义可能不同。
// 使用AvPacket的地方，应注明各字段的含义。
//
type AvPacket struct {
	PayloadType AvPacketPt
	Timestamp   uint32 // TODO(chef): 改成int64
	Payload     []byte
}

func (a AvPacketPt) ReadableString() string {
	switch a {
	case AvPacketPtUnknown:
		return "unknown"
	case AvPacketPtAvc:
		return "avc"
	case AvPacketPtHevc:
		return "hevc"
	case AvPacketPtAac:
		return "aac"
	}
	return ""
}

func (packet AvPacket) IsAudio() bool {
	return packet.PayloadType == AvPacketPtAac
}

func (packet AvPacket) IsVideo() bool {
	return packet.PayloadType == AvPacketPtAvc || packet.PayloadType == AvPacketPtHevc
}
