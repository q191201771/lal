// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"encoding/hex"

	"github.com/q191201771/naza/pkg/nazalog"
)

// TODO chef: 这个模块叫Stream可能更合适

type Session struct {
	ssrc    uint32
	isAudio bool
}

func NewSession(ssrc uint32, isAudio bool) *Session {
	return &Session{
		ssrc:    ssrc,
		isAudio: isAudio,
	}
}

func (s *Session) FeedAVCPacket(pkt RTPPacket) {
	b := pkt.raw[pkt.header.payloadOffset:]
	// h264
	{
		// rfc3984 5.3.  NAL Unit Octet Usage
		//
		// +---------------+
		// |0|1|2|3|4|5|6|7|
		// +-+-+-+-+-+-+-+-+
		// |F|NRI|  Type   |
		// +---------------+

		outerNALUType := b[0] & 0x1F
		if outerNALUType <= NALUTypeSingleMax {
			nazalog.Debugf("SINGLE. naluType=%d %s", outerNALUType, hex.Dump(b[12:32]))
		} else if outerNALUType == NALUTypeFUA {

			// rfc3984 5.8.  Fragmentation Units (FUs)
			//
			// 0                   1                   2                   3
			// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
			// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			// | FU indicator  |   FU header   |                               |
			// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+                               |
			// |                                                               |
			// |                         FU payload                            |
			// |                                                               |
			// |                               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			// |                               :...OPTIONAL RTP padding        |
			// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
			//
			// FU indicator:
			// +---------------+
			// |0|1|2|3|4|5|6|7|
			// +-+-+-+-+-+-+-+-+
			// |F|NRI|  Type   |
			// +---------------+
			//
			// Fu header:
			// +---------------+
			// |0|1|2|3|4|5|6|7|
			// +-+-+-+-+-+-+-+-+
			// |S|E|R|  Type   |
			// +---------------+

			//fuIndicator := b[0]
			fuHeader := b[1]

			startCode := (fuHeader & 0x80) != 0
			endCode := (fuHeader & 0x40) != 0

			//naluType := (fuIndicator & 0xE0) | (fuHeader & 0x1F)
			naluType := fuHeader & 0x1F

			nazalog.Debugf("FUA. outerNALUType=%d, naluType=%d, startCode=%t, endCode=%t %s", outerNALUType, naluType, startCode, endCode, hex.Dump(b[0:16]))
		} else {
			nazalog.Errorf("error. type=%d", outerNALUType)
		}

		// TODO chef: to be continued
		// 从SDP中获取SPS，PPS等信息
		// 将RTP包合并出视频帧
		// 先做一个rtsp server，接收rtsp的流，录制成ES流吧
	}

	// h265
	//{
	//	originNALUType := (b[h.payloadOffset] >> 1) & 0x3F
	//	if originNALUType == 49 {
	//		header2 := b[h.payloadOffset+2]
	//
	//		startCode := (header2 & 0x80) != 0
	//		endCode := (header2 & 0x40) != 0
	//
	//		naluType := header2 & 0x3F
	//
	//		nazalog.Debugf("FUA. originNALUType=%d, naluType=%d, startCode=%t, endCode=%t %s", originNALUType, naluType, startCode, endCode, hex.Dump(b[12:32]))
	//
	//	} else {
	//		nazalog.Debugf("SINGLE. naluType=%d %s", originNALUType, hex.Dump(b[12:32]))
	//	}
	//}
}

func (s *Session) FeedAACPacket(pkt RTPPacket) {
	return
	// TODO chef: 目前只实现了AAC MPEG4-GENERIC/44100/2

	/*
		// rfc3640 2.11.  Global Structure of Payload Format
		//
		// +---------+-----------+-----------+---------------+
		// | RTP     | AU Header | Auxiliary | Access Unit   |
		// | Header  | Section   | Section   | Data Section  |
		// +---------+-----------+-----------+---------------+
		//
		//           <----------RTP Packet Payload----------->
		//
		// rfc3640 3.2.1.  The AU Header Section
		//
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+- .. -+-+-+-+-+-+-+-+-+-+
		// |AU-headers-length|AU-header|AU-header|      |AU-header|padding|
		// |                 |   (1)   |   (2)   |      |   (n)   | bits  |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+- .. -+-+-+-+-+-+-+-+-+-+
		//
		// rfc3640 3.3.6.  High Bit-rate AAC
		//

		nazalog.Debugf("%s", hex.Dump(b[12:]))

		// au header section
		var auHeaderLength uint32
		auHeaderLength = uint32(b[h.payloadOffset])<<8 + uint32(b[h.payloadOffset+1])
		auHeaderLength = (auHeaderLength + 7) / 8
		nazalog.Debugf("auHeaderLength=%d", auHeaderLength)

		// no auxiliary section

		pauh := h.payloadOffset + uint32(2)                 // au header pos
		pau := h.payloadOffset + uint32(2) + auHeaderLength // au pos
		auNum := uint32(auHeaderLength) / 2
		for i := uint32(0); i < auNum; i++ {
			var auSize uint32
			auSize = uint32(b[pauh])<<8 | uint32(b[pauh+1]&0xF8) // 13bit
			auSize /= 8

			auIndex := b[pauh+1] & 0x7

			// data
			// pau, auSize
			nazalog.Debugf("%d %d %s", auSize, auIndex, hex.Dump(b[pau:pau+auSize]))

			pauh += 2
			pau += auSize
		}
	*/
}
