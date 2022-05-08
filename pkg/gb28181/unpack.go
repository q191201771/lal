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

	"github.com/q191201771/lal/pkg/hevc"

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
	psPackStartCodeHikStream        = 0x01bd

	psPackStartCodePesPsd      = 0xff // program_stream_directory
	psPackStartCodePesPrivate1 = 0xbd // private_stream_1
	psPackStartCodePesPadding  = 0xbe // padding_stream
	psPackStartCodePesPrivate2 = 0xbf // padding_stream_2
	psPackStartCodePesEcm      = 0xf0 // ECM_stream
	psPackStartCodePesEmm      = 0xf1 // EMM_stream

	psPackStartCodePackEnd = 0xb9
)
const (
	StreamTypeH264 = 0x1b
	StreamTypeH265 = 0x24
	StreamTypeAAC  = 0x0f
	StreamG711A    = 0x90 //PCMA
	StreamG7221    = 0x92
	StreamG7231    = 0x93
	StreamG729     = 0x99
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

	videoStreamType byte
	audioStreamType byte

	preVideoPts uint64
	preAudioPts uint64
	preVideoDts uint64
	preAudioDts uint64

	videoBuf []byte

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

func (p *PsUnpacker) VideoStreamType() byte {
	return p.videoStreamType
}

func (p *PsUnpacker) AudioStreamType() byte {
	return p.audioStreamType
}

// FeedRtpPacket
//
// @param rtpPacket: rtp包，注意，包含rtp包头部分
//
func (p *PsUnpacker) FeedRtpPacket(rtpPacket []byte, rtpTimestamp uint32) {
	nazalog.Debugf("> FeedRtpPacket. len=%d", len(rtpPacket))

	// skip rtp header
	p.FeedRtpBody(rtpPacket[12:], rtpTimestamp)
}

//pes 没有pts时使用外部传入，rtpBody 需要传入两个ps header之间的数据
func (p *PsUnpacker) FeedRtpBody(rtpBody []byte, rtpTimestamp uint32) {
	p.buf.Reset()
	p.buf.Write(rtpBody)
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
			if len(rb) <= i {
				return
			}
			// skip stuffing
			l := int(rb[i] & 0x7)
			i += 1 + l
			p.buf.Skip(i)
		case psPackStartCodeSystemHeader:
			nazalog.Debugf("-----system header-----")
			// skip
			p.SkipPackStreamBody(rb, i)
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
				if streamId >= 0xe0 && streamId <= 0xef {
					p.videoStreamType = streamType
				} else if streamId >= 0xc0 && streamId <= 0xdf {
					p.audioStreamType = streamType
				}
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
			fallthrough
		case psPackStartCodeVideoStream:
			//nazalog.Debugf("-----video stream-----")
			length := int(bele.BeUint16(rb[i:]))
			i += 2
			//nazalog.Debugf("CHEFERASEME %d %d %d", p.buf.Len(), len(rtpBody), length)

			if len(rb)-i < length {
				return
			}

			ptsDtsFlag := rb[i+1] >> 6
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
			if code == psPackStartCodeAudioStream {
				if pts == 0 {
					pts = uint64(rtpTimestamp)
				}
				if dts == 0 {
					dts = pts
				}
				p.preAudioPts = pts
				p.preAudioDts = dts
				if p.videoStreamType == StreamTypeAAC {
					nazalog.Debugf("audio code=%d, length=%d, ptsDtsFlag=%d, phdl=%d, pts=%d, dts=%d,type=%d", code, length, ptsDtsFlag, phdl, pts, dts, p.audioStreamType)
					if p.onAudio != nil {
						p.onAudio(rb[i:i+length-3-phdl], int64(dts), int64(pts))
					}
				}

			} else {
				if pts == 0 {
					if p.videoBuf == nil {
						pts = uint64(rtpTimestamp)
					} else {
						pts = p.preVideoPts
					}
				}
				if dts == 0 {
					dts = pts
				}
				//根据时间戳来判断是否是相同一帧，不同时间戳处理前一帧
				if p.preVideoPts != 0 {
					if p.preVideoPts != pts {
						if p.onVideo != nil {
							leading, preLeading := 0, 0
							startPos, preLeading := p.findNextNaluStartPos(p.videoBuf, 0)
							if startPos >= 0 {
								nextPos := startPos
								buf := p.videoBuf[:0]
								for startPos >= 0 {
									nextPos, leading = p.findNextNaluStartPos(p.videoBuf, startPos+3)
									if nextPos >= 0 {
										buf = p.videoBuf[startPos:nextPos]
									} else {
										buf = p.videoBuf[startPos:]
									}
									startPos = nextPos
									if p.videoStreamType == StreamTypeH265 {
										nazalog.Debugf("Video code=%d, length=%d,pts=%d, dts=%d, type=%s", code, len(buf), p.preVideoPts, p.preVideoDts, hevc.ParseNaluTypeReadable(buf[preLeading+1]))

									} else {
										nazalog.Debugf("Video code=%d, length=%d,pts=%d, dts=%d, type=%s", code, len(buf), p.preVideoPts, p.preVideoDts, avc.ParseNaluTypeReadable(buf[preLeading+1]))
									}
									p.onVideo(buf, int64(p.preVideoDts), int64(p.preVideoPts))
									if nextPos >= 0 {
										preLeading = leading
									}
								}

							}

							p.videoBuf = nil
						}
					}
				}
				p.preVideoPts = pts
				p.preVideoDts = dts
				//暂存当前帧
				p.videoBuf = append(p.videoBuf, rb[i:i+length-3-phdl]...)
			}

			p.buf.Skip(6 + length)
		case psPackStartCodeHikStream:
			p.SkipPackStreamBody(rb, i)
		case psPackStartCodePesPrivate2:
			fallthrough
		case psPackStartCodePesEcm:
			fallthrough
		case psPackStartCodePesEmm:
			fallthrough
		case psPackStartCodePesPadding:
			fallthrough
		case psPackStartCodePackEnd:
			p.buf.Skip(i)
		case psPackStartCodePesPsd:
			p.SkipPackStreamBody(rb, i)
		default:
			nazalog.Errorf("default %s", hex.Dump(nazabytes.Prefix(rb[i-4:], 32)))
			p.SkipPackStreamBody(rb, i)
			return

		}
	}
}
func (p *PsUnpacker) SkipPackStreamBody(rb []byte, indexId int) {
	if len(rb[:]) < indexId+2 {
		return
	}
	l := int(bele.BeUint16(rb[indexId:]))
	if len(rb[:]) < indexId+2+l {
		return
	}
	p.buf.Skip(indexId + 2 + l)
}

//参考media-server 中的ps解析
func (p *PsUnpacker) findNextNaluStartPos(buf []byte, index int) (startPos int, leading int) {
	bufLen := len(buf)
	startPos = -1
	leading = 0
	var pos int
	for i := index; i+2 < bufLen; i += pos {
		if pos, leading = p.findNaluStartPos(buf[i:]); pos < 0 {
			return
		}
		if p.videoStreamType == StreamTypeH265 {
			nalType := (buf[i+pos] >> 1) & 0x3f
			if bufLen > i+pos+2 {
				if hevcNalu(nalType, buf[i+pos:]) {
					startPos = i + pos - leading - 1
					return
				}
			}
		} else {
			nalType := buf[i+pos] & 0x1f
			if bufLen > i+pos+1 {
				if avcNalu(nalType, buf[i+pos:]) {
					startPos = i + pos - leading - 1
					return
				}
			}
		}

	}
	return
}
func (p *PsUnpacker) findNaluStartPos(buf []byte) (pos int, leading int) {
	bufLen := len(buf)
	zeros := 0
	pos = -1
	for i := 0; i+1 < bufLen; i++ {
		if buf[i] == 1 {
			if zeros > 2 {
				leading = 3
				pos = i + 1
				break
			} else if zeros == 2 {
				leading = 2
				pos = i + 1
				break
			}
		}
		if buf[i] == 0 {
			zeros += 1
		} else {
			zeros = 0
		}
	}
	return
}

func avcNalu(nalType byte, nalu []byte) bool {
	switch nalType {
	case avc.NaluTypeSlice:
		fallthrough
	case avc.NaluTypePartition_A:
		fallthrough
	case avc.NaluTypeIdrSlice:
		if nalu[1]&0x80 == 0 {
			return false
		} else {
			return true
		}
	case avc.NaluTypeSei:
		fallthrough
	case avc.NaluTypeSps:
		fallthrough
	case avc.NaluTypePps:
		fallthrough
	case avc.NaluTypeAud:
		return true
	default:
		if nalType > 14 && nalType < 18 {
			return true
		}
	}
	return false
}
func hevcNalu(nalType byte, nalu []byte) bool {
	nuhLayerId := ((nalType & 0x01) << 5) | ((nalu[1] >> 3) & 0x1F)
	if nalType == hevc.NaluTypeVps ||
		nalType == hevc.NaluTypeSps ||
		nalType == hevc.NaluTypePps ||
		(nuhLayerId == 0 && (nalType == hevc.NaluTypeAud ||
			nalType == hevc.NaluTypeSei ||
			nalType >= 41 && nalType <= 44 ||
			nalType >= 48 && nalType <= 55)) {
		return true
	} else if nalType <= 31 {
		if nalu[2]&0x80 == 0 {
			return false
		} else {
			return true
		}
	}

	return false
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
