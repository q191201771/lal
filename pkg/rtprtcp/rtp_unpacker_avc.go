// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtprtcp

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazalog"
)

func calcPositionIfNeededAVC(pkt *RTPPacket) {
	b := pkt.Raw[pkt.Header.payloadOffset:]

	// rfc3984 5.3.  NAL Unit Octet Usage
	//
	// +---------------+
	// |0|1|2|3|4|5|6|7|
	// +-+-+-+-+-+-+-+-+
	// |F|NRI|  Type   |
	// +---------------+

	outerNALUType := b[0] & 0x1F
	if outerNALUType <= NALUTypeAVCSingleMax {
		pkt.positionType = PositionTypeSingle
		return
	} else if outerNALUType == NALUTypeAVCFUA {

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

		fuIndicator := b[0]
		_ = fuIndicator
		fuHeader := b[1]

		startCode := (fuHeader & 0x80) != 0
		endCode := (fuHeader & 0x40) != 0

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
	} else if outerNALUType == NALUTypeAVCSTAPA {
		pkt.positionType = PositionTypeSTAPA
	} else {
		nazalog.Errorf("unknown nalu type. outerNALUType=%d", outerNALUType)
	}

	return
}

// AVC或HEVC格式的流，尝试合成一个完整的帧
func (r *RTPUnpacker) unpackOneAVCOrHEVC() bool {
	first := r.list.head.next
	if first == nil {
		return false
	}

	switch first.packet.positionType {
	case PositionTypeSingle:
		var pkt base.AVPacket
		pkt.PayloadType = r.payloadType
		pkt.Timestamp = first.packet.Header.Timestamp / uint32(r.clockRate/1000)

		pkt.Payload = make([]byte, len(first.packet.Raw)-int(first.packet.Header.payloadOffset)+4)
		bele.BEPutUint32(pkt.Payload, uint32(len(first.packet.Raw))-first.packet.Header.payloadOffset)
		copy(pkt.Payload[4:], first.packet.Raw[first.packet.Header.payloadOffset:])

		r.unpackedFlag = true
		r.unpackedSeq = first.packet.Header.Seq
		r.list.head.next = first.next
		r.list.size--
		r.onAVPacket(pkt)

		return true

	case PositionTypeSTAPA:
		var pkt base.AVPacket
		pkt.PayloadType = r.payloadType
		pkt.Timestamp = first.packet.Header.Timestamp / uint32(r.clockRate/1000)

		// 跳过首字节，并且将多nalu前的2字节长度，替换成4字节长度
		buf := first.packet.Raw[first.packet.Header.payloadOffset+1:]

		// 使用两次遍历，第一次遍历找出总大小，第二次逐个拷贝，目的是使得内存块一次就申请好，不用动态扩容造成额外性能开销
		totalSize := 0
		for i := 0; i != len(buf); {
			if len(buf)-i < 2 {
				nazalog.Errorf("invalid STAP-A packet.")
				return false
			}
			naluSize := int(bele.BEUint16(buf[i:]))
			totalSize += 4 + naluSize
			i += 2 + naluSize
		}

		pkt.Payload = make([]byte, totalSize)
		j := 0
		for i := 0; i != len(buf); {
			naluSize := int(bele.BEUint16(buf[i:]))
			bele.BEPutUint32(pkt.Payload[j:], uint32(naluSize))
			copy(pkt.Payload[j+4:], buf[i+2:i+2+naluSize])
			j += 4 + naluSize
			i += 2 + naluSize
		}

		r.unpackedFlag = true
		r.unpackedSeq = first.packet.Header.Seq
		r.list.head.next = first.next
		r.list.size--
		r.onAVPacket(pkt)

		return true

	case PositionTypeFUAStart:
		prev := first
		p := first.next
		for {
			if prev == nil || p == nil {
				return false
			}
			if SubSeq(p.packet.Header.Seq, prev.packet.Header.Seq) != 1 {
				return false
			}

			if p.packet.positionType == PositionTypeFUAMiddle {
				prev = p
				p = p.next
				continue
			} else if p.packet.positionType == PositionTypeFUAEnd {
				var pkt base.AVPacket
				pkt.PayloadType = r.payloadType
				pkt.Timestamp = p.packet.Header.Timestamp / uint32(r.clockRate/1000)

				var naluTypeLen int
				var naluType []byte
				if r.payloadType == base.AVPacketPTAVC {
					naluTypeLen = 1
					naluType = make([]byte, naluTypeLen)

					fuIndicator := first.packet.Raw[first.packet.Header.payloadOffset]
					fuHeader := first.packet.Raw[first.packet.Header.payloadOffset+1]
					naluType[0] = (fuIndicator & 0xE0) | (fuHeader & 0x1F)
				} else {
					naluTypeLen = 2
					naluType = make([]byte, naluTypeLen)

					buf := first.packet.Raw[first.packet.Header.payloadOffset:]
					fuType := buf[2] & 0x3f
					naluType[0] = (buf[0] & 0x81) | (fuType << 1)
					naluType[1] = buf[1]
				}

				// 使用两次遍历，第一次遍历找出总大小，第二次逐个拷贝，目的是使得内存块一次就申请好，不用动态扩容造成额外性能开销
				totalSize := 0
				pp := first
				for {
					totalSize += len(pp.packet.Raw) - int(pp.packet.Header.payloadOffset) - (naluTypeLen + 1)
					if pp == p {
						break
					}
					pp = pp.next
				}

				pkt.Payload = make([]byte, totalSize+4+naluTypeLen)
				bele.BEPutUint32(pkt.Payload, uint32(totalSize+naluTypeLen))
				var index int
				if r.payloadType == base.AVPacketPTAVC {
					pkt.Payload[4] = naluType[0]
					index = 5
				} else {
					pkt.Payload[4] = naluType[0]
					pkt.Payload[5] = naluType[1]
					index = 6
				}
				packetCount := 0
				pp = first
				for {
					copy(pkt.Payload[index:], pp.packet.Raw[int(pp.packet.Header.payloadOffset)+(naluTypeLen+1):])
					index += len(pp.packet.Raw) - int(pp.packet.Header.payloadOffset) - (naluTypeLen + 1)
					packetCount++

					if pp == p {
						break
					}
					pp = pp.next
				}

				r.unpackedFlag = true
				r.unpackedSeq = p.packet.Header.Seq
				r.list.head.next = p.next
				r.list.size -= packetCount
				r.onAVPacket(pkt)

				return true
			} else {
				// 不应该出现其他类型
				nazalog.Errorf("invalid position type. position=%d", p.packet.positionType)
				return false
			}
		}

	case PositionTypeFUAMiddle:
		// noop
	case PositionTypeFUAEnd:
		// noop
	default:
		nazalog.Errorf("invalid position. pos=%d", first.packet.positionType)
	}

	return false
}
