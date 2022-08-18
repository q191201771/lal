// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtprtcp

import (
	"github.com/q191201771/lal/pkg/avc"
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/hevc"
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

const (
	RtpFixedHeaderLength = 12

	DefaultRtpVersion = 2
)

const (
	PositionTypeSingle    uint8 = 1
	PositionTypeFuaStart  uint8 = 2
	PositionTypeFuaMiddle uint8 = 3
	PositionTypeFuaEnd    uint8 = 4
	PositionTypeStapa     uint8 = 5 // 1个rtp包包含多个帧，目前供h264的stapa使用
	PositionTypeAp        uint8 = 6 // 1个rtp包包含多个帧，目前供h265的ap使用
)

type RtpHeader struct {
	Version    uint8  // 2b  *
	Padding    uint8  // 1b
	Extension  uint8  // 1
	CsrcCount  uint8  // 4b
	Mark       uint8  // 1b  *
	PacketType uint8  // 7b
	Seq        uint16 // 16b **
	Timestamp  uint32 // 32b **** samples
	Ssrc       uint32 // 32b **** Synchronization source

	payloadOffset uint32
}

type RtpPacket struct {
	Header RtpHeader
	Raw    []byte // 包含header内存

	positionType uint8
}

func (h *RtpHeader) PackTo(out []byte) {
	out[0] = h.CsrcCount | (h.Extension << 4) | (h.Padding << 5) | (h.Version << 6)
	out[1] = h.PacketType | (h.Mark << 7)
	bele.BePutUint16(out[2:], h.Seq)
	bele.BePutUint32(out[4:], h.Timestamp)
	bele.BePutUint32(out[8:], h.Ssrc)
}

func MakeDefaultRtpHeader() RtpHeader {
	return RtpHeader{
		Version:       DefaultRtpVersion,
		Padding:       0,
		Extension:     0,
		CsrcCount:     0,
		payloadOffset: RtpFixedHeaderLength,
	}
}

func MakeRtpPacket(h RtpHeader, payload []byte) (pkt RtpPacket) {
	pkt.Header = h
	pkt.Raw = make([]byte, RtpFixedHeaderLength+len(payload))
	pkt.Header.PackTo(pkt.Raw)
	copy(pkt.Raw[RtpFixedHeaderLength:], payload)
	return
}

func ParseRtpHeader(b []byte) (h RtpHeader, err error) {
	if len(b) < RtpFixedHeaderLength {
		err = base.ErrRtpRtcpShortBuffer
		return
	}

	h.Version = b[0] >> 6
	h.Padding = (b[0] >> 5) & 0x1
	h.Extension = (b[0] >> 4) & 0x1
	h.CsrcCount = b[0] & 0xF
	h.Mark = b[1] >> 7
	h.PacketType = b[1] & 0x7F
	h.Seq = bele.BeUint16(b[2:])
	h.Timestamp = bele.BeUint32(b[4:])
	h.Ssrc = bele.BeUint32(b[8:])

	h.payloadOffset = RtpFixedHeaderLength
	return
}

// ParseRtpPacket 函数调用结束后，不持有参数<b>的内存块
func ParseRtpPacket(b []byte) (pkt RtpPacket, err error) {
	pkt.Header, err = ParseRtpHeader(b)
	if err != nil {
		return
	}
	pkt.Raw = make([]byte, len(b))
	copy(pkt.Raw, b)
	return
}

func (p *RtpPacket) Body() []byte {
	if p.Header.payloadOffset == 0 {
		Log.Warnf("CHEFNOTICEME. payloadOffset=%d", p.Header.payloadOffset)
		p.Header.payloadOffset = RtpFixedHeaderLength
	}
	return p.Raw[p.Header.payloadOffset:]
}

// IsAvcHevcBoundary @param pt: 取值范围为AvPacketPtAvc或AvPacketPtHevc，否则直接返回false
//
func IsAvcHevcBoundary(pkt RtpPacket, pt base.AvPacketPt) bool {
	switch pt {
	case base.AvPacketPtAvc:
		return IsAvcBoundary(pkt)
	case base.AvPacketPtHevc:
		return IsHevcBoundary(pkt)
	}
	return false
}

func IsAvcBoundary(pkt RtpPacket) bool {
	boundaryNaluTypes := map[uint8]struct{}{
		avc.NaluTypeSps:      {},
		avc.NaluTypePps:      {},
		avc.NaluTypeIdrSlice: {},
	}

	b := pkt.Body()
	outerNaluType := avc.ParseNaluType(b[0])

	if _, ok := boundaryNaluTypes[outerNaluType]; ok {
		return true
	}

	if outerNaluType == NaluTypeAvcStapa {
		t := avc.ParseNaluType(b[3])
		if _, ok := boundaryNaluTypes[t]; ok {
			return true
		}
	}

	if outerNaluType == NaluTypeAvcFua {
		t := avc.ParseNaluType(b[1])
		if _, ok := boundaryNaluTypes[t]; ok {
			if b[1]&0x80 != 0 {
				return true
			}
		}
	}

	return false
}

func IsHevcBoundary(pkt RtpPacket) bool {
	boundaryNaluTypes := map[uint8]struct{}{
		hevc.NaluTypeVps:               {},
		hevc.NaluTypeSps:               {},
		hevc.NaluTypePps:               {},
		hevc.NaluTypeSliceBlaWlp:       {},
		hevc.NaluTypeSliceBlaWradl:     {},
		hevc.NaluTypeSliceBlaNlp:       {},
		hevc.NaluTypeSliceIdr:          {},
		hevc.NaluTypeSliceIdrNlp:       {},
		hevc.NaluTypeSliceCranut:       {},
		hevc.NaluTypeSliceRsvIrapVcl22: {},
		hevc.NaluTypeSliceRsvIrapVcl23: {},
	}

	b := pkt.Body()
	outerNaluType := hevc.ParseNaluType(b[0])

	if _, ok := boundaryNaluTypes[outerNaluType]; ok {
		return true
	}

	if outerNaluType == NaluTypeHevcFua {
		t := b[2] & 0x3F // 注意，这里是后6位，不是中间6位
		if _, ok := boundaryNaluTypes[t]; ok {
			if b[2]&0x80 != 0 {
				return true
			}
		}
	}

	return false
}
