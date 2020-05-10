// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import (
	"fmt"
	"os"

	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazalog"
)

type Session struct {
	spspps   []byte
	videoCC  uint8
	opened   bool
	frag     uint64
	videoOut []byte
}

func NewSession() *Session {
	videoOut := make([]byte, 1024*1024)
	return &Session{
		videoOut: videoOut,
	}
}

func (s *Session) Start() {

}

func (s *Session) FeedRTMPMessage(msg rtmp.AVMsg) {
	// HLS功能正在实现中
	return

	switch msg.Header.MsgTypeID {
	case rtmp.TypeidAudio:
		s.feedAudio(msg)
	case rtmp.TypeidVideo:
		s.feedVideo(msg)
	}
}

func (s *Session) Stop() {

}

func (s *Session) feedAudio(msg rtmp.AVMsg) {

}

func (s *Session) feedVideo(msg rtmp.AVMsg) {
	ftype := msg.Payload[0] & 0xf0 >> 4
	htype := msg.Payload[1]
	if ftype == 1 && htype == 0 {
		s.cacheSPSPPS(msg)
		return
	}
	cts := bele.BEUint24(msg.Payload[2:])

	audSent := false
	spsppsSent := false
	// 优化这块buffer
	out := s.videoOut[0:0]
	for i := 5; i != len(msg.Payload); {
		nalBytes := int(bele.BEUint32(msg.Payload[i:]))
		i += 4
		srcNalType := msg.Payload[i]
		nalType := srcNalType & 0x1F
		nazalog.Debug(nalType)

		if nalType >= 7 && nalType <= 9 {
			nazalog.Warn("should not reach here.")
			i += nalBytes
			continue
		}

		if !audSent {
			switch nalType {
			case 1, 5, 6:
				out = append(out, audNal...)
				audSent = true
			case 9:
				audSent = true
			}
		}

		switch nalType {
		case 1:
			spsppsSent = false
		case 5:
			if !spsppsSent {
				out = s.appendSPSPPS(out)
			}
			spsppsSent = true

		}

		if len(out) == 0 {
			out = append(out, nalStartCode...)
		} else {
			out = append(out, nalStartCode3...)
		}
		out = append(out, msg.Payload[i:i+nalBytes]...)

		i += nalBytes
	}

	var frame MPEGTSFrame
	frame.cc = s.videoCC
	frame.dts = uint64(msg.Header.TimestampAbs) * 90
	frame.pts = frame.dts + uint64(cts)*90
	frame.pid = pidVideo
	frame.sid = streamIDVideo
	frame.key = ftype == 1

	boundary := frame.key && !s.opened

	s.updateFragment(frame.dts, boundary, 1)
	mpegtsWriteFrame(&frame, out)
	s.videoCC = frame.cc
}

func (s *Session) cacheSPSPPS(msg rtmp.AVMsg) {
	nazalog.Debug("cacheSPSPPS")
	s.spspps = msg.Payload
}

func (s *Session) appendSPSPPS(out []byte) []byte {
	index := 10
	nnals := s.spspps[index] & 0x1f
	index++
	nazalog.Debugf("SPS number: %d", nnals)
	for n := 0; ; n++ {
		for ; nnals != 0; nnals-- {
			len := int(bele.BEUint16(s.spspps[index:]))
			nazalog.Debugf("header NAL length:%d", len)
			index += 2
			out = append(out, nalStartCode...)
			out = append(out, s.spspps[index:index+len]...)
			index += len
		}

		if n == 1 {
			break
		}
		nnals = s.spspps[index]
		nazalog.Debugf("PPS number: %d", nnals)
		index++
	}
	return out
}

func (s *Session) updateFragment(dts uint64, boundary bool, flushRate int) {
	force := false
	discont := true
	if boundary || force {
		s.openFragment(dts, discont)
	}
}

func (s *Session) openFragment(ts uint64, discont bool) {
	if s.opened {
		return
	}

	s.ensureDir()
	id := s.getFragmentID()

	filename := fmt.Sprintf("%s%d.ts", outPath, id)
	mpegtsOpenFile(filename)
	s.opened = true
}

func (s *Session) ensureDir() {
	err := os.MkdirAll(outPath, 0777)
	nazalog.Assert(nil, err)
}

func (s *Session) getFragmentID() uint64 {
	return s.frag
}
