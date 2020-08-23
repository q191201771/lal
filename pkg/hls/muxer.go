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

	"github.com/q191201771/lal/pkg/mpegts"

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

	fragment Fragment
	opened   bool
	adts     aac.ADTS
	spspps   []byte // AnnexB
	videoCC  uint8
	audioCC  uint8
	videoOut []byte // 帧

	fragTS uint64 // 新建立fragment时的时间戳，毫秒 * 90

	nfrags int            // 大序号，增长到winfrags后，就增长frag
	frag   int            // 写入m3u8的EXT-X-MEDIA-SEQUENCE字段
	frags  []fragmentInfo // TS文件的环形队列，记录TS的信息，比如写M3U8文件时要用 2 * winfrags + 1

	audioCacheFrames        []byte // 缓存音频帧数据，注意，可能包含多个音频帧
	audioCacheFirstFramePTS uint64 // audioCacheFrames中第一个音频帧的时间戳
}

func NewMuxer(streamName string, config *MuxerConfig) *Muxer {
	uk := unique.GenUniqueKey("HLSMUXER")
	op := getMuxerOutPath(config.OutPath, streamName)
	playlistFilename := getM3U8Filename(op, streamName)
	playlistFilenameBak := fmt.Sprintf("%s.bak", playlistFilename)
	videoOut := make([]byte, 1024*1024)
	videoOut = videoOut[0:0]
	frags := make([]fragmentInfo, 2*config.FragmentNum+1) // TODO chef: 为什么是 * 2 + 1
	m := &Muxer{
		UniqueKey:           uk,
		streamName:          streamName,
		outPath:             op,
		playlistFilename:    playlistFilename,
		playlistFilenameBak: playlistFilenameBak,
		config:              config,
		videoOut:            videoOut,
		audioCacheFrames:    nil,
		frags:               frags,
	}
	nazalog.Infof("[%s] lifecycle new hls muxer. muxer=%p, streamName=%s", uk, m, streamName)
	return m
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

// @param msg 函数调用结束后，内部不持有msg中的内存块
//
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

	// 将数据转换成AnnexB

	// 如果是sps pps，缓存住，然后直接返回
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

	// tag中可能有多个NALU，逐个获取
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

		nazalog.Debugf("[%s] hls: h264 NAL type=%d, len=%d(%d) cts=%d.", m.UniqueKey, nalType, nalBytes, len(msg.Payload), cts)

		// sps pps前面已经缓存过了，这里就不用处理了
		// aud有自己的生产逻辑，原流中的aud直接过滤掉
		if nalType == avc.NALUTypeSPS || nalType == avc.NALUTypePPS || nalType == avc.NALUTypeAUD {
			i += nalBytes
			continue
		}

		if !audSent {
			switch nalType {
			case avc.NALUTypeSlice, avc.NALUTypeIDRSlice, avc.NALUTypeSEI:
				// 在前面写入aud
				out = append(out, audNal...)
				audSent = true
				//case avc.NALUTypeAUD:
				//	// 上面aud已经continue跳过了，应该进不到这个分支，可以考虑删除这个分支代码
				//	audSent = true
			}
		}

		switch nalType {
		case avc.NALUTypeSlice:
			spsppsSent = false
		case avc.NALUTypeIDRSlice:
			// 如果是关键帧，在前面写入sps pps
			if !spsppsSent {
				out = m.appendSPSPPS(out)
			}
			spsppsSent = true

		}

		// 这里不知为什么要区分写入两种类型的start code
		if len(out) == 0 {
			out = append(out, avc.NALUStartCode4...)
		} else {
			out = append(out, avc.NALUStartCode3...)
		}

		out = append(out, msg.Payload[i:i+nalBytes]...)

		i += nalBytes
	}

	key := ftype == 1
	dts := uint64(msg.Header.TimestampAbs) * 90

	boundary := key && (!m.opened || !m.adts.HasInited() || m.audioCacheFrames != nil)

	m.updateFragment(dts, boundary, false)

	if !m.opened {
		nazalog.Warnf("[%s] not opened.", m.UniqueKey)
		return
	}

	var frame mpegts.Frame
	frame.CC = m.videoCC
	frame.DTS = dts
	frame.PTS = frame.DTS + uint64(cts)*90
	frame.Key = key
	frame.Raw = out
	frame.Pid = mpegts.PidVideo
	frame.Sid = mpegts.StreamIDVideo
	nazalog.Debugf("[%s] WriteFrame V. dts=%d, len=%d", m.UniqueKey, frame.DTS, len(frame.Raw))
	m.fragment.WriteFrame(&frame)
	m.videoCC = frame.CC
}

func (m *Muxer) feedAudio(msg base.RTMPMsg) {
	if len(msg.Payload) < 3 {
		nazalog.Errorf("[%s] invalid audio message length. len=%d", m.UniqueKey, len(msg.Payload))
	}
	if msg.Payload[0]>>4 != 10 {
		return
	}

	nazalog.Debugf("[%s] hls: feedAudio. dts=%d len=%d", m.UniqueKey, msg.Header.TimestampAbs, len(msg.Payload))

	if msg.Payload[1] == 0 {
		m.cacheAACSeqHeader(msg)
		return
	}

	if !m.adts.HasInited() {
		nazalog.Warnf("[%s] feed audio message but aac seq header not exist.", m.UniqueKey)
		return
	}

	pts := uint64(msg.Header.TimestampAbs) * 90

	m.updateFragment(pts, m.spspps == nil, true)

	if m.audioCacheFrames == nil {
		m.audioCacheFirstFramePTS = pts
	}

	adtsHeader, _ := m.adts.CalcADTSHeader(uint16(msg.Header.MsgLen - 2))
	m.audioCacheFrames = append(m.audioCacheFrames, adtsHeader...)
	m.audioCacheFrames = append(m.audioCacheFrames, msg.Payload[2:]...)
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

// 决定是否开启新的TS切片文件（注意，可能已经有TS切片，也可能没有，这是第一个切片），以及落盘音频数据
//
// @param boundary  调用方认为可能是开启新TS切片的时间点
// @param isByAudio 触发该函数调用，是因为收到音频数据，还是视频数据
//
func (m *Muxer) updateFragment(ts uint64, boundary bool, isByAudio bool) {
	force := false
	discont := true
	var f *fragmentInfo

	// 如果已经有TS切片，检查是否需要强制开启新的切片，以及切片是否发生跳跃
	// 注意，音频和视频是在一起检查的
	if m.opened {
		f = m.getCurrFrag()

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

		// 已经有TS切片，那么只有当前fragment的时长超过设置的TS切片阈值才开启新的切片
		if f.duration < float64(m.config.FragmentDurationMS)/1000 {
			boundary = false
		}
	}

	// 开启新的fragment
	if boundary || force {
		m.closeFragment(false)
		m.openFragment(ts, discont)
	}

	// 音频已经缓存了一定时长的数据了，需要落盘了
	var maxAudioDelay uint64
	if isByAudio {
		maxAudioDelay = maxAudioCacheDelayByAudio
	} else {
		maxAudioDelay = maxAudioCacheDelayByVideo
	}
	if m.opened && m.audioCacheFrames != nil && ((m.audioCacheFirstFramePTS + maxAudioDelay) < ts) {
		m.flushAudio()
	}
}

// @param discont 不连续标志，会在m3u8文件的fragment前增加`#EXT-X-DISCONTINUITY`
func (m *Muxer) openFragment(ts uint64, discont bool) {
	if m.opened {
		return
	}

	id := m.getFragmentID()

	filename := getTSFilename(m.outPath, m.streamName, id)
	_ = m.fragment.OpenFile(filename)
	m.opened = true

	frag := m.getCurrFrag()
	frag.discont = discont
	frag.id = id

	m.fragTS = ts

	m.flushAudio()
}

func (m *Muxer) closeFragment(isLast bool) {
	if !m.opened {
		return
	}

	m.fragment.CloseFile()

	m.opened = false
	//更新序号，为下个分片准备好
	m.incrFrag()

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

func (m *Muxer) getCurrFrag() *fragmentInfo {
	return m.getFrag(m.nfrags)
}

func (m *Muxer) incrFrag() {
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

	if m.audioCacheFrames == nil {
		return
	}

	var frame mpegts.Frame
	frame.CC = m.audioCC
	frame.DTS = m.audioCacheFirstFramePTS
	frame.PTS = m.audioCacheFirstFramePTS
	frame.Key = false
	frame.Raw = m.audioCacheFrames
	frame.Pid = mpegts.PidAudio
	frame.Sid = mpegts.StreamIDAudio
	nazalog.Debugf("[%s] WriteFrame A. dts=%d, len=%d", m.UniqueKey, frame.DTS, len(frame.Raw))
	m.fragment.WriteFrame(&frame)
	m.audioCC = frame.CC

	m.audioCacheFrames = nil
}
