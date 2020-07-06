// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazalog"
)

// rfc3550

const (
	PacketTypeSR = 200 // Sender Report
)

type RTCPHeader struct {
	version       uint8  // 2b
	padding       uint8  // 1b
	countOrFormat uint8  // 5b
	packetType    uint8  // 8b
	length        uint16 // 16b, byte length = (length+1) * 4
}

type SR struct {
	senderSSRC uint32
	msw        uint32
	lsw        uint32
	ts         uint32
	pktCnt     uint32
	octetCnt   uint32
}

func parseRTCPPacket(b []byte) {
	var h RTCPHeader
	h.version = b[0] >> 6
	h.padding = (b[0] >> 5) & 0x1
	h.countOrFormat = b[0] & 0x1F
	h.packetType = b[1]
	h.length = bele.BEUint16(b[2:])
	nazalog.Debugf("%+v", h)

	switch h.packetType {
	case PacketTypeSR:
		parseSR(b)
	default:
		nazalog.Warnf("unknown packet type. type=%d", h.packetType)
	}
}

// rfc3550 6.4.1
func parseSR(b []byte) {
	var s SR
	s.senderSSRC = bele.BEUint32(b[4:])
	s.msw = bele.BEUint32(b[8:])
	s.lsw = bele.BEUint32(b[12:])
	s.ts = bele.BEUint32(b[16:])
	s.pktCnt = bele.BEUint32(b[20:])
	s.octetCnt = bele.BEUint32(b[24:])
	nazalog.Debugf("%+v", s)
}
