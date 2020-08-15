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

// 传入RTP包，合成帧数据，并回调。
// 一路音频或一路视频各对应一个对象。
// 目前支持AVC和AAC MPEG4-GENERIC/44100/2
// 后续增加其他格式，可能会拆分出一些子结构体

type RTPPacketListItem struct {
	packet RTPPacket
	next   *RTPPacketListItem
}

type RTPPacketList struct {
	head RTPPacketListItem // 哨兵，自身不存放rtp包，第一个rtp包存在在head.next中
	size int               // 实际元素个数
}

type RTPUnpacker struct {
	payloadType int
	clockRate   int
	maxSize     int
	onAVPacket  OnAVPacket

	list         RTPPacketList
	unpackedFlag bool
	unpackedSeq  uint16
}

// @param pkt: pkt.Timestamp   RTP包头中的时间戳(pts)经过clockrate换算后的时间戳，单位毫秒
//                             注意，不支持带B帧的视频流，pts和dts永远相同
//             pkt.PayloadType base.RTPPacketTypeXXX
//             pkt.Payload     如果是AAC，返回的是raw frame，一个AVPacket只包含一帧
//                             如果是AVC，一个AVPacket可能包含多个NAL(受STAP-A影响)，所以NAL前包含4字节的长度信息
//                             AAC引用的是接收到的RTP包中的内存块
//                             AVC是新申请的内存块，回调结束后，内部不再使用该内存块
type OnAVPacket func(pkt base.AVPacket)

func NewRTPUnpacker(payloadType int, clockRate int, maxSize int, onAVPacket OnAVPacket) *RTPUnpacker {
	return &RTPUnpacker{
		payloadType: payloadType,
		clockRate:   clockRate,
		maxSize:     maxSize,
		onAVPacket:  onAVPacket,
	}
}

// 输入收到的rtp包
func (r *RTPUnpacker) Feed(pkt RTPPacket) {
	if r.isStale(pkt.Header.Seq) {
		return
	}

	calcPositionIfNeeded(&pkt)
	r.insert(pkt)

	// 尽可能多的合成顺序的帧
	count := 0
	for {
		if !r.unpackOneSequential() {
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
		if !r.unpackOne() {

			// 合成失败了，丢弃过期数据
			r.list.head.next = r.list.head.next.next
			r.list.size--
		}

		// 再次尝试，尽可能多的合成顺序的帧
		for {
			if !r.unpackOneSequential() {
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
func (r *RTPUnpacker) isStale(seq uint16) bool {
	if !r.unpackedFlag {
		return false
	}
	return CompareSeq(seq, r.unpackedSeq) <= 0
}

// 对AVC格式的流，计算rtp包处于帧中的位置
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
	//nazalog.Debugf("outerNALUType=%d", outerNALUType)
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
	} else if outerNALUType == NALUTypeSTAPA {
		pkt.positionType = PositionTypeSTAPA
	} else {
		nazalog.Errorf("unknown nalu type. outerNALUType=%d", outerNALUType)
	}

	return
}

// 将rtp包按seq排序插入队列中
func (r *RTPUnpacker) insert(pkt RTPPacket) {
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

// 从队列头部，尝试合成一个完整的帧。保证这次合成的帧的首个seq和上次合成帧的尾部seq是连续的
func (r *RTPUnpacker) unpackOneSequential() bool {
	if r.unpackedFlag {
		first := r.list.head.next
		if first == nil {
			return false
		}
		if SubSeq(first.packet.Header.Seq, r.unpackedSeq) != 1 {
			return false
		}
	}

	return r.unpackOne()
}

// 从队列头部，尝试合成一个完整的帧。不保证这次合成的帧的首个seq和上次合成帧的尾部seq是连续的
func (r *RTPUnpacker) unpackOne() bool {
	switch r.payloadType {
	case base.RTPPacketTypeAVC:
		return r.unpackOneAVC()
	case base.RTPPacketTypeAAC:
		return r.unpackOneAAC()
	}

	return false
}

// AAC格式的流，尝试合成一个完整的帧
func (r *RTPUnpacker) unpackOneAAC() bool {
	first := r.list.head.next
	if first == nil {
		return false
	}

	// TODO chef:
	// 2. 只处理了一个RTP包含多个音频包的情况，没有处理一个音频包跨越多个RTP包的情况（是否有这种情况）

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

		r.onAVPacket(outPkt)

		pauh += 2
		pau += auSize
	}

	r.unpackedFlag = true
	r.unpackedSeq = first.packet.Header.Seq
	r.list.head.next = first.next
	r.list.size--
	return true
}

// AVC格式的流，尝试合成一个完整的帧
func (r *RTPUnpacker) unpackOneAVC() bool {
	first := r.list.head.next
	if first == nil {
		return false
	}

	switch first.packet.positionType {
	case PositionTypeSingle:
		var pkt base.AVPacket
		pkt.PayloadType = base.RTPPacketTypeAVC
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
		pkt.PayloadType = base.RTPPacketTypeAVC
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
				pkt.PayloadType = base.RTPPacketTypeAVC
				pkt.Timestamp = p.packet.Header.Timestamp / uint32(r.clockRate/1000)

				fuIndicator := first.packet.Raw[first.packet.Header.payloadOffset]
				fuHeader := first.packet.Raw[first.packet.Header.payloadOffset+1]
				naluType := (fuIndicator & 0xE0) | (fuHeader & 0x1F)

				// 使用两次遍历，第一次遍历找出总大小，第二次逐个拷贝，目的是使得内存块一次就申请好，不用动态扩容造成额外性能开销
				totalSize := 0
				pp := first
				for {
					totalSize += len(pp.packet.Raw) - int(pp.packet.Header.payloadOffset) - 2
					if pp == p {
						break
					}
					pp = pp.next
				}

				pkt.Payload = make([]byte, totalSize+5) // 4+1
				bele.BEPutUint32(pkt.Payload, uint32(totalSize+1))
				pkt.Payload[4] = naluType
				index := 5
				packetCount := 0
				pp = first
				for {
					copy(pkt.Payload[index:], pp.packet.Raw[pp.packet.Header.payloadOffset+2:])
					index += len(pp.packet.Raw) - int(pp.packet.Header.payloadOffset) - 2
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
