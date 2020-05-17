// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/q191201771/lal/pkg/aac"

	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazalog"
)

type Frag struct {
	id       uint64
	keyID    uint64
	duration float64 // 当前fragment中数据的时长，单位秒
	active   bool
	discont  bool
}

type Session struct {
	adts aac.ADTS
	//aacSeqHeader []byte
	spspps   []byte
	videoCC  uint8
	opened   bool
	videoOut []byte // 帧
	fp       *os.File

	fragTS uint64 // 新建立fragment时的时间戳，毫秒 * 90
	frag   int    // 写入m3u8的EXT-X-MEDIA-SEQUENCE字段
	nfrags int    // 大序号，增长到winfrags后，就增长frag
	frags  []Frag // TS文件的环形队列，记录TS的信息，比如写M3U8文件时要用 2 * winfrags + 1

	aframe     []byte
	aframeBase uint64 // 上一个音频帧的时间戳
	//aframeNum  uint64
	aframePTS uint64
}

func NewSession() *Session {
	videoOut := make([]byte, 1024*1024)
	aframe := make([]byte, 1024*1024)
	frags := make([]Frag, 2*winfrags+1) // TODO chef: 为什么是 * 2 + 1
	return &Session{
		videoOut: videoOut,
		aframe:   aframe,
		frags:    frags,
	}
}

func (s *Session) Start() {

}

func (s *Session) FeedRTMPMessage(msg rtmp.AVMsg) {
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

//var debugCount int

func (s *Session) feedVideo(msg rtmp.AVMsg) {
	//if debugCount == 3 {
	//	//os.Exit(0)
	//}
	//debugCount++

	if msg.Payload[0]&0xF != 7 {
		// TODO chef: HLS视频现在只做了h264的支持
		return
	}

	ftype := msg.Payload[0] & 0xF0 >> 4
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

		nazalog.Debugf("hls: h264 NAL type=%d, len=%d(%d) cts=%d.", nalType, nalBytes, len(msg.Payload), cts)

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

	//boundary := frame.key && (true || !s.opened)
	boundary := frame.key

	s.updateFragment(frame.dts, boundary, 1)
	mpegtsWriteFrame(s.fp, &frame, out)
	s.videoCC = frame.cc
}

func (s *Session) feedAudio(msg rtmp.AVMsg) {
	if msg.Payload[0]>>4 != 10 {
		// TODO chef: HLS音频现在只做了h264的支持
		return
	}

	if msg.Payload[1] == 0 {
		s.cacheAACSeqHeader(msg)
		return
	}

	pts := uint64(msg.Header.TimestampAbs) * 90

	s.updateFragment(pts, s.spspps == nil, 2)

	adtsHeader := s.adts.GetADTS(uint16(msg.Header.MsgLen))
	s.aframe = append(s.aframe, adtsHeader...)
	s.aframe = append(s.aframe, msg.Payload...)

	s.aframePTS = pts
}

func (s *Session) cacheAACSeqHeader(msg rtmp.AVMsg) {
	nazalog.Debug("cacheAACSeqHeader")
	s.adts.PutAACSequenceHeader(msg.Payload)
}

func (s *Session) cacheSPSPPS(msg rtmp.AVMsg) {
	nazalog.Debugf("cacheSPSPPS. %s", hex.Dump(msg.Payload))
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

func (s *Session) updateFragment(ts uint64, boundary bool, flushRate int) {
	force := false
	discont := true
	var f *Frag

	if s.opened {
		f = s.getFrag(s.nfrags)

		if (ts > s.fragTS && ts-s.fragTS > maxfraglen) || (s.fragTS > ts && s.fragTS-ts > negMaxfraglen) {
			nazalog.Warnf("hls: force fragment split: fragTS=%d, ts=%d", s.fragTS, ts)
			force = true
		} else {
			// TODO chef: 考虑ts比fragTS小的情况
			f.duration = float64(ts-s.fragTS) / 90000
			discont = false
		}
	}

	if f != nil && f.duration < fraglen/float64(1000) {
		boundary = false
	}

	if boundary || force {
		s.openFragment(ts, discont)
	}
}

func (s *Session) openFragment(ts uint64, discont bool) {
	if s.opened {
		return
	}

	s.ensureDir()
	id := s.getFragmentID()

	filename := fmt.Sprintf("%s%d.ts", outPath, id)
	s.fp = mpegtsOpenFile(filename)
	s.opened = true
	s.fragTS = ts
}

func (s *Session) closeFragment() {
	if !s.opened {
		// TODO chef: 关注下是否有这种情况
		nazalog.Assert(true, s.opened)
	}

	mpegtsCloseFile(s.fp)

	s.nextFrag()

	s.writePlaylist()

	s.opened = false
}

func (s *Session) writePlaylist() {
	// to be continued
}

func (s *Session) ensureDir() {
	err := os.MkdirAll(outPath, 0777)
	nazalog.Assert(nil, err)
}

func (s *Session) getFragmentID() int {
	return s.frag
}

func (s *Session) getFrag(n int) *Frag {
	return &s.frags[(s.frag+n)%(winfrags*2+1)]
}

// TODO chef: 这个函数重命名为incr更好些
func (s *Session) nextFrag() {
	if s.nfrags == winfrags {
		s.frag++
	} else {
		s.nfrags++
	}
}

//
// ngx_rtmp_hls_next_frag() 如果nfrags达到了winfrags，则递增frag，否则递增nfrags
// 关闭fragment时调用
