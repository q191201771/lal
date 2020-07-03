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

// rfc3550 5.1 RTP Fixed Header Fields
type RTPHeader struct {
	version    uint8  // 2b
	padding    uint8  // 1b
	extension  uint8  // 1
	csrcCount  uint8  // 4b
	mark       uint8  // 1b
	packetType uint8  // 7b
	seq        uint16 // 16b
	timestamp  uint32 // 32b
	ssrc       uint32 // 32b
}

func parseRTPPacket(b []byte) {
	var h RTPHeader
	h.version = b[0] >> 6
	h.padding = (b[0] >> 5) & 0x1
	h.extension = (b[0] >> 4) & 0x1
	h.csrcCount = b[0] & 0xF
	h.mark = b[1] >> 7
	h.packetType = b[1] & 0x7F
	h.seq = bele.BEUint16(b[2:])
	h.timestamp = bele.BEUint32(b[4:])
	h.ssrc = bele.BEUint32(b[8:])
	nazalog.Debugf("%+v", h)
}
