// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package gb28181

import (
	"encoding/hex"
	"github.com/q191201771/lal/pkg/avc"
	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazabits"
	"github.com/q191201771/naza/pkg/nazabytes"
	"github.com/q191201771/naza/pkg/nazalog"
)

// TODO(chef): gb28181 package处于开发中阶段，请不使用

const (
	psPackStartCodePackHeader       = 0x01ba
	psPackStartCodeSystemHeader     = 0x01bb
	psPackStartCodeProgramStreamMap = 0x01bc
	psPackStartCodeAudioStream      = 0x01c0
	psPackStartCodeVideoStream      = 0x01e0
)

type onAudioFn func(payload []byte, dts int64, pts int64)
type onVideoFn func(payload []byte, dts int64, pts int64)

type IPsUnpackerObserver interface {
	OnAudio(payload []byte, dts int64, pts int64)

	// OnVideo
	//
	// @param payload: annexb格式
	//
	OnVideo(payload []byte, dts int64, pts int64)
}

// PsUnpacker 解析ps(Progream Stream)流
//
type PsUnpacker struct {
	buf *nazabytes.Buffer

	onAudio onAudioFn
	onVideo onVideoFn
}

func NewPsUnpacker() *PsUnpacker {
	return &PsUnpacker{
		buf: nazabytes.NewBuffer(4096),
	}
}

func (p *PsUnpacker) WithObserver(obs IPsUnpackerObserver) *PsUnpacker {
	p.onAudio = obs.OnAudio
	p.onVideo = obs.OnVideo
	return p
}

func (p *PsUnpacker) WithCallbackFunc(onAudio onAudioFn, onVideo onVideoFn) *PsUnpacker {
	p.onAudio = onAudio
	p.onVideo = onVideo
	return p
}

// FeedRtpPacket
//
// @param rtpPacket: rtp包，注意，包含rtp包头部分
//
func (p *PsUnpacker) FeedRtpPacket(rtpPacket []byte) {
	nazalog.Debugf("> FeedRtpPacket. len=%d", len(rtpPacket))

	// skip rtp header
	p.FeedRtpBody(rtpPacket[12:])
}

func (p *PsUnpacker) FeedRtpBody(rtpBody []byte) {
	_, _ = p.buf.Write(rtpBody)

	// ISO/IEC iso13818-1
	//
	// 2.5 Program Stream bitstream requirements
	//
	// 2.5.3.3 Pack layer of Program Stream
	// Table 2-33 - Program Stream pack header
	//
	// 2.5.3.5 System header
	// Table 2-32 - Program Stream system header
	//
	// 2.5.4 Program Stream map
	// Table 2-35 - Program Stream map
	//
	// TODO(chef): [fix] 有些没做有效长度判断
	for p.buf.Len() != 0 {
		rb := p.buf.Bytes()
		i := 0
		code := bele.BeUint32(rb[i:])
		i += 4
		switch code {
		case psPackStartCodePackHeader:
			nazalog.Debugf("-----pack header-----")
			// skip system clock reference(SCR)
			// skip PES program mux rate
			i += 6 + 3
			// skip stuffing
			l := int(rb[i] & 0x7)
			i += 1 + l
			p.buf.Skip(i)
		case psPackStartCodeSystemHeader:
			nazalog.Debugf("-----system header-----")
			// skip
			l := int(bele.BeUint16(rb[i:]))
			p.buf.Skip(i + 2 + l)
		case psPackStartCodeProgramStreamMap:
			nazalog.Debugf("-----program stream map-----")

			if len(rb[i:]) < 6 {
				return
			}

			// skip program_stream_map_length
			// skip current_next_indicator
			// skip reserved
			// skip program_stream_map_version
			// skip reverved
			// skip marked_bit
			i += 4 // 2 + 1 + 1

			// program_stream_info_length
			l := int(bele.BeUint16(rb[i:]))
			i += 2

			if len(rb[i:]) < l+2 {
				return
			}

			i += l

			// elementary_stream_map_length
			esml := int(bele.BeUint16(rb[i:]))
			nazalog.Debugf("l=%d, esml=%d", l, esml)
			i += 2

			if len(rb[i:]) < esml+4 {
				return
			}

			for esml > 0 {
				streamType := rb[i]
				i += 1
				streamId := rb[i]
				i += 1
				// elementary_stream_info_length
				esil := int(bele.BeUint16(rb[i:]))
				nazalog.Debugf("streamType=%d, streamId=%d, esil=%d", streamType, streamId, esil)
				i += 2 + esil
				esml = esml - 4 - esil
			}
			// skip
			i += 4
			nazalog.Debugf("CHEFERASEME i=%d, esml=%d, %s", i, esml, hex.Dump(nazabytes.Prefix(p.buf.Bytes(), 128)))
			p.buf.Skip(i)
		case psPackStartCodeAudioStream:
			nazalog.Debugf("-----audio stream-----")
		case psPackStartCodeVideoStream:
			nazalog.Debugf("-----video stream-----")
			length := int(bele.BeUint16(rb[i:]))
			i += 2
			//nazalog.Debugf("CHEFERASEME %d %d %d", p.buf.Len(), len(rtpBody), length)

			if len(rb)-6 < length {
				return
			}

			ptsDtsFlag := rb[i+1] & 0x0c
			phdl := int(rb[i+2])
			i += 3

			var pts uint64
			var dts uint64
			j := 0
			if ptsDtsFlag&0x2 != 0 {
				_, pts = readPts(rb[i:])
				j += 5
			}
			if ptsDtsFlag&0x1 != 0 {
				_, dts = readPts(rtpBody[i+j:])
			} else {
				dts = pts
			}

			i += phdl

			nazalog.Debugf("code=%d, length=%d, ptsDtsFlag=%d, phdl=%d, pts=%d, dts=%d, type=%s", code, length, ptsDtsFlag, phdl, pts, dts, avc.ParseNaluTypeReadable(rb[i+4]))
			if p.onVideo != nil {
				p.onVideo(rb[i:i+length-3-phdl], int64(dts), int64(pts))
			}

			p.buf.Skip(6 + length)
		default:
			nazalog.Errorf("%s", hex.Dump(nazabytes.Prefix(rb, 32)))
			return
		}
	}
}

// ---------------------------------------------------------------------------------------------------------------------

// TODO(chef): [refactor] 以下代码拷贝来自package mpegts，重复了

// Pes -----------------------------------------------------------
// <iso13818-1.pdf>
// <2.4.3.6 PES packet> <page 49/174>
// <Table E.1 - PES packet header example> <page 142/174>
// <F.0.2 PES packet> <page 144/174>
// packet_start_code_prefix  [24b] *** always 0x00, 0x00, 0x01
// stream_id                 [8b]  *
// PES_packet_length         [16b] **
// '10'                      [2b]
// PES_scrambling_control    [2b]
// PES_priority              [1b]
// data_alignment_indicator  [1b]
// copyright                 [1b]
// original_or_copy          [1b]  *
// PTS_DTS_flags             [2b]
// ESCR_flag                 [1b]
// ES_rate_flag              [1b]
// DSM_trick_mode_flag       [1b]
// additional_copy_info_flag [1b]
// PES_CRC_flag              [1b]
// PES_extension_flag        [1b]  *
// PES_header_data_length    [8b]  *
// -----------------------------------------------------------
type Pes struct {
	pscp       uint32
	sid        uint8
	ppl        uint16
	pad1       uint8
	ptsDtsFlag uint8
	pad2       uint8
	phdl       uint8
	pts        uint64
	dts        uint64
}

func ParsePes(b []byte) (pes Pes, length int) {
	br := nazabits.NewBitReader(b)
	//pes.pscp, _ = br.ReadBits32(24)
	//pes.sid, _ = br.ReadBits8(8)
	//pes.ppl, _ = br.ReadBits16(16)

	pes.pad1, _ = br.ReadBits8(8)
	pes.ptsDtsFlag, _ = br.ReadBits8(2)
	pes.pad2, _ = br.ReadBits8(6)
	pes.phdl, _ = br.ReadBits8(8)

	_, _ = br.ReadBytes(uint(pes.phdl))
	length = 9 + int(pes.phdl)

	// 处理得不是特别标准
	if pes.ptsDtsFlag&0x2 != 0 {
		_, pes.pts = readPts(b[9:])
	}
	if pes.ptsDtsFlag&0x1 != 0 {
		_, pes.dts = readPts(b[14:])
	} else {
		pes.dts = pes.pts
	}
	//pes.pts = (pes.pts - delay) / 90
	//pes.dts = (pes.dts - delay) / 90

	return
}

// read pts or dts
func readPts(b []byte) (fb uint8, pts uint64) {
	fb = b[0] >> 4
	pts |= uint64((b[0]>>1)&0x07) << 30
	pts |= (uint64(b[1])<<8 | uint64(b[2])) >> 1 << 15
	pts |= (uint64(b[3])<<8 | uint64(b[4])) >> 1
	return
}
