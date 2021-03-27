// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/circularqueue"
	"github.com/q191201771/naza/pkg/nazalog"
)

// 处理音频和视频的时间戳：
// 1. 让音频和视频的时间戳都从0开始（改变原时间戳）
// 2. 让音频和视频的时间戳交替递增输出（不改变原时间戳）

// 注意，本模块默认音频和视频都存在，如果只有音频或只有视频，则不要使用该模块

const maxQueueSize = 128

type OnAVPacket func(pkt base.AVPacket)

type AVPacketQueue struct {
	onAVPacket  OnAVPacket
	audioBaseTS int64                        // audio base timestamp
	videoBaseTS int64                        // video base timestamp
	audioQueue  *circularqueue.CircularQueue // TODO chef: 特化成AVPacket类型
	videoQueue  *circularqueue.CircularQueue
}

func NewAVPacketQueue(onAVPacket OnAVPacket) *AVPacketQueue {
	return &AVPacketQueue{
		onAVPacket:  onAVPacket,
		audioBaseTS: -1,
		videoBaseTS: -1,
		audioQueue:  circularqueue.New(maxQueueSize),
		videoQueue:  circularqueue.New(maxQueueSize),
	}
}

// 注意，调用方保证，音频相较于音频，视频相较于视频，时间戳是线性递增的。
func (a *AVPacketQueue) Feed(pkt base.AVPacket) {
	//nazalog.Debugf("AVQ feed. t=%d, ts=%d", pkt.PayloadType, pkt.Timestamp)
	switch pkt.PayloadType {
	case base.AVPacketPTAVC:
		fallthrough
	case base.AVPacketPTHEVC:
		// 时间戳回退了
		if int64(pkt.Timestamp) < a.videoBaseTS {
			nazalog.Warnf("video ts rotate. pktTS=%d, audioBaseTS=%d, videoBaseTS=%d, audioQueue=%d, videoQueue=%d",
				pkt.Timestamp, a.audioBaseTS, a.videoBaseTS, a.audioQueue.Size(), a.videoQueue.Size())
			a.videoBaseTS = -1
			a.audioBaseTS = -1
			a.PopAllByForce()
		}
		// 第一次
		if a.videoBaseTS == -1 {
			a.videoBaseTS = int64(pkt.Timestamp)
		}
		// 根据基准调节
		pkt.Timestamp -= uint32(a.videoBaseTS)

		_ = a.videoQueue.PushBack(pkt)
	case base.AVPacketPTAAC:
		if int64(pkt.Timestamp) < a.audioBaseTS {
			nazalog.Warnf("audio ts rotate. pktTS=%d, audioBaseTS=%d, videoBaseTS=%d, audioQueue=%d, videoQueue=%d",
				pkt.Timestamp, a.audioBaseTS, a.videoBaseTS, a.audioQueue.Size(), a.videoQueue.Size())
			a.videoBaseTS = -1
			a.audioBaseTS = -1
			a.PopAllByForce()
		}
		if a.audioBaseTS == -1 {
			a.audioBaseTS = int64(pkt.Timestamp)
		}
		pkt.Timestamp -= uint32(a.audioBaseTS)
		_ = a.audioQueue.PushBack(pkt)
	}

	// 如果音频和视频都存在，则按序输出，直到其中一个为空
	for !a.audioQueue.Empty() && !a.videoQueue.Empty() {
		apkt, _ := a.audioQueue.Front()
		vpkt, _ := a.videoQueue.Front()
		aapkt := apkt.(base.AVPacket)
		vvpkt := vpkt.(base.AVPacket)
		if aapkt.Timestamp < vvpkt.Timestamp {
			_, _ = a.audioQueue.PopFront()
			a.onAVPacket(aapkt)
		} else {
			_, _ = a.videoQueue.PopFront()
			a.onAVPacket(vvpkt)
		}
	}

	// 如果视频满了，则全部输出
	if a.videoQueue.Full() {
		nazalog.Assert(true, a.audioQueue.Empty())
		a.popAllVideo()
		return
	}

	// 如果音频满了，则全部输出
	if a.audioQueue.Full() {
		nazalog.Assert(true, a.videoQueue.Empty())
		a.popAllAudio()
		return
	}
}

func (a *AVPacketQueue) PopAllByForce() {
	if a.audioQueue.Empty() && a.videoQueue.Empty() {
		// noop
	} else if a.audioQueue.Empty() && !a.videoQueue.Empty() {
		a.popAllVideo()
	} else if !a.audioQueue.Empty() && a.videoQueue.Empty() {
		a.popAllAudio()
	}

	// never reach here
	nazalog.Assert(false, !a.audioQueue.Empty() && !a.videoQueue.Empty())
}

func (a *AVPacketQueue) popAllAudio() {
	for !a.audioQueue.Empty() {
		pkt, _ := a.audioQueue.Front()
		ppkt := pkt.(base.AVPacket)
		_, _ = a.audioQueue.PopFront()
		a.onAVPacket(ppkt)
	}
}

func (a *AVPacketQueue) popAllVideo() {
	for !a.videoQueue.Empty() {
		pkt, _ := a.videoQueue.Front()
		ppkt := pkt.(base.AVPacket)
		_, _ = a.videoQueue.PopFront()
		a.onAVPacket(ppkt)
	}
}
