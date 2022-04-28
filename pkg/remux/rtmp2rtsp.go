// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package remux

import (
	"encoding/hex"
	"math/rand"
	"time"

	"github.com/q191201771/lal/pkg/aac"
	"github.com/q191201771/lal/pkg/avc"
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/hevc"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"
)

// TODO(chef): refactor 将analyze部分独立出来作为一个filter
// TODO(chef): fix 如果前面来的音频和视频数据没有seq header，都是gop中间的数据，那么analyze分析的结果可能是音频和视频都没有

var (
	// config
	// TODO(chef): 提供option，另外还有ssrc和pt都支持自定义
	maxAnalyzeAvMsgSize = 16
)

// Rtmp2RtspRemuxer 提供rtmp数据向sdp+rtp数据的转换
type Rtmp2RtspRemuxer struct {
	onSdp       OnSdp
	onRtpPacket OnRtpPacket

	analyzeDone        bool
	msgCache           []base.RtmpMsg
	vps, sps, pps, asc []byte
	audioPt            base.AvPacketPt
	videoPt            base.AvPacketPt

	audioSsrc   uint32
	videoSsrc   uint32
	audioPacker *rtprtcp.RtpPacker
	videoPacker *rtprtcp.RtpPacker
}

type OnSdp func(sdpCtx sdp.LogicContext)
type OnRtpPacket func(pkt rtprtcp.RtpPacket)

// NewRtmp2RtspRemuxer @param onSdp:       每次回调为独立的内存块，回调结束后，内部不再使用该内存块
// @param onRtpPacket: 每次回调为独立的内存块，回调结束后，内部不再使用该内存块
//
func NewRtmp2RtspRemuxer(onSdp OnSdp, onRtpPacket OnRtpPacket) *Rtmp2RtspRemuxer {
	return &Rtmp2RtspRemuxer{
		onSdp:       onSdp,
		onRtpPacket: onRtpPacket,
		audioPt:     base.AvPacketPtUnknown,
		videoPt:     base.AvPacketPtUnknown,
	}
}

// FeedRtmpMsg @param msg: 函数调用结束后，内部不持有`msg`内存块
//
func (r *Rtmp2RtspRemuxer) FeedRtmpMsg(msg base.RtmpMsg) {
	var err error

	switch msg.Header.MsgTypeId {
	case base.RtmpTypeIdMetadata:
		return
	case base.RtmpTypeIdAudio:
		if len(msg.Payload) <= 2 {
			Log.Warnf("rtmp msg too short, ignore. header=%+v, payload=%s", msg.Header, hex.Dump(msg.Payload))
			return
		}
	case base.RtmpTypeIdVideo:
		if len(msg.Payload) <= 5 {
			Log.Warnf("rtmp msg too short, ignore. header=%+v, payload=%s", msg.Header, hex.Dump(msg.Payload))
			return
		}
	}

	if msg.Header.MsgTypeId == base.RtmpTypeIdMetadata {
		// noop
		return
	}

	// 我们需要先接收一部分rtmp数据，得到音频头、视频头
	// 并且考虑，流中只有音频或只有视频的情况
	// 我们把前面这个阶段叫做Analyze分析阶段

	if !r.analyzeDone {
		if msg.IsAvcKeySeqHeader() || msg.IsHevcKeySeqHeader() {
			if msg.IsAvcKeySeqHeader() {
				r.sps, r.pps, err = avc.ParseSpsPpsFromSeqHeader(msg.Payload)
				Log.Assert(nil, err)
			} else if msg.IsHevcKeySeqHeader() {
				r.vps, r.sps, r.pps, err = hevc.ParseVpsSpsPpsFromSeqHeader(msg.Payload)
				Log.Assert(nil, err)
			}
			r.doAnalyze()
			return
		}

		if msg.IsAacSeqHeader() {
			r.asc = msg.Clone().Payload[2:]
			r.doAnalyze()
			return
		}

		r.msgCache = append(r.msgCache, msg.Clone())
		r.doAnalyze()
		return
	}

	// 正常阶段

	// 音视频头已通过sdp回调，rtp数据中不再包含音视频头
	if msg.IsAvcKeySeqHeader() || msg.IsHevcKeySeqHeader() || msg.IsAacSeqHeader() {
		return
	}

	r.remux(msg)
}

func (r *Rtmp2RtspRemuxer) doAnalyze() {
	Log.Assert(false, r.analyzeDone)

	if r.isAnalyzeEnough() {
		if r.sps != nil && r.pps != nil {
			if r.vps != nil {
				r.videoPt = base.AvPacketPtHevc
			} else {
				r.videoPt = base.AvPacketPtAvc
			}
		}
		if r.asc != nil {
			r.audioPt = base.AvPacketPtAac
		}

		// 回调sdp
		ctx, err := sdp.Pack(r.vps, r.sps, r.pps, r.asc)
		Log.Assert(nil, err)
		r.onSdp(ctx)

		// 分析阶段缓存的数据
		for i := range r.msgCache {
			r.remux(r.msgCache[i])
		}
		r.msgCache = nil

		r.analyzeDone = true
	}
}

// 是否应该退出Analyze阶段
func (r *Rtmp2RtspRemuxer) isAnalyzeEnough() bool {
	// 音视频头都收集好了
	if r.sps != nil && r.pps != nil && r.asc != nil {
		return true
	}

	// 达到分析包数阈值了
	if len(r.msgCache) >= maxAnalyzeAvMsgSize {
		return true
	}

	return false
}

func (r *Rtmp2RtspRemuxer) remux(msg base.RtmpMsg) {
	var packer *rtprtcp.RtpPacker
	var rtppkts []rtprtcp.RtpPacket
	switch msg.Header.MsgTypeId {
	case base.RtmpTypeIdAudio:
		packer = r.getAudioPacker()
		if packer != nil {
			rtppkts = packer.Pack(base.AvPacket{
				Timestamp:   int64(msg.Header.TimestampAbs),
				PayloadType: r.audioPt,
				Payload:     msg.Payload[2:],
			})
		}
	case base.RtmpTypeIdVideo:
		packer = r.getVideoPacker()
		if packer != nil {
			rtppkts = r.getVideoPacker().Pack(base.AvPacket{
				Timestamp:   int64(msg.Header.TimestampAbs),
				PayloadType: r.videoPt,
				Payload:     msg.Payload[5:],
			})
		}
	}

	for i := range rtppkts {
		r.onRtpPacket(rtppkts[i])
	}
}

func (r *Rtmp2RtspRemuxer) getAudioPacker() *rtprtcp.RtpPacker {
	if r.asc == nil {
		return nil
	}

	if r.audioPacker == nil {
		// TODO(chef): ssrc随机产生，并且整个lal没有在setup信令中传递ssrc
		r.audioSsrc = rand.Uint32()

		ascCtx, err := aac.NewAscContext(r.asc)
		if err != nil {
			Log.Errorf("parse asc failed. err=%+v", err)
			return nil
		}
		clockRate, err := ascCtx.GetSamplingFrequency()
		if err != nil {
			Log.Errorf("get sampling frequency failed. err=%+v, asc=%s", err, hex.Dump(r.asc))
		}

		pp := rtprtcp.NewRtpPackerPayloadAac()
		r.audioPacker = rtprtcp.NewRtpPacker(pp, clockRate, r.audioSsrc)
	}
	return r.audioPacker
}

func (r *Rtmp2RtspRemuxer) getVideoPacker() *rtprtcp.RtpPacker {
	if r.sps == nil {
		return nil
	}
	if r.videoPacker == nil {
		r.videoSsrc = rand.Uint32()
		pp := rtprtcp.NewRtpPackerPayloadAvcHevc(r.videoPt, func(option *rtprtcp.RtpPackerPayloadAvcHevcOption) {
			option.Typ = rtprtcp.RtpPackerPayloadAvcHevcTypeAvcc
		})
		r.videoPacker = rtprtcp.NewRtpPacker(pp, 90000, r.videoSsrc)
	}
	return r.videoPacker
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
