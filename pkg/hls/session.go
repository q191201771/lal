// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import (
	"bytes"
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
	discont  bool // #EXT-X-DISCONTINUITY
}

type Session struct {
	streamName          string
	playlistFilename    string
	playlistFilenameBak string

	adts aac.ADTS
	//aacSeqHeader []byte
	spspps   []byte
	videoCC  uint8
	audioCC  uint8
	opened   bool
	videoOut []byte // 帧
	fp       *os.File

	fragTS uint64 // 新建立fragment时的时间戳，毫秒 * 90

	nfrags int    // 大序号，增长到winfrags后，就增长frag
	frag   int    // 写入m3u8的EXT-X-MEDIA-SEQUENCE字段
	frags  []Frag // TS文件的环形队列，记录TS的信息，比如写M3U8文件时要用 2 * winfrags + 1

	aaframe []byte
	//aframeBase uint64 // 上一个音频帧的时间戳
	//aframeNum  uint64
	aframePTS uint64 // 最新音频帧的时间戳
}

func NewSession(streamName string) *Session {
	playlistFilename := fmt.Sprintf("%s%s.m3u8", outPath, streamName)
	playlistFilenameBak := fmt.Sprintf("%s.bak", playlistFilename)
	videoOut := make([]byte, 1024*1024)
	videoOut = videoOut[0:0]
	frags := make([]Frag, 2*winfrags+1) // TODO chef: 为什么是 * 2 + 1
	return &Session{
		videoOut:            videoOut,
		aaframe:             nil,
		frags:               frags,
		streamName:          streamName,
		playlistFilename:    playlistFilename,
		playlistFilenameBak: playlistFilenameBak,
	}
}

func (s *Session) Start() {

}

func (s *Session) Stop() {
	s.flushAudio()
	s.closeFragment()
}

func (s *Session) FeedRTMPMessage(msg rtmp.AVMsg) {
	// TODO chef: to be continued
	// HLS还没有开发完
	return
	switch msg.Header.MsgTypeID {
	case rtmp.TypeidAudio:
		s.feedAudio(msg)
	case rtmp.TypeidVideo:
		s.feedVideo(msg)
	}
}

func (s *Session) feedVideo(msg rtmp.AVMsg) {
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

		//nazalog.Debugf("hls: h264 NAL type=%d, len=%d(%d) cts=%d.", nalType, nalBytes, len(msg.Payload), cts)

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
	frame.pid = PidVideo
	frame.sid = streamIDVideo
	frame.key = ftype == 1

	boundary := frame.key && (!s.opened || s.adts.IsNil() || s.aaframe != nil)

	s.updateFragment(frame.dts, boundary, 1)

	if !s.opened {
		nazalog.Warn("not opened.")
		return
	}

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

	if s.aaframe == nil {
		s.aframePTS = pts
	}

	adtsHeader := s.adts.GetADTS(uint16(msg.Header.MsgLen))
	s.aaframe = append(s.aaframe, adtsHeader...)
	s.aaframe = append(s.aaframe, msg.Payload[2:]...)

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

		// 当前时间戳跳跃很大，或者是往回跳跃超过了阈值，强制开启新的fragment
		if (ts > s.fragTS && ts-s.fragTS > maxfraglen) || (s.fragTS > ts && s.fragTS-ts > negMaxfraglen) {
			nazalog.Warnf("hls: force fragment split: fragTS=%d, ts=%d", s.fragTS, ts)
			force = true
		} else {
			// TODO chef: 考虑ts比fragTS小的情况
			f.duration = float64(ts-s.fragTS) / 90000
			discont = false
		}
	}

	// 时长超过设置的ts文件切片阈值才行
	if f != nil && f.duration < fraglen/float64(1000) {
		boundary = false
	}

	// 开启新的fragment
	if boundary || force {
		s.closeFragment()
		s.openFragment(ts, discont)
	}

	// 音频已经缓存了一定时长的数据了，需要落盘了
	//nazalog.Debugf("CHEFERASEME 05191839, flush_rate=%d, size=%d, aframe_pts=%d, ts=%d",
	//	flushRate, len(s.aaframe), s.aframePTS, ts)
	if s.opened && s.aaframe != nil && ((s.aframePTS + maxAudioDelay*90/uint64(flushRate)) < ts) {
		//nazalog.Debugf("CHEFERASEME 05191839.")
		s.flushAudio()
	}
}

func (s *Session) openFragment(ts uint64, discont bool) {
	if s.opened {
		return
	}

	s.ensureDir()
	id := s.getFragmentID()

	filename := fmt.Sprintf("%s%s-%d.ts", outPath, s.streamName, id)
	s.fp = mpegtsOpenFile(filename)
	s.opened = true

	frag := s.getFrag(s.nfrags)
	frag.active = true
	frag.discont = discont
	frag.id = uint64(id)

	s.fragTS = ts

	s.flushAudio()
}

func (s *Session) closeFragment() {
	if !s.opened {
		return
	}

	mpegtsCloseFile(s.fp)

	s.opened = false

	s.nextFrag()

	s.writePlaylist()

}

func (s *Session) writePlaylist() {
	fp, err := os.Create(s.playlistFilenameBak)
	nazalog.Assert(nil, err)

	// 找出时长最长的fragment
	maxFrag := float64(fraglen / 1000)
	for i := 0; i < s.nfrags; i++ {
		frag := s.getFrag(i)
		if frag.duration > maxFrag {
			maxFrag = frag.duration + 0.5
		}
	}

	// TODO chef 优化这块buffer的构造
	var buf bytes.Buffer
	buf.WriteString("#EXTM3U\n")
	buf.WriteString("#EXT-X-VERSION:3\n")
	buf.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n", s.frag))
	buf.WriteString(fmt.Sprintf("#EXT-X-TARGETRATION:%d\n", int(maxFrag)))

	for i := 0; i < s.nfrags; i++ {
		frag := s.getFrag(i)

		if frag.discont {
			buf.WriteString("#EXT-X-DISCONTINUITY\n")
		}

		buf.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n%s-%d.ts\n", frag.duration, s.streamName, frag.id))
	}

	_, err = fp.Write(buf.Bytes())
	nazalog.Assert(nil, err)
	_ = fp.Close()
	err = os.Rename(s.playlistFilenameBak, s.playlistFilename)
	nazalog.Assert(nil, err)
}

func (s *Session) ensureDir() {
	err := os.MkdirAll(outPath, 0777)
	nazalog.Assert(nil, err)
}

func (s *Session) getFragmentID() int {
	return s.frag + s.nfrags
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

// 将音频数据落盘的几种情况：
// 1. open fragment时，如果aframe中还有数据
// 2. update fragment时，判断音频的时间戳
// 3. 音频队列长度过长时
// 4. 流关闭时
func (s *Session) flushAudio() {
	if !s.opened {
		nazalog.Warn("flushAudio by not opened.")
		return
	}

	if s.aaframe == nil {
		nazalog.Warn("flushAudio by aframe is nil.")
		return
	}

	frame := &MPEGTSFrame{
		pts: s.aframePTS,
		dts: s.aframePTS,
		pid: PidAudio,
		sid: streamIDAudio,
		cc:  s.audioCC,
		key: false,
	}

	mpegtsWriteFrame(s.fp, frame, s.aaframe)

	s.audioCC = frame.cc
	s.aaframe = nil
}
