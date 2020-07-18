// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

// TODO chef: move to package base
type AVPacket struct {
	timestamp uint32
	payload   []byte
}

type RTPPacketListItem struct {
	packet RTPPacket
	next   *RTPPacketListItem
}

type RTPPacketList struct {
	head RTPPacketListItem // 哨兵，自身不存放rtp包
	size int               // 实际元素个数
}

type RTPComposer struct {
	maxSize int
	cb      OnAVPacketComposed

	list         RTPPacketList
	composedFlag bool
	composedSeq  uint16
}

type OnAVPacketComposed func(pkt AVPacket)

func NewRTPComposer(maxSize int, cb OnAVPacketComposed) *RTPComposer {
	return &RTPComposer{
		maxSize: maxSize,
		cb:      cb,
	}
}

func (r *RTPComposer) Feed(pkt RTPPacket) {
	if r.isStale(pkt.header.seq) {
		return
	}
	calcPosition(pkt)
	r.insert(pkt)
}

// 检查rtp包是否已经过期
func (r *RTPComposer) isStale(seq uint16) bool {
	if !r.composedFlag {
		return false
	}
	return compareSeq(seq, r.composedSeq) <= 0
}

// 计算rtp包处于帧中的位置
func calcPosition(pkt RTPPacket) {
	// TODO chef: 目前只写了264部分

	b := pkt.raw[pkt.header.payloadOffset:]

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
	l := r.list
	l.size++

	p := &l.head
	for ; p.next != nil; p = p.next {
		res := compareSeq(pkt.header.seq, p.next.packet.header.seq)
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

// 从头部检查，尽可能多的合成连续的完整的帧
func (r *RTPComposer) tryCompose() {
	for p := r.list.head.next; p != nil; p = p.next {
		switch p.packet.positionType {
		case PositionTypeSingle:

		}
	}
}

// 从头部检查，是否可以合成一个完整的帧
// TODO chef: 增加参数，用于区分两种逻辑，是连续的帧，还是跳跃的帧
func (r *RTPComposer) tryComposeOne() bool {
	first := r.list.head.next
	if first == nil {
		return false
	}

	switch first.packet.positionType {
	case PositionTypeSingle:
		pkt := AVPacket{
			timestamp: first.packet.header.timestamp,
			payload:   first.packet.raw[first.packet.header.payloadOffset:],
		}
		r.composedFlag = true
		r.composedSeq = first.packet.header.seq
		r.cb(pkt)

		return true
	case PositionTypeMultiStart:

		// to be continued
	}

	return false
}
