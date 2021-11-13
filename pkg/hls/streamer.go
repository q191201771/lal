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

	"github.com/q191201771/lal/pkg/aac"
	"github.com/q191201771/lal/pkg/avc"
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/hevc"
	"github.com/q191201771/lal/pkg/mpegts"
	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazastring"
)

type StreamerObserver interface {
	// @param b const只读内存块，上层可以持有，但是不允许修改
	OnPatPmt(b []byte)

	// @param streamer: 供上层获取streamer内部的一些状态，比如spspps是否已缓存，音频缓存队列是否有数据等
	//
	// @param frame:    各字段含义见mpegts.Frame结构体定义
	//                  frame.CC  注意，回调结束后，Streamer会保存frame.CC，上层在TS打包完成后，可通过frame.CC将cc值传递给Streamer
	//                  frame.Raw 回调结束后，这块内存可能会被内部重复使用
	//
	OnFrame(streamer *Streamer, frame *mpegts.Frame)
}

// 输入rtmp流，回调转封装成Annexb格式的流
type Streamer struct {
	UniqueKey string

	observer                StreamerObserver
	calcFragmentHeaderQueue *Queue
	videoOut                []byte // Annexb TODO chef: 优化这块buff
	spspps                  []byte // Annexb 也可能是vps+sps+pps
	ascCtx                  *aac.AscContext
	audioCacheFrames        []byte // 缓存音频帧数据，注意，可能包含多个音频帧 TODO chef: 优化这块buff
	audioCacheFirstFramePts uint64 // audioCacheFrames中第一个音频帧的时间戳 TODO chef: rename to DTS
	audioCc                 uint8
	videoCc                 uint8
}

func NewStreamer(observer StreamerObserver) *Streamer {
	uk := base.GenUkStreamer()
	videoOut := make([]byte, 1024*1024)
	videoOut = videoOut[0:0]
	streamer := &Streamer{
		UniqueKey: uk,
		observer:  observer,
		videoOut:  videoOut,
	}
	streamer.calcFragmentHeaderQueue = NewQueue(calcFragmentHeaderQueueSize, streamer)
	return streamer
}

// @param msg msg.Payload 调用结束后，函数内部不会持有这块内存
//
// TODO chef: 可以考虑数据有问题时，返回给上层，直接主动关闭输入流的连接
func (s *Streamer) FeedRtmpMessage(msg base.RtmpMsg) {
	s.calcFragmentHeaderQueue.Push(msg)
}

func (s *Streamer) OnPatPmt(b []byte) {
	s.observer.OnPatPmt(b)
}

func (s *Streamer) OnPop(msg base.RtmpMsg) {
	switch msg.Header.MsgTypeId {
	case base.RtmpTypeIdAudio:
		s.feedAudio(msg)
	case base.RtmpTypeIdVideo:
		s.feedVideo(msg)
	}
}

func (s *Streamer) AudioSeqHeaderCached() bool {
	return s.ascCtx != nil
}

func (s *Streamer) VideoSeqHeaderCached() bool {
	return s.spspps != nil
}

func (s *Streamer) AudioCacheEmpty() bool {
	return s.audioCacheFrames == nil
}

func (s *Streamer) feedVideo(msg base.RtmpMsg) {
	// 注意，有一种情况是msg.Payload为 27 02 00 00 00
	// 此时打印错误并返回也不影响
	//
	if len(msg.Payload) <= 5 {
		nazalog.Errorf("[%s] invalid video message length. header=%+v, payload=%s", s.UniqueKey, msg.Header, hex.Dump(msg.Payload))
		return
	}
	//nazalog.Debugf("[%s] feed video. header=%+v, payload=%s", s.UniqueKey, msg.Header, hex.Dump(nazastring.SubSliceSafety(msg.Payload, 16)))

	codecId := msg.Payload[0] & 0xF
	if codecId != base.RtmpCodecIdAvc && codecId != base.RtmpCodecIdHevc {
		return
	}

	// 将数据转换成Annexb

	// 如果是sps pps，缓存住，然后直接返回
	var err error
	if msg.IsAvcKeySeqHeader() {
		if s.spspps, err = avc.SpsPpsSeqHeader2Annexb(msg.Payload); err != nil {
			nazalog.Errorf("[%s] cache spspps failed. err=%+v", s.UniqueKey, err)
		}
		return
	} else if msg.IsHevcKeySeqHeader() {
		if s.spspps, err = hevc.VpsSpsPpsSeqHeader2Annexb(msg.Payload); err != nil {
			nazalog.Errorf("[%s] cache vpsspspps failed. err=%+v", s.UniqueKey, err)
		}
		return
	}

	cts := bele.BeUint24(msg.Payload[2:])

	audSent := false
	spsppsSent := false
	// 优化这块buffer
	out := s.videoOut[0:0]

	// msg中可能有多个NALU，逐个获取
	nals, err := avc.SplitNaluAvcc(msg.Payload[5:])
	if err != nil {
		nazalog.Errorf("[%s] iterate nalu failed. err=%+v, header=%+v, payload=%s", err, s.UniqueKey, msg.Header, hex.Dump(nazastring.SubSliceSafety(msg.Payload, 32)))
		return
	}
	for _, nal := range nals {
		var nalType uint8
		switch codecId {
		case base.RtmpCodecIdAvc:
			nalType = avc.ParseNaluType(nal[0])
		case base.RtmpCodecIdHevc:
			nalType = hevc.ParseNaluType(nal[0])
		}

		//nazalog.Debugf("[%s] naltype=%d, len=%d(%d), cts=%d, key=%t.", s.UniqueKey, nalType, nalBytes, len(msg.Payload), cts, msg.IsVideoKeyNalu())

		// 过滤掉原流中的sps pps aud
		// sps pps前面已经缓存过了，后面有自己的写入逻辑
		// aud有自己的写入逻辑
		if (codecId == base.RtmpCodecIdAvc && (nalType == avc.NaluTypeSps || nalType == avc.NaluTypePps || nalType == avc.NaluTypeAud)) ||
			(codecId == base.RtmpCodecIdHevc && (nalType == hevc.NaluTypeVps || nalType == hevc.NaluTypeSps || nalType == hevc.NaluTypePps || nalType == hevc.NaluTypeAud)) {
			continue
		}

		// tag中的首个nalu前面写入aud
		if !audSent {
			// 注意，因为前面已经过滤了sps pps aud的信息，所以这里可以认为都是需要用aud分隔的，不需要单独判断了
			//if codecId == base.RtmpCodecIdAvc && (nalType == avc.NaluTypeSei || nalType == avc.NaluTypeIdrSlice || nalType == avc.NaluTypeSlice) {
			switch codecId {
			case base.RtmpCodecIdAvc:
				out = append(out, avc.AudNalu...)
			case base.RtmpCodecIdHevc:
				out = append(out, hevc.AudNalu...)
			}
			audSent = true
		}

		// 关键帧前追加sps pps
		if codecId == base.RtmpCodecIdAvc {
			// h264的逻辑，一个tag中，多个连续的关键帧只追加一个，不连续则每个关键帧前都追加。为什么要这样处理
			switch nalType {
			case avc.NaluTypeIdrSlice:
				if !spsppsSent {
					if out, err = s.appendSpsPps(out); err != nil {
						nazalog.Warnf("[%s] append spspps by not exist.", s.UniqueKey)
						return
					}
				}
				spsppsSent = true
			case avc.NaluTypeSlice:
				// 这里只有P帧，没有SEI。为什么要这样处理
				spsppsSent = false
			}
		} else {
			switch nalType {
			case hevc.NaluTypeSliceIdr, hevc.NaluTypeSliceIdrNlp, hevc.NaluTypeSliceCranut:
				if !spsppsSent {
					if out, err = s.appendSpsPps(out); err != nil {
						nazalog.Warnf("[%s] append spspps by not exist.", s.UniqueKey)
						return
					}
				}
				spsppsSent = true
			default:
				// 这里简化了，只要不是关键帧，就刷新标志
				spsppsSent = false
			}
		}

		// 如果写入了aud或spspps，则用start code3，否则start code4。为什么要这样处理
		// 这里不知为什么要区分写入两种类型的start code
		if len(out) == 0 {
			out = append(out, avc.NaluStartCode4...)
		} else {
			out = append(out, avc.NaluStartCode3...)
		}

		out = append(out, nal...)
	}

	dts := uint64(msg.Header.TimestampAbs) * 90

	if s.audioCacheFrames != nil && s.audioCacheFirstFramePts+maxAudioCacheDelayByVideo < dts {
		s.FlushAudio()
	}

	var frame mpegts.Frame
	frame.Cc = s.videoCc
	frame.Dts = dts
	frame.Pts = frame.Dts + uint64(cts)*90
	frame.Key = msg.IsVideoKeyNalu()
	frame.Raw = out
	frame.Pid = mpegts.PidVideo
	frame.Sid = mpegts.StreamIdVideo

	s.observer.OnFrame(s, &frame)
	s.videoCc = frame.Cc
}

func (s *Streamer) feedAudio(msg base.RtmpMsg) {
	if len(msg.Payload) < 3 {
		nazalog.Errorf("[%s] invalid audio message length. len=%d", s.UniqueKey, len(msg.Payload))
		return
	}
	if msg.Payload[0]>>4 != base.RtmpSoundFormatAac {
		return
	}

	//nazalog.Debugf("[%s] hls: feedAudio. dts=%d len=%d", s.UniqueKey, msg.Header.TimestampAbs, len(msg.Payload))

	if msg.Payload[1] == base.RtmpAacPacketTypeSeqHeader {
		if err := s.cacheAacSeqHeader(msg); err != nil {
			nazalog.Errorf("[%s] cache aac seq header failed. err=%+v", s.UniqueKey, err)
		}
		return
	}

	if !s.AudioSeqHeaderCached() {
		nazalog.Warnf("[%s] feed audio message but aac seq header not exist.", s.UniqueKey)
		return
	}

	pts := uint64(msg.Header.TimestampAbs) * 90

	if s.audioCacheFrames != nil && s.audioCacheFirstFramePts+maxAudioCacheDelayByAudio < pts {
		s.FlushAudio()
	}

	if s.audioCacheFrames == nil {
		s.audioCacheFirstFramePts = pts
	}

	adtsHeader := s.ascCtx.PackAdtsHeader(int(msg.Header.MsgLen - 2))
	s.audioCacheFrames = append(s.audioCacheFrames, adtsHeader...)
	s.audioCacheFrames = append(s.audioCacheFrames, msg.Payload[2:]...)
}

// 吐出音频数据的三种情况：
// 1. 收到音频或视频时，音频缓存队列已达到一定长度
// 2. 打开一个新的TS文件切片时
// 3. 输入流关闭时
func (s *Streamer) FlushAudio() {
	if s.audioCacheFrames == nil {
		return
	}

	var frame mpegts.Frame
	frame.Cc = s.audioCc
	frame.Dts = s.audioCacheFirstFramePts
	frame.Pts = s.audioCacheFirstFramePts
	frame.Key = false
	frame.Raw = s.audioCacheFrames
	frame.Pid = mpegts.PidAudio
	frame.Sid = mpegts.StreamIdAudio
	s.observer.OnFrame(s, &frame)
	s.audioCc = frame.Cc

	s.audioCacheFrames = nil
}

func (s *Streamer) cacheAacSeqHeader(msg base.RtmpMsg) error {
	var err error
	s.ascCtx, err = aac.NewAscContext(msg.Payload[2:])
	return err
}

func (s *Streamer) appendSpsPps(out []byte) ([]byte, error) {
	if s.spspps == nil {
		return out, ErrHls
	}

	out = append(out, s.spspps...)
	return out, nil
}
