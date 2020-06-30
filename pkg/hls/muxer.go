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

	"github.com/q191201771/naza/pkg/unique"

	"github.com/q191201771/lal/pkg/aac"

	"github.com/q191201771/lal/pkg/rtmp"
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
	spspps     []byte
	videoCC    uint8 //视频连续计数器
	audioCC    uint8 //音频连续计数器
	videoOut   []byte // 帧

	fragTS uint64 // 新建立fragment时的时间戳，毫秒 * 90

	nfrags int            //1~config.FragmentNum(winfrags)，增长到winfrags后就不变了，新分片就增长frag了
	frag   int            // 写入m3u8的EXT-X-MEDIA-SEQUENCE字段
	frags  []fragmentInfo // TS文件的环形队列，记录TS的信息，比如写M3U8文件时要用 2 * winfrags + 1

	aaframe   []byte
	aframePTS uint64 // 最新音频帧的时间戳
}

func NewMuxer(streamName string, config *MuxerConfig) *Muxer {
	uk := unique.GenUniqueKey("HLSMUXER")
	nazalog.Infof("lifecycle new hls muxer. [%s] streamName=%s", uk, streamName)

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
	nazalog.Infof("start hls muxer. [%s]", m.UniqueKey)
	m.ensureDir()
}

func (m *Muxer) Dispose() {
	nazalog.Infof("lifecycle dispose hls muxer. [%s]", m.UniqueKey)
	m.flushAudio()
	m.closeFragment()
}

func (m *Muxer) FeedRTMPMessage(msg rtmp.AVMsg) {
	switch msg.Header.MsgTypeID {
	case rtmp.TypeidAudio:
		m.feedAudio(msg)
	case rtmp.TypeidVideo:
		m.feedVideo(msg)
	}
}

// TODO chef: 可以考虑数据有问题时，返回给上层，直接主动关闭输入流的连接
func (m *Muxer) feedVideo(msg rtmp.AVMsg) {
	if len(msg.Payload) < 5 {
		nazalog.Errorf("invalid video message length. [%s] len=%d", m.UniqueKey, len(msg.Payload))
		return
	}
	/*
	@see: E.4.3 Video Tags, video_file_format_spec_v10_1.pdf, page 78
	Frame Type, Type of video frame.
	CodecID, Codec Identifier.
	set the rtmp header
	*/
	if msg.Payload[0]&0xF != 7 {
		// TODO chef: HLS视频现在只做了h264的支持
		return
	}

	/*
	Frame Type
	Type of video frame. The following values are defined:
	1 = key frame (for AVC, a seekable frame)
	2 = inter frame (for AVC, a non-seekable frame)
	3 = disposable inter frame (H.263 only)
	4 = generated key frame (reserved for server use only)
	5 = video info/command frame
	*/
	ftype := msg.Payload[0] & 0xF0 >> 4
	/*
	AVCPacketType
	0 = AVC sequence header
	1 = AVC NALU
	2 = AVC end of sequence (lower level NALU sequence ender is
	 */
	htype := msg.Payload[1]

	//当前msg是SPS_PPS缓存它们
	if ftype == 1 && htype == 0 {
		m.cacheSPSPPS(msg)
		return
	}
	//Composition time offset
	cts := bele.BEUint24(msg.Payload[2:])

	audSent := false
	spsppsSent := false
	// 优化这块buffer
	out := m.videoOut[0:0]
	/*
	一个msg里面可能会有多个NAL单元数据(一个ES + SEI/SPS_PPS)或多个ES帧数据(这种应该算是一种错误)
	正常情况一个msg只会包括一个ES帧I/P/B或一个NAL单元数据或一个SEI + 一个ES
	不会有msg=SPS_PPS+ES这种情况,因为它们专门msg传递
	注:
		如果要兼容多个ES帧在一个msg中这种情况,TS封装标准没有禁止多个帧封装到一个PES段中
		但是如果这么做会出现这几个帧就会使用相同的PTS/DTS,至于终端能否正常显示就看终端的兼容性了。
		或者考虑把这个msg中的多个帧分开封装，但是需要去调整PTS/DTS(要不然这几个帧还是相同的PTS/DTS)
	 */
	for i := 5; i != len(msg.Payload); {
		if i+4 > len(msg.Payload) {
			nazalog.Errorf("slice len not enough. [%s] i=%d, len=%d", m.UniqueKey, i, len(msg.Payload))
			return
		}
		/*
		NALUnitLength
		5.3.4.2.1 Syntax, ISO_IEC_14496-15-AVC-format-2012.pdf, page 16
		lengthSizeMinusOne, or NAL_unit_length, always use 4bytes size
		mux the avc NALU in "ISO Base Media File Format"
		from ISO_IEC_14496-15-AVC-format-2012.pdf, page 20
		*/
		nalBytes := int(bele.BEUint32(msg.Payload[i:]))
		i += 4
		if i+nalBytes > len(msg.Payload) {
			nazalog.Errorf("slice len not enough. [%s] i=%d, payload len=%d, nalBytes=%d", m.UniqueKey, i, len(msg.Payload), nalBytes)
			return
		}
		/*
		msg.Payload[9],这里开始是NAL单元数据(无pic_start_code)
		nal_unit_type，NAL单元语法见“H.264官方中文版.pdf”pg54
		 */
		srcNalType := msg.Payload[i]
		nalType := srcNalType & 0x1F
		//nazalog.Debugf("hls: h264 NAL type=%d, len=%d(%d) cts=%d.", nalType, nalBytes, len(msg.Payload), cts)

		/*
		如果NAL单元是sps_pps_aud则跳过处理
		因为前面m.cacheSPSPPS(msg)有专门的msg传递sps_pps且已经缓存了
		 */
		if nalType >= 7 && nalType <= 9 {
			//nazalog.Warn("should not reach here.")
			i += nalBytes
			continue
		}

		/*
		在P/IDR/SEI前面插入AUD数据，每个MSG中如果有多个NAL单元保证只插入一次AUD
		nal_unit_type
		0 = B SLICE types
		1 = P SLICE types
		5 = IDR SLICE types
		6 = SEI types
		7 = SPS types
		8 = PPS types
		*/
		if !audSent {
			switch nalType {
			case 1, 5, 6:
				out = append(out, audNal...)
				audSent = true
			case 9:
				//前面AUD数据已经跳过了
				audSent = true
			}
		}
		/*
		每个IDR帧前插入SPS_PPS，如果一个MSG中多个I帧则每个都需要插入
		*/
		switch nalType {
		case 1:
			spsppsSent = false //I后面有P，P后面还有I，继续插入
		case 5:
			if !spsppsSent {
				out = m.appendSPSPPS(out)
			}
			spsppsSent = true
		}

		//添加pic_start_code
		if len(out) == 0 {
			//其它未知NAL单元数据也保存，正常情况不会到这里
			out = append(out, nalStartCode...)
		} else {
			out = append(out, nalStartCode3...)
		}
		//保存NAL单元数据
		out = append(out, msg.Payload[i:i+nalBytes]...)
		//下一个NAL单元
		i += nalBytes
	}

	var frame mpegTSFrame
	frame.cc = m.videoCC
	//rmtp_timestamp单位是ms,TS的pts/dts单位是90k时钟,dts=timestamp*1000/90000
	frame.dts = uint64(msg.Header.TimestampAbs) * 90
	frame.pts = frame.dts + uint64(cts)*90
	frame.pid = PidVideo
	frame.sid = streamIDVideo
	frame.key = ftype == 1

	/*
	程序刚运行还未创建第1个分片文件时，等到I帧来时boundary=true让updateFragment中去创建第1个分片
	如果先来的是非I帧的其它数据则丢弃(因为updateFragment中不会创建分片)
	*/
	boundary := frame.key && (!m.opened || m.adts.IsNil() || m.aaframe != nil)
	//更新分片信息和playlist文件
	m.updateFragment(frame.dts, boundary, 1)

	if !m.opened {
		nazalog.Warnf("not opened. [%s]", m.UniqueKey)
		return
	}
	//一个ES或一个SEI+ES或多个ES,封装成TS,并写入当前分片文件
	m.fragmentOP.WriteFrame(&frame, out)
	//保存WriteFrame中修改的连续计数器给下次调用WriteFrame时使用
	m.videoCC = frame.cc
}

func (m *Muxer) feedAudio(msg rtmp.AVMsg) {
	if len(msg.Payload) < 3 {
		nazalog.Errorf("invalid audio message length. [%s] len=%d", m.UniqueKey, len(msg.Payload))
	}
	if msg.Payload[0]>>4 != 10 {
		return
	}

	if msg.Payload[1] == 0 {
		m.cacheAACSeqHeader(msg)
		return
	}

	if m.adts.IsNil() {
		nazalog.Warnf("feed audio message but aac seq header not exist. [%s]", m.UniqueKey)
		return
	}

	pts := uint64(msg.Header.TimestampAbs) * 90

	m.updateFragment(pts, m.spspps == nil, 2)

	if m.aaframe == nil {
		m.aframePTS = pts
	}

	adtsHeader, _ := m.adts.GetADTS(uint16(msg.Header.MsgLen))
	m.aaframe = append(m.aaframe, adtsHeader...)
	m.aaframe = append(m.aaframe, msg.Payload[2:]...)
}

func (m *Muxer) cacheAACSeqHeader(msg rtmp.AVMsg) {
	_ = m.adts.PutAACSequenceHeader(msg.Payload)
}

func (m *Muxer) cacheSPSPPS(msg rtmp.AVMsg) {
	m.spspps = msg.Payload
}

func (m *Muxer) appendSPSPPS(out []byte) []byte {
	if m.spspps == nil {
		nazalog.Warnf("append spspps by not exist. [%s]", m.UniqueKey)
		return out
	}

	index := 10
	nnals := m.spspps[index] & 0x1f
	index++
	for n := 0; ; n++ {
		for ; nnals != 0; nnals-- {
			length := int(bele.BEUint16(m.spspps[index:]))
			index += 2
			out = append(out, nalStartCode...)
			out = append(out, m.spspps[index:index+length]...)
			index += length
		}

		if n == 1 {
			break
		}
		nnals = m.spspps[index]
		index++
	}
	return out
}

func (m *Muxer) updateFragment(ts uint64, boundary bool, flushRate int) {
	force := false
	discont := true
	var f *fragmentInfo

	if m.opened {
		f = m.getFrag(m.nfrags)

		// 当前时间戳跳跃很大，或者是往回跳跃超过了阈值，强制开启新的fragment
		maxfraglen := uint64(m.config.FragmentDurationMS * 90 * 10)
		if (ts > m.fragTS && ts-m.fragTS > maxfraglen) || (m.fragTS > ts && m.fragTS-ts > negMaxfraglen) {
			nazalog.Warnf("force fragment split. [%s] fragTS=%d, ts=%d", m.UniqueKey, m.fragTS, ts)
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
		m.closeFragment()
		m.openFragment(ts, discont)
	}

	// 音频已经缓存了一定时长的数据了，需要落盘了
	if m.opened && m.aaframe != nil && ((m.aframePTS + maxAudioDelay*90/uint64(flushRate)) < ts) {
		m.flushAudio()
	}
}
/*
创建一个新TS分片文件
ts: 当前帧的时间pts/dts
discont:
*/
func (m *Muxer) openFragment(ts uint64, discont bool) {
	if m.opened {
		return
	}
	//文件序号，一直增加唯一数值
	id := m.getFragmentID()

	filename := getTSFilename(m.outPath, m.streamName, id)
	_ = m.fragmentOP.OpenFile(filename)
	m.opened = true

	frag := m.getFrag(m.nfrags)
	frag.discont = discont
	frag.id = id
	//新分片的开始时间
	m.fragTS = ts
	//把好个分片还未写入文件的音频帧写入新分片中
	m.flushAudio()
}

func (m *Muxer) closeFragment() {
	if !m.opened {
		return
	}
	//关闭当前分片文件
	m.fragmentOP.CloseFile()
	m.opened = false
	m.clearTS()
	//更新序号，为下个分片准备好
	m.nextFrag()
	//一个分片完成后，把这个分片文件名写入playlist文件中
	m.writePlaylist()
}
//ljy added.删除没有在playlist中的TS文件
func (m *Muxer) clearTS() {
	//frag>0表示已经有超过config.FragmentNum个文件产生，可以删除了
	if m.frag > 0 {
		name := getTSFilename(m.outPath, m.streamName, m.frag - 1)
		nazalog.Println("delete ts file: ",name)
		if os.Remove(name) != nil {
			nazalog.Warnf("delete ts file %s error",name)
		}
	}
}
func (m *Muxer) writePlaylist() {
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

	//写入当前config.FragmentNum(winfrags)内的所有分片时长和文件名信息
	for i := 0; i < m.nfrags; i++ {
		frag := m.getFrag(i)
		if frag.discont {
			buf.WriteString("#EXT-X-DISCONTINUITY\n")
		}
		buf.WriteString(fmt.Sprintf("#EXTINF:%.3f,\n%s\n", frag.duration, getTSFilenameWithoutPath(m.streamName, frag.id)))
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
		//当nfrags==m.config.FragmentNum时就不会执行这里了
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
		nazalog.Warnf("flushAudio by not opened. [%s]", m.UniqueKey)
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
