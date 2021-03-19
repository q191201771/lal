// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtprtcp

import (
	"github.com/cfeeling/lal/pkg/hevc"
	"github.com/q191201771/naza/pkg/nazalog"
)

func calcPositionIfNeededHEVC(pkt *RTPPacket) {
	b := pkt.Raw[pkt.Header.payloadOffset:]

	// +---------------+---------------+
	// |0|1|2|3|4|5|6|7|0|1|2|3|4|5|6|7|
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
	// |F|   Type    |  LayerId  | TID |
	// +-------------+-----------------+

	outerNALUType := hevc.ParseNALUType(b[0])

	switch outerNALUType {
	case hevc.NALUTypeVPS:
		fallthrough
	case hevc.NALUTypeSPS:
		fallthrough
	case hevc.NALUTypePPS:
		fallthrough
	case hevc.NALUTypeSEI:
		fallthrough
	case hevc.NALUTypeSliceTrailR:
		fallthrough
	case hevc.NALUTypeSliceIDRNLP:
		pkt.positionType = PositionTypeSingle
		return
	case NALUTypeHEVCFUA:
		// Figure 1: The Structure of the HEVC NAL Unit Header

		// 0                   1                   2                   3
		// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |    PayloadHdr (Type=49)       |   FU header   | DONL (cond)   |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-|
		// | DONL (cond)   |                                               |
		// |-+-+-+-+-+-+-+-+                                               |
		// |                         FU payload                            |
		// |                                                               |
		// |                               +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
		// |                               :...OPTIONAL RTP padding        |
		// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

		// Figure 9: The Structure of an FU

		// +---------------+
		// |0|1|2|3|4|5|6|7|
		// +-+-+-+-+-+-+-+-+
		// |S|E|  FuType   |
		// +---------------+

		// Figure 10: The Structure of FU Header

		startCode := (b[2] & 0x80) != 0
		endCode := (b[2] & 0x40) != 0

		if startCode {
			pkt.positionType = PositionTypeFUAStart
			return
		}

		if endCode {
			pkt.positionType = PositionTypeFUAEnd
			return
		}

		pkt.positionType = PositionTypeFUAMiddle
		return
	default:
		// TODO chef: 没有实现 AP 48
		nazalog.Errorf("unknown nalu type. outerNALUType=%d", outerNALUType)
	}

}

// hevc rtp包合帧部分见func unpackOneAVCOrHEVC
