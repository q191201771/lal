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

// -------------------------------------------
// rfc3550 6.4.1 SR: Sender Report RTCP Packet
// -------------------------------------------
//
//        0                   1                   2                   3
//        0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// header |V=2|P|    RC   |   PT=SR=200   |             length            |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        |                         SSRC of sender                        |
//        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
// sender |              NTP timestamp, most significant word             |
// info   +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        |             NTP timestamp, least significant word             |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        |                         RTP timestamp                         |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        |                     sender's packet count                     |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        |                      sender's octet count                     |
//        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
// report |                 SSRC_1 (SSRC of first source)                 |
// block  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   1    | fraction lost |       cumulative number of packets lost       |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        |           extended highest sequence number received           |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        |                      interarrival jitter                      |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        |                         last SR (LSR)                         |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        |                   delay since last SR (DLSR)                  |
//        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
// report |                 SSRC_2 (SSRC of second source)                |
// block  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   2    :                               ...                             :
//        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
//        |                  profile-specific extensions                  |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

// ---------------------------------------------
// rfc3550 6.4.2 RR: Receiver Report RTCP Packet
// ---------------------------------------------
//
//        0                   1                   2                   3
//        0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// header |V=2|P|    RC   |   PT=RR=201   |             length            |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        |                     SSRC of packet sender                     |
//        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
// report |                 SSRC_1 (SSRC of first source)                 |
// block  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   1    | fraction lost |       cumulative number of packets lost       |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        |           extended highest sequence number received           |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        |                      interarrival jitter                      |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        |                         last SR (LSR)                         |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//        |                   delay since last SR (DLSR)                  |
//        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
// report |                 SSRC_2 (SSRC of second source)                |
// block  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//   2    :                               ...                             :
//        +=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+=+
//        |                  profile-specific extensions                  |
//        +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

var ErrRTCP = errors.New("lal.rtcp: fxxk")

const (
	RTCPPacketTypeSR = 200 // Sender Report
	RTCPPacketTypeRR = 201 // Receiver Report

	RTCPVersion = 2
)

type RTCPHeader struct {
	Version       uint8  // 2b
	Padding       uint8  // 1b
	CountOrFormat uint8  // 5b
	PacketType    uint8  // 8b
	Length        uint16 // 16b, byte length = (length+1) * 4
}

type SR struct {
	SenderSSRC uint32
	MSW        uint32 // NTP timestamp, most significant word
	LSW        uint32 // NTP timestamp, least significant word
	Timestamp  uint32
	PktCnt     uint32
	OctetCnt   uint32
}

func ParseRTCPHeader(b []byte) RTCPHeader {
	var h RTCPHeader
	h.Version = b[0] >> 6
	h.Padding = (b[0] >> 5) & 0x1
	h.CountOrFormat = b[0] & 0x1F
	h.PacketType = b[1]
	h.Length = bele.BEUint16(b[2:])
	return h
}

// rfc3550 6.4.1
//
// @param b rtcp包，包含包头
func ParseSR(b []byte) SR {
	var s SR
	s.SenderSSRC = bele.BEUint32(b[4:])
	s.MSW = bele.BEUint32(b[8:])
	s.LSW = bele.BEUint32(b[12:])
	s.Timestamp = bele.BEUint32(b[16:])
	s.PktCnt = bele.BEUint32(b[20:])
	s.OctetCnt = bele.BEUint32(b[24:])
	return s
}

// @param out 传出参数，注意，调用方保证长度>=4
func (r *RTCPHeader) PackTo(out []byte) {
	out[0] = r.Version<<6 | r.Padding<<5 | r.CountOrFormat
	out[1] = r.PacketType
	bele.BEPutUint16(out[2:], r.Length)
}

func (s *SR) GetMiddleNTP() uint32 {
	return uint32(((uint64(s.MSW)<<32 | uint64(s.LSW)) << 16) >> 32)
}
