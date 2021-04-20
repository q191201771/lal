// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtprtcp

type RTPPacketListItem struct {
	packet RTPPacket
	next   *RTPPacketListItem
}

type RTPPacketList struct {
	head RTPPacketListItem // 哨兵，自身不存放rtp包，第一个rtp包存在在head.next中
	size int               // 实际元素个数
}

type RTPUnpackContainer struct {
	maxSize int

	unpackerProtocol IRTPUnpackerProtocol

	list         RTPPacketList
	unpackedFlag bool   // 是否成功合成过
	unpackedSeq  uint16 // 成功合成的最后一个seq号
}

func NewRTPUnpackContainer(maxSize int, unpackerProtocol IRTPUnpackerProtocol) *RTPUnpackContainer {
	return &RTPUnpackContainer{
		maxSize:          maxSize,
		unpackerProtocol: unpackerProtocol,
	}
}

// 输入收到的rtp包
func (r *RTPUnpackContainer) Feed(pkt RTPPacket) {
	// 过期的包
	if r.isStale(pkt.Header.Seq) {
		return
	}

	// 计算位置
	r.unpackerProtocol.CalcPositionIfNeeded(&pkt)
	// 根据序号插入有序链表
	r.insert(pkt)

	// 尽可能多的合成顺序的帧
	count := 0
	for {
		if !r.tryUnpackOneSequential() {
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
		packed := r.tryUnpackOne()

		if !packed {
			// 合成失败了，丢弃一包过期数据
			r.list.head.next = r.list.head.next.next
			r.list.size--
		} else {
			// 合成成功了，再次尝试，尽可能多的合成顺序的帧
			for {
				if !r.tryUnpackOneSequential() {
					break
				}
			}
		}
	}
}

// 检查rtp包是否已经过期
//
// @return true  表示过期
//         false 没过期
//
func (r *RTPUnpackContainer) isStale(seq uint16) bool {
	// 从来没有合成成功过
	if !r.unpackedFlag {
		return false
	}
	// 序号太小
	return CompareSeq(seq, r.unpackedSeq) <= 0
}

// 将rtp包按seq排序插入队列中
func (r *RTPUnpackContainer) insert(pkt RTPPacket) {
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
func (r *RTPUnpackContainer) tryUnpackOneSequential() bool {
	if r.unpackedFlag {
		first := r.list.head.next
		if first == nil {
			return false
		}
		if SubSeq(first.packet.Header.Seq, r.unpackedSeq) != 1 {
			return false
		}
	}

	return r.tryUnpackOne()
}

// 从队列头部，尝试合成一个完整的帧。不保证这次合成的帧的首个seq和上次合成帧的尾部seq是连续的
func (r *RTPUnpackContainer) tryUnpackOne() bool {
	unpackedFlag, unpackedSeq := r.unpackerProtocol.TryUnpackOne(&r.list)
	if unpackedFlag {
		r.unpackedFlag = unpackedFlag
		r.unpackedSeq = unpackedSeq
	}
	return unpackedFlag
}
