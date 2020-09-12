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
	"io/ioutil"
	"os"
	"time"

	"github.com/q191201771/lal/pkg/mpegts"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/unique"

	"github.com/q191201771/naza/pkg/nazalog"
)

// TODO chef: 转换TS流的功能（通过回调供httpts使用）也放在了Muxer中，好处是hls和httpts可以共用一份TS流。
//            后续从架构上考虑，packet hls,mpegts,logic的分工

type MuxerObserver interface {
	// @param rawFrame TS流，回调结束后，内部不再使用该内存块
	// @param boundary 新的TS流接收者，应该从该标志为true时开始发送数据
	//
	OnTSPackets(rawFrame []byte, boundary bool)
}

type MuxerConfig struct {
	Enable             bool   `json:"enable"`   // 如果false，说明hls功能没开，也即不写磁盘，但是MuxerObserver依然会回调
	OutPath            string `json:"out_path"` // m3u8和ts文件的输出根目录，注意，末尾需已'/'结束
	FragmentDurationMS int    `json:"fragment_duration_ms"`
	FragmentNum        int    `json:"fragment_num"`
}

type Muxer struct {
	UniqueKey string

	streamName                string
	outPath                   string
	playlistFilename          string
	playlistFilenameBak       string
	recordPlayListFilename    string
	recordPlayListFilenameBak string

	config   *MuxerConfig
	observer MuxerObserver

	fragment Fragment
	opened   bool
	videoCC  uint8
	audioCC  uint8

	fragTS                uint64         // 新建立fragment时的时间戳，毫秒 * 90
	nfrags                int            // 大序号，增长到winfrags后，就增长frag
	frag                  int            // 写入m3u8的EXT-X-MEDIA-SEQUENCE字段
	frags                 []fragmentInfo // TS文件的环形队列，记录TS的信息，比如写M3U8文件时要用 2 * winfrags + 1
	recordMaxFragDuration float64

	streamer *Streamer
}

// 记录fragment的一些信息，注意，写m3u8文件时可能还需要用到历史fragment的信息
type fragmentInfo struct {
	id       int     // fragment的自增序号
	duration float64 // 当前fragment中数据的时长，单位秒
	discont  bool    // #EXT-X-DISCONTINUITY
	filename string
}

// @param observer 可以为nil，如果不为nil，TS流将回调给上层
func NewMuxer(streamName string, config *MuxerConfig, observer MuxerObserver) *Muxer {
	uk := unique.GenUniqueKey("HLSMUXER")
	op := getMuxerOutPath(config.OutPath, streamName)
	playlistFilename := getM3U8Filename(op, streamName)
	playlistFilenameBak := fmt.Sprintf("%s.bak", playlistFilename)
	recordPlaylistFilename := getRecordM3U8Filename(op, streamName)
	recordPlaylistFilenameBak := fmt.Sprintf("%s.bak", recordPlaylistFilename)
	frags := make([]fragmentInfo, 2*config.FragmentNum+1) // TODO chef: 为什么是 * 2 + 1
	m := &Muxer{
		UniqueKey:                 uk,
		streamName:                streamName,
		outPath:                   op,
		playlistFilename:          playlistFilename,
		playlistFilenameBak:       playlistFilenameBak,
		recordPlayListFilename:    recordPlaylistFilename,
		recordPlayListFilenameBak: recordPlaylistFilenameBak,
		config:   config,
		observer: observer,
		frags:    frags,
	}
	streamer := NewStreamer(m)
	m.streamer = streamer
	nazalog.Infof("[%s] lifecycle new hls muxer. muxer=%p, streamName=%s", uk, m, streamName)
	return m
}

func (m *Muxer) Start() {
	nazalog.Infof("[%s] start hls muxer.", m.UniqueKey)
	m.ensureDir()
}

func (m *Muxer) Dispose() {
	nazalog.Infof("[%s] lifecycle dispose hls muxer.", m.UniqueKey)
	m.streamer.FlushAudio()
	if err := m.closeFragment(true); err != nil {
		nazalog.Errorf("[%s] close fragment error. err=%+v", m.UniqueKey, err)
	}
}

// @param msg 函数调用结束后，内部不持有msg中的内存块
//
func (m *Muxer) FeedRTMPMessage(msg base.RTMPMsg) {
	m.streamer.FeedRTMPMessage(msg)
}

func (m *Muxer) OnFrame(streamer *Streamer, frame *mpegts.Frame) {
	var boundary bool
	var packets []byte

	if frame.Sid == mpegts.StreamIDAudio {
		// 为了考虑没有视频的情况也能切片，所以这里判断spspps为空时，也建议生成fragment
		boundary = !streamer.VideoSeqHeaderCached()
		if err := m.updateFragment(frame.PTS, boundary); err != nil {
			nazalog.Errorf("[%s] update fragment error. err=%+v", m.UniqueKey, err)
			return
		}

		if !m.opened {
			nazalog.Warnf("[%s] OnFrame A not opened.", m.UniqueKey)
			return
		}

		//nazalog.Debugf("[%s] WriteFrame A. dts=%d, len=%d", m.UniqueKey, frame.DTS, len(frame.Raw))
	} else {
		// 收到视频，可能触发建立fragment的条件是：
		// 关键帧数据 &&
		// ((没有收到过音频seq header) || -> 只有视频
		//  (收到过音频seq header && fragment没有打开) || -> 音视频都有，且都已ready
		//  (收到过音频seq header && fragment已经打开 && 音频缓存数据不为空) -> 为什么音频缓存需不为空？
		// )
		boundary = frame.Key && (!streamer.AudioSeqHeaderCached() || !m.opened || !streamer.AudioCacheEmpty())
		if err := m.updateFragment(frame.DTS, boundary); err != nil {
			nazalog.Errorf("[%s] update fragment error. err=%+v", m.UniqueKey, err)
			return
		}

		if !m.opened {
			nazalog.Warnf("[%s] OnFrame V not opened.", m.UniqueKey)
			return
		}

		//nazalog.Debugf("[%s] WriteFrame V. dts=%d, len=%d", m.UniqueKey, frame.DTS, len(frame.Raw))
	}

	mpegts.PackTSPacket(frame, func(packet []byte) {
		if m.config.Enable {
			if err := m.fragment.WriteFile(packet); err != nil {
				nazalog.Errorf("[%s] fragment write error. err=%+v", m.UniqueKey, err)
				return
			}
		}
		if m.observer != nil {
			packets = append(packets, packet...)
		}
	})
	if m.observer != nil {
		m.observer.OnTSPackets(packets, boundary)
	}
}

// 决定是否开启新的TS切片文件（注意，可能已经有TS切片，也可能没有，这是第一个切片）
//
// @param boundary 调用方认为可能是开启新TS切片的时间点
//
func (m *Muxer) updateFragment(ts uint64, boundary bool) error {
	discont := true

	// 如果已经有TS切片，检查是否需要强制开启新的切片，以及切片是否发生跳跃
	// 注意，音频和视频是在一起检查的
	if m.opened {
		f := m.getCurrFrag()

		// 当前时间戳跳跃很大，或者是往回跳跃超过了阈值，强制开启新的fragment
		maxfraglen := uint64(m.config.FragmentDurationMS * 90 * 10)
		if (ts > m.fragTS && ts-m.fragTS > maxfraglen) || (m.fragTS > ts && m.fragTS-ts > negMaxfraglen) {
			nazalog.Warnf("[%s] force fragment split. fragTS=%d, ts=%d", m.UniqueKey, m.fragTS, ts)

			if err := m.closeFragment(false); err != nil {
				return err
			}
			if err := m.openFragment(ts, true); err != nil {
				return err
			}
		}

		// 更新当前分片的时间长度
		//
		// TODO chef:
		// f.duration（也即写入m3u8中记录分片时间长度）的做法我觉得有问题
		// 此处用最新收到的数据更新f.duration
		// 但是假设fragment翻滚，数据可能是写入下一个分片中
		// 是否就导致了f.duration和实际分片时间长度不一致
		if ts > m.fragTS {
			duration := float64(ts-m.fragTS) / 90000
			if duration > f.duration {
				f.duration = duration
			}
		}

		discont = false

		// 已经有TS切片，切片时长没有达到设置的阈值，则不开启新的切片
		if f.duration < float64(m.config.FragmentDurationMS)/1000 {
			return nil
		}
	}

	// 开启新的fragment
	if boundary {
		if err := m.closeFragment(false); err != nil {
			return err
		}
		if err := m.openFragment(ts, discont); err != nil {
			return err
		}
	}

	return nil
}

// @param discont 不连续标志，会在m3u8文件的fragment前增加`#EXT-X-DISCONTINUITY`
//
func (m *Muxer) openFragment(ts uint64, discont bool) error {
	if m.opened {
		return ErrHLS
	}

	id := m.getFragmentID()

	filename := getTSFilename(m.streamName, id, int(time.Now().Unix()))
	filenameWithPath := getTSFilenameWithPath(m.outPath, filename)
	if m.config.Enable {
		if err := m.fragment.OpenFile(filenameWithPath); err != nil {
			return err
		}
	}
	m.opened = true

	frag := m.getCurrFrag()
	frag.discont = discont
	frag.id = id
	frag.filename = filename
	frag.duration = 0

	m.fragTS = ts

	// nrm said: start fragment with audio to make iPhone happy
	m.streamer.FlushAudio()

	return nil
}

func (m *Muxer) closeFragment(isLast bool) error {
	if !m.opened {
		// 注意，首次调用closeFragment时，有可能opened为false
		return nil
	}

	if m.config.Enable {
		if err := m.fragment.CloseFile(); err != nil {
			return err
		}
	}

	m.opened = false
	//更新序号，为下个分片准备好
	m.incrFrag()

	m.writePlaylist(isLast)
	m.writeRecordPlaylist(isLast)

	return nil
}

func (m *Muxer) writeRecordPlaylist(isLast bool) {
	if !m.config.Enable {
		return
	}

	//frag := m.getCurrFrag()
	currFrag := m.getFrag(m.nfrags - 1)
	if currFrag.duration > m.recordMaxFragDuration {
		m.recordMaxFragDuration = currFrag.duration + 0.5
	}

	fragLines := fmt.Sprintf("#EXTINF:%.3f,\n%s\n", currFrag.duration, currFrag.filename)

	content, err := ioutil.ReadFile(m.recordPlayListFilename)
	if err == nil {
		// m3u8文件已经存在

		content = bytes.TrimSuffix(content, []byte("#EXT-X-ENDLIST\n"))
		content, err = updateTargetDurationInM3U8(content, int(m.recordMaxFragDuration))
		if err != nil {
			nazalog.Errorf("[%s] update target duration failed. err=%+v", m.UniqueKey, err)
			return
		}

		if currFrag.discont {
			content = append(content, []byte("#EXT-X-DISCONTINUITY\n")...)
		}

		content = append(content, []byte(fragLines)...)
		content = append(content, []byte("#EXT-X-ENDLIST\n")...)
	} else {
		// m3u8文件不存在
		var buf bytes.Buffer
		buf.WriteString("#EXTM3U\n")
		buf.WriteString("#EXT-X-VERSION:3\n")
		buf.WriteString(fmt.Sprintf("#EXT-X-TARGETDURATION:%d\n", int(m.recordMaxFragDuration)))
		buf.WriteString(fmt.Sprintf("#EXT-X-MEDIA-SEQUENCE:%d\n\n", 0))

		if currFrag.discont {
			buf.WriteString("#EXT-X-DISCONTINUITY\n")
		}

		buf.WriteString(fragLines)
		buf.WriteString("#EXT-X-ENDLIST\n")

		content = buf.Bytes()
	}

	if err := writeM3U8File(content, m.recordPlayListFilename, m.recordPlayListFilenameBak); err != nil {
		nazalog.Errorf("[%s] write record m3u8 file error. err=%+v", m.UniqueKey, err)
	}
}

func (m *Muxer) writePlaylist(isLast bool) {
	if !m.config.Enable {
		return
	}

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

		buf.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n%s\n", frag.duration, frag.filename))
	}

	if isLast {
		buf.WriteString("#EXT-X-ENDLIST\n")
	}

	if err := writeM3U8File(buf.Bytes(), m.playlistFilename, m.playlistFilenameBak); err != nil {
		nazalog.Errorf("[%s] write live m3u8 file error. err=%+v", m.UniqueKey, err)
	}
}

func (m *Muxer) ensureDir() {
	if !m.config.Enable {
		return
	}
	//err := os.RemoveAll(m.outPath)
	//nazalog.Assert(nil, err)
	err := os.MkdirAll(m.outPath, 0777)
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
