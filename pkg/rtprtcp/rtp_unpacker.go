// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtprtcp

import (
	"github.com/cfeeling/lal/pkg/base"
)

// 传入RTP包，合成帧数据，并回调。
// 一路音频或一路视频各对应一个对象。
// 目前支持AVC，HEVC和AAC MPEG4-GENERIC/44100/2

type RTPPacketListItem struct {
	packet RTPPacket
	next   *RTPPacketListItem
}

type RTPPacketList struct {
	head RTPPacketListItem // 哨兵，自身不存放rtp包，第一个rtp包存在在head.next中
	size int               // 实际元素个数
}

type RTPUnpacker struct {
	payloadType base.AVPacketPT
	clockRate   int
	maxSize     int
	onAVPacket  OnAVPacket

	list         RTPPacketList
	unpackedFlag bool
	unpackedSeq  uint16
}

// @param pkt: pkt.Timestamp   RTP包头中的时间戳(pts)经过clockrate换算后的时间戳，单位毫秒
//                             注意，不支持带B帧的视频流，pts和dts永远相同
//             pkt.PayloadType base.AVPacketPTXXX
//             pkt.Payload     如果是AAC，返回的是raw frame，一个AVPacket只包含一帧
//                             如果是AVC或HEVC，一个AVPacket可能包含多个NAL(受STAP-A影响)，所以NAL前包含4字节的长度信息
//                             AAC引用的是接收到的RTP包中的内存块
//                             AVC或者HEVC是新申请的内存块，回调结束后，内部不再使用该内存块
type OnAVPacket func(pkt base.AVPacket)

func NewRTPUnpacker(payloadType base.AVPacketPT, clockRate int, maxSize int, onAVPacket OnAVPacket) *RTPUnpacker {
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

	r.calcPositionIfNeeded(&pkt)
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

// 计算rtp包处于帧中的位置
func (r *RTPUnpacker) calcPositionIfNeeded(pkt *RTPPacket) {
	switch r.payloadType {
	case base.AVPacketPTAVC:
		calcPositionIfNeededAVC(pkt)
	case base.AVPacketPTHEVC:
		calcPositionIfNeededHEVC(pkt)
	case base.AVPacketPTAAC:
		// noop
		break
	default:
		// can't reach here
	}
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
	case base.AVPacketPTAAC:
		return r.unpackOneAAC()
	case base.AVPacketPTAVC:
		fallthrough
	case base.AVPacketPTHEVC:
		return r.unpackOneAVCOrHEVC()
	}

	return false
}
