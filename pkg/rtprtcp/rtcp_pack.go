// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtprtcp

import "github.com/q191201771/naza/pkg/bele"

type RR struct {
	senderSSRC  uint32
	mediaSSRC   uint32
	fraction    uint8
	lost        uint32
	cycles      uint16
	extendedSeq uint32
	jitter      uint32
	lsr         uint32
	dlsr        uint32 // default 0
}

func (r *RR) Pack() []byte {
	const lenInWords = 8

	b := make([]byte, lenInWords*4)

	var h RTCPHeader
	h.Version = RTCPVersion
	h.Padding = 0
	h.CountOrFormat = 1
	h.PacketType = RTCPPacketTypeRR
	h.Length = lenInWords - 1
	h.PackTo(b)

	bele.BEPutUint32(b[4:], r.senderSSRC)
	bele.BEPutUint32(b[8:], r.mediaSSRC)
	b[12] = r.fraction
	// TODO chef: lost的表示是否正确
	bele.BEPutUint24(b[13:], r.lost>>8)
	bele.BEPutUint16(b[16:], r.cycles)
	bele.BEPutUint32(b[18:], r.extendedSeq)
	bele.BEPutUint32(b[20:], r.jitter)
	bele.BEPutUint32(b[24:], r.lsr)
	bele.BEPutUint32(b[28:], 0)

	return b
}
