// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/mpegts"
)

// 缓存流起始的一些数据，判断流中是否存在音频、视频，以及编码格式
// 一旦判断结束，该队列变成直进直出，不再有实际缓存
type Queue struct {
	maxMsgSize int
	data       []base.RTMPMsg
	observer   IQueueObserver

	audioCodecID int
	videoCodecID int
	done         bool
}

type IQueueObserver interface {
	// 该回调一定发生在数据回调之前
	// TODO(chef) 这里可以考虑换成只通知drain，由上层完成FragmentHeader的组装逻辑
	OnPATPMT(b []byte)

	OnPop(msg base.RTMPMsg)
}

// @param maxMsgSize 最大缓存多少个包
func NewQueue(maxMsgSize int, observer IQueueObserver) *Queue {
	return &Queue{
		maxMsgSize:   maxMsgSize,
		data:         make([]base.RTMPMsg, maxMsgSize)[0:0],
		observer:     observer,
		audioCodecID: -1,
		videoCodecID: -1,
		done:         false,
	}
}

// @param msg 函数调用结束后，内部不持有该内存块
func (q *Queue) Push(msg base.RTMPMsg) {
	if q.done {
		q.observer.OnPop(msg)
		return
	}

	q.data = append(q.data, msg.Clone())

	switch msg.Header.MsgTypeID {
	case base.RTMPTypeIDAudio:
		q.audioCodecID = int(msg.Payload[0] >> 4)
	case base.RTMPTypeIDVideo:
		q.videoCodecID = int(msg.Payload[0] & 0xF)
	}

	if q.videoCodecID != -1 && q.audioCodecID != -1 {
		q.drain()
		return
	}

	if len(q.data) >= q.maxMsgSize {
		q.drain()
		return
	}
}

func (q *Queue) drain() {
	switch q.videoCodecID {
	case int(base.RTMPCodecIDAVC):
		q.observer.OnPATPMT(mpegts.FixedFragmentHeader)
	case int(base.RTMPCodecIDHEVC):
		q.observer.OnPATPMT(mpegts.FixedFragmentHeaderHEVC)
	default:
		// TODO(chef) 正确处理只有音频或只有视频的情况 #56
		q.observer.OnPATPMT(mpegts.FixedFragmentHeader)
	}
	for i := range q.data {
		q.observer.OnPop(q.data[i])
	}
	q.data = nil

	q.done = true
}
