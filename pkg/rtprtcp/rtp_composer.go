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
	"github.com/q191201771/naza/pkg/nazalog"
)

// 传入RTP包，合成帧数据，并回调
// 一路音频或一路视频对应一个对象
// 目前支持AVC和AAC

// TODO chef: 由于音频数据，存在多个帧放一个RTP包的情况，叫composer不一定合适了，可以改名为unpacker

type RTPPacketListItem struct {
	packet RTPPacket
	next   *RTPPacketListItem
}

type RTPPacketList struct {
	head RTPPacketListItem // 哨兵，自身不存放rtp包，第一个rtp包存在在head.next中
	size int               // 实际元素个数
}

type RTPComposer struct {
	payloadType        int
	clockRate          int
	maxSize            int
	onAVPacketComposed OnAVPacketComposed

	list         RTPPacketList
	composedFlag bool
	composedSeq  uint16
}

// @param pkt: Timestamp   返回的是RTP包头中的时间戳(pts)，经过clockrate换算后的时间戳，单位毫秒
//             PayloadType 返回的是RTPPacketTypeXXX
type OnAVPacketComposed func(pkt base.AVPacket)

func NewRTPComposer(payloadType int, clockRate int, maxSize int, onAVPacketComposed OnAVPacketComposed) *RTPComposer {
	return &RTPComposer{
		payloadType:        payloadType,
		clockRate:          clockRate,
		maxSize:            maxSize,
		onAVPacketComposed: onAVPacketComposed,
	}
}

func (r *RTPComposer) Feed(pkt RTPPacket) {
	if r.isStale(pkt.Header.Seq) {
		return
	}
	calcPositionIfNeeded(&pkt)
	r.insert(pkt)

	// 尽可能多的合成顺序的帧
	count := 0
	for {
		if !r.composeOneSequential() {
			break
		}
		count++
	}

	// 合成顺序的帧成功了，直接返回
	if count > 0 {
		return
	}

	// 缓存达到最大值
	if r.list.size > r.maxSize {
		// 尝试合成一帧发生跳跃的帧
		if !r.composeOne() {

			// 合成失败了，丢弃过期数据
			r.list.head.next = r.list.head.next.next
			r.list.size--
		}

		// 再次尝试，尽可能多的合成顺序的帧
		for {
			if !r.composeOneSequential() {
				break
			}
		}
	}
}

// 检查rtp包是否已经过期
//
// @return true  表示过期
//         false 没过期
//
func (r *RTPComposer) isStale(seq uint16) bool {
	if !r.composedFlag {
		return false
	}
	return CompareSeq(seq, r.composedSeq) <= 0
}

// 计算rtp包处于帧中的位置
func calcPositionIfNeeded(pkt *RTPPacket) {
	if pkt.Header.PacketType != base.RTPPacketTypeAVC {
		return
	}

	b := pkt.Raw[pkt.Header.payloadOffset:]

	// rfc3984 5.3.  NAL Unit Octet Usage
	//
	// +---------------+
	// |0|1|2|3|4|5|6|7|
	// +-+-+-+-+-+-+-+-+
	// |F|NRI|  Type   |
	// +---------------+

	outerNALUType := b[0] & 0x1F
	if outerNALUType <= NALUTypeSingleMax {
		pkt.positionType = PositionTypeSingle
		return
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

		if startCode {
			pkt.positionType = PositionTypeMultiStart
			return
		}

		if endCode {
			pkt.positionType = PositionTypeMultiEnd
			return
		}

		pkt.positionType = PositionTypeMultiMiddle
		return
	}
}

// 将rtp包插入队列中的合适位置
func (r *RTPComposer) insert(pkt RTPPacket) {
	//l := r.list
	r.list.size++

	p := &r.list.head
	for ; p.next != nil; p = p.next {
		res := CompareSeq(pkt.Header.Seq, p.next.packet.Header.Seq)
		switch res {
		case 0:
			return
		case 1:
			// noop
		case -1:
			item := &RTPPacketListItem{
				packet: pkt,
				next:   p.next,
			}
			p.next = item
			return
		}
	}

	item := &RTPPacketListItem{
		packet: pkt,
		next:   p.next,
	}
	p.next = item
}

// 从头部检查，是否可以合成一个完成的帧。并且，需保证这次合成的帧的首个seq和上次处理的seq是连续的
func (r *RTPComposer) composeOneSequential() bool {
	if r.composedFlag {
		first := r.list.head.next
		if first == nil {
			return false
		}
		if SubSeq(first.packet.Header.Seq, r.composedSeq) != 1 {
			return false
		}
	}

	return r.composeOne()
}

func (r *RTPComposer) composeOne() bool {
	switch r.payloadType {
	case base.RTPPacketTypeAVC:
		return r.composeOneAVC()
	case base.RTPPacketTypeAAC:
		return r.composeOneAAC()
	}

	return false
}

func (r *RTPComposer) composeOneAAC() bool {
	first := r.list.head.next
	if first == nil {
		return false
	}

	// TODO chef:
	// 目前只实现了AAC MPEG4-GENERIC/44100/2
	//
	// 只处理了一个RTP包含多个音频包的情况
	// 没有处理一个音频包跨越多个RTP包的情况

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

	b := first.packet.Raw[first.packet.Header.payloadOffset:]
	//nazalog.Debugf("%d, %d, %s", len(pkt.Raw), pkt.Header.timestamp, hex.Dump(b))

	// AU Header Section
	var auHeaderLength uint32
	auHeaderLength = uint32(b[0])<<8 + uint32(b[1])
	auHeaderLength = (auHeaderLength + 7) / 8
	//nazalog.Debugf("auHeaderLength=%d", auHeaderLength)

	// no Auxiliary Section

	pauh := uint32(2)                 // AU Header pos
	pau := uint32(2) + auHeaderLength // AU pos
	auNum := uint32(auHeaderLength) / 2
	for i := uint32(0); i < auNum; i++ {
		var auSize uint32
		auSize = uint32(b[pauh])<<8 | uint32(b[pauh+1]&0xF8) // 13bit
		auSize /= 8

		//auIndex := b[pauh+1] & 0x7

		// raw AAC frame
		// pau, auSize
		//nazalog.Debugf("%d %d %s", auSize, auIndex, hex.Dump(b[pau:pau+auSize]))
		var outPkt base.AVPacket
		outPkt.Timestamp = first.packet.Header.Timestamp / uint32(r.clockRate/1000)
		outPkt.Timestamp += i * uint32((1024*1000)/r.clockRate)
		outPkt.Payload = b[pau : pau+auSize]
		outPkt.PayloadType = base.RTPPacketTypeAAC

		r.onAVPacketComposed(outPkt)

		pauh += 2
		pau += auSize
	}

	r.composedFlag = true
	r.composedSeq = first.packet.Header.Seq
	r.list.head.next = first.next
	r.list.size--
	return true
}

// 从头部检查，是否可以合成一个完整的帧
func (r *RTPComposer) composeOneAVC() bool {
	first := r.list.head.next
	if first == nil {
		return false
	}

	switch first.packet.positionType {
	case PositionTypeSingle:
		var pkt base.AVPacket
		pkt.Timestamp = first.packet.Header.Timestamp / uint32(r.clockRate/1000)
		pkt.Payload = first.packet.Raw[first.packet.Header.payloadOffset:]
		pkt.PayloadType = base.RTPPacketTypeAVC

		r.composedFlag = true
		r.composedSeq = first.packet.Header.Seq
		r.list.head.next = first.next
		r.list.size--
		r.onAVPacketComposed(pkt)

		return true
	case PositionTypeMultiStart:
		prev := first
		p := first.next
		for {
			if prev == nil || p == nil {
				return false
			}
			if SubSeq(p.packet.Header.Seq, prev.packet.Header.Seq) != 1 {
				return false
			}

			if p.packet.positionType == PositionTypeMultiMiddle {
				prev = p
				p = p.next
				continue
			} else if p.packet.positionType == PositionTypeMultiEnd {
				var pkt base.AVPacket
				pkt.Timestamp = p.packet.Header.Timestamp / uint32(r.clockRate/1000)

				fuIndicator := first.packet.Raw[first.packet.Header.payloadOffset]
				fuHeader := first.packet.Raw[first.packet.Header.payloadOffset+1]
				naluType := (fuIndicator & 0xE0) | (fuHeader & 0x1F)
				pkt.Payload = append(pkt.Payload, naluType)
				pkt.PayloadType = base.RTPPacketTypeAVC

				pp := first
				packetCount := 0
				for {
					pkt.Payload = append(pkt.Payload, pp.packet.Raw[pp.packet.Header.payloadOffset+2:]...)
					packetCount++

					if pp == p {
						break
					}
					pp = pp.next
				}

				r.composedFlag = true
				r.composedSeq = p.packet.Header.Seq
				r.list.head.next = p.next
				r.list.size -= packetCount
				r.onAVPacketComposed(pkt)

				return true
			} else {
				// 正常不应该出现single和start
				nazalog.Errorf("invalid position type. position=%d", p.packet.positionType)
				return false
			}
		}
	case PositionTypeMultiMiddle:
		// noop
	case PositionTypeMultiEnd:
		// noop
	}

	return false
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
