// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtprtcp

import (
	"errors"

	"github.com/q191201771/naza/pkg/bele"
)

// -----------------------------------
// rfc3550 5.1 RTP Fixed Header Fields
// -----------------------------------
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |V=2|P|X|  CC   |M|     PT      |       sequence number         |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                           timestamp                           |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |           synchronization source (SSRC) identifier            |
// +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
// |            contributing source (CSRC) identifiers             |
// |                             ....                              |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

var ErrRTP = errors.New("lal.rtp: fxxk")

const (
	RTPFixedHeaderLength = 12
)

// rfc3984 5.2.  Common Structure of the RTP Payload Format
// Table 1.  Summary of NAL unit types and their payload structures
//
// Type   Packet    Type name                        Section
// ---------------------------------------------------------
// 0      undefined                                    -
// 1-23   NAL unit  Single NAL unit packet per H.264   5.6
// 24     STAP-A    Single-time aggregation packet     5.7.1
// 25     STAP-B    Single-time aggregation packet     5.7.1
// 26     MTAP16    Multi-time aggregation packet      5.7.2
// 27     MTAP24    Multi-time aggregation packet      5.7.2
// 28     FU-A      Fragmentation unit                 5.8
// 29     FU-B      Fragmentation unit                 5.8
// 30-31  undefined                                    -

const (
	NALUTypeSingleMax = 23
	NALUTypeFUA       = 28
)

const (
	//PositionUnknown uint8 = 0
	PositionTypeSingle      uint8 = 1
	PositionTypeMultiStart  uint8 = 2
	PositionTypeMultiMiddle uint8 = 3
	PositionTypeMultiEnd    uint8 = 4
)

type RTPHeader struct {
	Version    uint8  // 2b  *
	Padding    uint8  // 1b
	Extension  uint8  // 1
	CsrcCount  uint8  // 4b
	Mark       uint8  // 1b  *
	PacketType uint8  // 7b
	Seq        uint16 // 16b **
	Timestamp  uint32 // 32b ****
	Ssrc       uint32 // 32b **** Synchronization source

	payloadOffset uint32
}

type RTPPacket struct {
	Header RTPHeader
	Raw    []byte // 包含header内存

	positionType uint8
}

func ParseRTPPacket(b []byte) (h RTPHeader, err error) {
	if len(b) < RTPFixedHeaderLength {
		err = ErrRTP
		return
	}

	h.Version = b[0] >> 6
	h.Padding = (b[0] >> 5) & 0x1
	h.Extension = (b[0] >> 4) & 0x1
	h.CsrcCount = b[0] & 0xF
	h.Mark = b[1] >> 7
	h.PacketType = b[1] & 0x7F
	h.Seq = bele.BEUint16(b[2:])
	h.Timestamp = bele.BEUint32(b[4:])
	h.Ssrc = bele.BEUint32(b[8:])

	h.payloadOffset = RTPFixedHeaderLength
	return
}

// 比较序号的值，内部处理序号翻转问题，见单元测试中的例子
func CompareSeq(a, b uint16) int {
	if a == b {
		return 0
	}
	if a > b {
		if a-b < 16384 {
			return 1
		}

		return -1
	}

	// must be a < b
	if b-a < 16384 {
		return -1
	}

	return 1
}

// a减b的值，内部处理序号翻转问题，如果a小于b，则返回负值，见单元测试中的例子
func SubSeq(a, b uint16) int {
	if a == b {
		return 0
	}

	if a > b {
		d := a - b
		if d < 16384 {
			return int(d)
		}
		return int(d) - 65536
	}

	// must be a < b
	d := b - a
	if d < 16384 {
		return -int(d)
	}

	return 65536 - int(d)
}
