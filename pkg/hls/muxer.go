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
	"fmt"
	"os"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/avc"

	"github.com/q191201771/naza/pkg/unique"

	"github.com/q191201771/lal/pkg/aac"

	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazalog"
)

// 记录fragment的一些信息，注意，写m3u8文件时可能还需要用到历史fragment的信息
type fragmentInfo struct {
	id       int     // fragment的自增序号
	duration float64 // 当前fragment中数据的时长，单位秒
	discont  bool    // #EXT-X-DISCONTINUITY
}

type MuxerConfig struct {
	OutPath            string `json:"out_path"`
	FragmentDurationMS int    `json:"fragment_duration_ms"`
	FragmentNum        int    `json:"fragment_num"`
}

type Muxer struct {
	UniqueKey string

	streamName          string
	outPath             string
	playlistFilename    string
	playlistFilenameBak string

	config *MuxerConfig

	fragmentOP FragmentOP
	opened     bool
	adts       aac.ADTS
	spspps     []byte // AnnexB
	videoCC    uint8
	audioCC    uint8
	videoOut   []byte // 帧

	fragTS uint64 // 新建立fragment时的时间戳，毫秒 * 90

	nfrags int            // 大序号，增长到winfrags后，就增长frag
	frag   int            // 写入m3u8的EXT-X-MEDIA-SEQUENCE字段
	frags  []fragmentInfo // TS文件的环形队列，记录TS的信息，比如写M3U8文件时要用 2 * winfrags + 1

	aaframe   []byte
	aframePTS uint64 // 最新音频帧的时间戳
}

func NewMuxer(streamName string, config *MuxerConfig) *Muxer {
	uk := unique.GenUniqueKey("HLSMUXER")
	nazalog.Infof("[%s] lifecycle new hls muxer. streamName=%s", uk, streamName)

	op := getMuxerOutPath(config.OutPath, streamName)
	playlistFilename := getM3U8Filename(op, streamName)
	playlistFilenameBak := fmt.Sprintf("%s.bak", playlistFilename)
	videoOut := make([]byte, 1024*1024)
	videoOut = videoOut[0:0]
	frags := make([]fragmentInfo, 2*config.FragmentNum+1) // TODO chef: 为什么是 * 2 + 1
	return &Muxer{
		UniqueKey:           uk,
		streamName:          streamName,
		outPath:             op,
		playlistFilename:    playlistFilename,
		playlistFilenameBak: playlistFilenameBak,
		config:              config,
		videoOut:            videoOut,
		aaframe:             nil,
		frags:               frags,
	}
}

func (m *Muxer) Start() {
	nazalog.Infof("[%s] start hls muxer.", m.UniqueKey)
	m.ensureDir()
}

func (m *Muxer) Dispose() {
	nazalog.Infof("[%s] lifecycle dispose hls muxer.", m.UniqueKey)
	m.flushAudio()
	m.closeFragment(true)
}

// 函数调用结束后，内部不持有msg中的内存块
func (m *Muxer) FeedRTMPMessage(msg base.RTMPMsg) {
	switch msg.Header.MsgTypeID {
	case base.RTMPTypeIDAudio:
		m.feedAudio(msg)
	case base.RTMPTypeIDVideo:
		m.feedVideo(msg)
	}
}

// TODO chef: 可以考虑数据有问题时，返回给上层，直接主动关闭输入流的连接
func (m *Muxer) feedVideo(msg base.RTMPMsg) {
	if len(msg.Payload) < 5 {
		nazalog.Errorf("[%s] invalid video message length. len=%d", m.UniqueKey, len(msg.Payload))
		return
	}
	if msg.Payload[0]&0xF != 7 {
		// TODO chef: HLS视频现在只做了h264的支持
		return
	}

	ftype := msg.Payload[0] & 0xF0 >> 4
	htype := msg.Payload[1]

	if ftype == 1 && htype == 0 {
		if err := m.cacheSPSPPS(msg); err != nil {
			nazalog.Errorf("[%s] cache spspps failed. err=%+v", m.UniqueKey, err)
		}
		return
	}

	cts := bele.BEUint24(msg.Payload[2:])

	audSent := false
	spsppsSent := false
	// 优化这块buffer
	out := m.videoOut[0:0]
	for i := 5; i != len(msg.Payload); {
		if i+4 > len(msg.Payload) {
			nazalog.Errorf("[%s] slice len not enough. i=%d, len=%d", m.UniqueKey, i, len(msg.Payload))
			return
		}
		nalBytes := int(bele.BEUint32(msg.Payload[i:]))
		i += 4
		if i+nalBytes > len(msg.Payload) {
			nazalog.Errorf("[%s] slice len not enough. i=%d, payload len=%d, nalBytes=%d", m.UniqueKey, i, len(msg.Payload), nalBytes)
			return
		}

		nalType := avc.ParseNALUType(msg.Payload[i])

		//nazalog.Debugf("hls: h264 NAL type=%d, len=%d(%d) cts=%d.", nalType, nalBytes, len(msg.Payload), cts)

		if nalType == avc.NALUTypeSPS || nalType == avc.NALUTypePPS || nalType == avc.NALUTypeAUD {
			i += nalBytes
			continue
		}

		if !audSent {
			switch nalType {
			case avc.NALUTypeSlice, avc.NALUTypeIDRSlice, avc.NALUTypeSEI:
				out = append(out, audNal...)
				audSent = true
			case avc.NALUTypeAUD:
				audSent = true
			}
		}

		switch nalType {
		case avc.NALUTypeSlice:
			spsppsSent = false
		case avc.NALUTypeIDRSlice:
			if !spsppsSent {
				out = m.appendSPSPPS(out)
			}
			spsppsSent = true

		}

		if len(out) == 0 {
			out = append(out, avc.NALUStartCode4...)
		} else {
			out = append(out, avc.NALUStartCode3...)
		}
		out = append(out, msg.Payload[i:i+nalBytes]...)

		i += nalBytes
	}

	var frame mpegTSFrame
	frame.cc = m.videoCC
	frame.dts = uint64(msg.Header.TimestampAbs) * 90
	frame.pts = frame.dts + uint64(cts)*90
	frame.pid = PidVideo
	frame.sid = streamIDVideo
	frame.key = ftype == 1

	boundary := frame.key && (!m.opened || !m.adts.HasInited() || m.aaframe != nil)

	m.updateFragment(frame.dts, boundary, 1)

	if !m.opened {
		nazalog.Warnf("[%s] not opened.", m.UniqueKey)
		return
	}

	m.fragmentOP.WriteFrame(&frame, out)
	m.videoCC = frame.cc
}

func (m *Muxer) feedAudio(msg base.RTMPMsg) {
	if len(msg.Payload) < 3 {
		nazalog.Errorf("[%s] invalid audio message length. len=%d", m.UniqueKey, len(msg.Payload))
	}
	if msg.Payload[0]>>4 != 10 {
		return
	}

	if msg.Payload[1] == 0 {
		m.cacheAACSeqHeader(msg)
		return
	}

	if !m.adts.HasInited() {
		nazalog.Warnf("[%s] feed audio message but aac seq header not exist.", m.UniqueKey)
		return
	}

	pts := uint64(msg.Header.TimestampAbs) * 90

	m.updateFragment(pts, m.spspps == nil, 2)

	if m.aaframe == nil {
		m.aframePTS = pts
	}

	adtsHeader, _ := m.adts.CalcADTSHeader(uint16(msg.Header.MsgLen - 2))
	m.aaframe = append(m.aaframe, adtsHeader...)
	m.aaframe = append(m.aaframe, msg.Payload[2:]...)
}

func (m *Muxer) cacheAACSeqHeader(msg base.RTMPMsg) {
	_ = m.adts.InitWithAACAudioSpecificConfig(msg.Payload[2:])
}

func (m *Muxer) cacheSPSPPS(msg base.RTMPMsg) error {
	var err error
	m.spspps, err = avc.SPSPPSSeqHeader2AnnexB(msg.Payload)
	return err
}

func (m *Muxer) appendSPSPPS(out []byte) []byte {
	if m.spspps == nil {
		nazalog.Warnf("[%s] append spspps by not exist.", m.UniqueKey)
		return out
	}

	out = append(out, m.spspps...)
	return out
}

func (m *Muxer) updateFragment(ts uint64, boundary bool, flushRate int) {
	force := false
	discont := true
	var f *fragmentInfo

	// 注意，音频和视频是在一起检查的
	if m.opened {
		f = m.getFrag(m.nfrags)

		// 当前时间戳跳跃很大，或者是往回跳跃超过了阈值，强制开启新的fragment
		maxfraglen := uint64(m.config.FragmentDurationMS * 90 * 10)
		if (ts > m.fragTS && ts-m.fragTS > maxfraglen) || (m.fragTS > ts && m.fragTS-ts > negMaxfraglen) {
			nazalog.Warnf("[%s] force fragment split. fragTS=%d, ts=%d", m.UniqueKey, m.fragTS, ts)
			force = true
		} else {
			// TODO chef: 考虑ts比fragTS小的情况
			f.duration = float64(ts-m.fragTS) / 90000
			discont = false
		}
	}

	// 时长超过设置的ts文件切片阈值才行
	if f != nil && f.duration < float64(m.config.FragmentDurationMS)/1000 {
		boundary = false
	}

	// 开启新的fragment
	if boundary || force {
		m.closeFragment(false)
		m.openFragment(ts, discont)
	}

	// 音频已经缓存了一定时长的数据了，需要落盘了
	if m.opened && m.aaframe != nil && ((m.aframePTS + maxAudioDelay*90/uint64(flushRate)) < ts) {
		m.flushAudio()
	}
}

func (m *Muxer) openFragment(ts uint64, discont bool) {
	if m.opened {
		return
	}

	id := m.getFragmentID()

	filename := getTSFilename(m.outPath, m.streamName, id)
	_ = m.fragmentOP.OpenFile(filename)
	m.opened = true

	frag := m.getFrag(m.nfrags)
	frag.discont = discont
	frag.id = id

	m.fragTS = ts

	m.flushAudio()
}

func (m *Muxer) closeFragment(isLast bool) {
	if !m.opened {
		return
	}

	m.fragmentOP.CloseFile()

	m.opened = false
	//更新序号，为下个分片准备好
	m.nextFrag()

	m.writePlaylist(isLast)
}

func (m *Muxer) writePlaylist(isLast bool) {
	fp, err := os.Create(m.playlistFilenameBak)
	nazalog.Assert(nil, err)

	// 找出时长最长的fragment
	maxFrag := float64(m.config.FragmentDurationMS) / 1000
	for i := 0; i < m.nfrags; i++ {
		frag := m.getFrag(i)
		if frag.duration > maxFrag {
			maxFrag = frag.duration + 0.5
		}
	}

	// TODO chef 优化这块buffer的构造
	var buf bytes.Buffer
	buf.WriteString("#EXTM3U\n")
	buf.WriteString("#EXT-X-VERSION:3\n")
	buf.WriteString("#EXT-X-ALLOW-CACHE:NO\n")
	buf.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", int(maxFrag)))
	buf.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n\n", m.frag))

	for i := 0; i < m.nfrags; i++ {
		frag := m.getFrag(i)

		if frag.discont {
			buf.WriteString("#EXT-X-DISCONTINUITY\n")
		}

		buf.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n%s\n", frag.duration, getTSFilenameWithoutPath(m.streamName, frag.id)))
	}

	if isLast {
		buf.WriteString("#EXT-X-ENDLIST\n")
	}

	_, err = fp.Write(buf.Bytes())
	nazalog.Assert(nil, err)
	_ = fp.Close()
	err = os.Rename(m.playlistFilenameBak, m.playlistFilename)
	nazalog.Assert(nil, err)
}

// 创建文件夹，如果文件夹已经存在，老的文件夹会被删除
func (m *Muxer) ensureDir() {
	err := os.RemoveAll(m.outPath)
	nazalog.Assert(nil, err)
	err = os.MkdirAll(m.outPath, 0777)
	nazalog.Assert(nil, err)
}

func (m *Muxer) getFragmentID() int {
	return m.frag + m.nfrags
}

func (m *Muxer) getFrag(n int) *fragmentInfo {
	return &m.frags[(m.frag+n)%(m.config.FragmentNum*2+1)]
}

// TODO chef: 这个函数重命名为incr更好些
func (m *Muxer) nextFrag() {
	if m.nfrags == m.config.FragmentNum {
		m.frag++
	} else {
		m.nfrags++
	}
}

// 将音频数据落盘的几种情况：
// 1. open fragment时，如果aframe中还有数据
// 2. update fragment时，判断音频的时间戳
// 3. 音频队列长度过长时
// 4. 流关闭时
func (m *Muxer) flushAudio() {
	if !m.opened {
		nazalog.Warnf("[%s] flushAudio by not opened.", m.UniqueKey)
		return
	}

	if m.aaframe == nil {
		return
	}

	frame := &mpegTSFrame{
		pts: m.aframePTS,
		dts: m.aframePTS,
		pid: PidAudio,
		sid: streamIDAudio,
		cc:  m.audioCC,
		key: false,
	}

	m.fragmentOP.WriteFrame(frame, m.aaframe)

	m.audioCC = frame.cc
	m.aaframe = nil
}
