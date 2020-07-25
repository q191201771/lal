// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

type OnAVPacket func(pkt AVPacket)

type Stream struct {
	onAVPacket OnAVPacket
	composer   *RTPComposer
}

func NewStream(payloadType int, clockRate int, onAVPacket OnAVPacket) *Stream {
	var s Stream
	s.onAVPacket = onAVPacket
	s.composer = NewRTPComposer(payloadType, clockRate, composerItemMaxSize, s.onAVPacketComposed)
	return &s
}

func (s *Stream) FeedAVCPacket(pkt RTPPacket) {
	s.composer.Feed(pkt)
}

func (s *Stream) FeedAACPacket(pkt RTPPacket) {
	s.composer.Feed(pkt)
}

func (s *Stream) onAVPacketComposed(pkt AVPacket) {
	s.onAVPacket(pkt)
}
