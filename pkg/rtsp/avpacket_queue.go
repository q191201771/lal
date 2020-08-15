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
	case base.RTPPacketTypeAVC:
		if a.videoBaseTS == -1 {
			a.videoBaseTS = int64(pkt.Timestamp)
		}
		pkt.Timestamp -= uint32(a.videoBaseTS)

		if a.videoQueue.Full() {
			_, _ = a.videoQueue.PopFront()
			nazalog.Warnf("video queue full, drop front packet.")
		}
		_ = a.videoQueue.PushBack(pkt)
		//nazalog.Debugf("AVQ v push. a=%d, v=%d", a.audioQueue.Size(), a.videoQueue.Size())
	case base.RTPPacketTypeAAC:
		if a.audioBaseTS == -1 {
			a.audioBaseTS = int64(pkt.Timestamp)
		}
		pkt.Timestamp -= uint32(a.audioBaseTS)

		if a.audioQueue.Full() {
			_, _ = a.audioQueue.PopFront()
			nazalog.Warnf("audio queue full, drop front packet. a=%d, v=%d", a.audioQueue.Size(), a.videoQueue.Size())
		}
		_ = a.audioQueue.PushBack(pkt)
		//nazalog.Debugf("AVQ a push. a=%d, v=%d", a.audioQueue.Size(), a.videoQueue.Size())
	} //switch loop

	for !a.audioQueue.Empty() && !a.videoQueue.Empty() {
		apkt, _ := a.audioQueue.Front()
		vpkt, _ := a.videoQueue.Front()
		aapkt := apkt.(base.AVPacket)
		vvpkt := vpkt.(base.AVPacket)
		if aapkt.Timestamp < vvpkt.Timestamp {
			_, _ = a.audioQueue.PopFront()
			//nazalog.Debugf("AVQ a pop. a=%d, v=%d", a.audioQueue.Size(), a.videoQueue.Size())
			a.onAVPacket(aapkt)
		} else {
			_, _ = a.videoQueue.PopFront()
			//nazalog.Debugf("AVQ v pop. a=%d, v=%d", a.audioQueue.Size(), a.videoQueue.Size())
			a.onAVPacket(vvpkt)
		}
	}
}
