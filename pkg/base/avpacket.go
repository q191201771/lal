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
	AvPacketPtAvc     AvPacketPt = RtpPacketTypeAvcOrHevc
	AvPacketPtHevc    AvPacketPt = RtpPacketTypeHevc
	AvPacketPtAac     AvPacketPt = RtpPacketTypeAac
)

// 不同场景使用时，字段含义可能不同。
// 使用AvPacket的地方，应注明各字段的含义。
type AvPacket struct {
	Timestamp   uint32
	PayloadType AvPacketPt
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
