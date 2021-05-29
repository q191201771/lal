// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package remux

import (
	"github.com/q191201771/lal/pkg/aac"
	"github.com/q191201771/lal/pkg/avc"
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/hevc"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazalog"
)

// AVPacket转换为RTMP
// 目前AVPacket来自RTSP的sdp以及rtp包。理论上也支持webrtc，后续接入webrtc时再验证
type AVPacket2RTMPRemuxer struct {
	onRTMPAVMsg rtmp.OnReadRTMPAVMsg

	hasEmittedMetadata bool
	audioType          base.AVPacketPT
	videoType          base.AVPacketPT

	vps []byte // 从AVPacket数据中获取
	sps []byte
	pps []byte
}

func NewAVPacket2RTMPRemuxer(onRTMPAVMsg rtmp.OnReadRTMPAVMsg) *AVPacket2RTMPRemuxer {
	return &AVPacket2RTMPRemuxer{
		onRTMPAVMsg: onRTMPAVMsg,
		audioType:   base.AVPacketPTUnknown,
		videoType:   base.AVPacketPTUnknown,
	}
}

// 实现RTSP回调数据的三个接口，使得接入时方便些
func (r *AVPacket2RTMPRemuxer) OnRTPPacket(pkt rtprtcp.RTPPacket) {
	// noop
}
func (r *AVPacket2RTMPRemuxer) OnSDP(sdpCtx sdp.LogicContext) {
	r.InitWithAVConfig(sdpCtx.ASC, sdpCtx.VPS, sdpCtx.SPS, sdpCtx.PPS)
}
func (r *AVPacket2RTMPRemuxer) OnAVPacket(pkt base.AVPacket) {
	r.FeedAVPacket(pkt)
}

// rtsp场景下，有时sps、pps等信息只包含在sdp中，有时包含在rtp包中，
// 这里提供输入sdp的sps、pps等信息的机会，如果没有，可以不调用
//
// 内部不持有输入参数的内存块
//
func (r *AVPacket2RTMPRemuxer) InitWithAVConfig(asc, vps, sps, pps []byte) {
	var err error
	var bVsh []byte
	var bAsh []byte

	if asc != nil {
		r.audioType = base.AVPacketPTAAC
	}
	if sps != nil && pps != nil {
		if vps != nil {
			r.videoType = base.AVPacketPTHEVC
		} else {
			r.videoType = base.AVPacketPTAVC
		}
	}

	if r.audioType == base.AVPacketPTUnknown && r.videoType == base.AVPacketPTUnknown {
		nazalog.Warn("has no audio or video")
		return
	}

	if r.audioType != base.AVPacketPTUnknown {
		bAsh, err = aac.BuildAACSeqHeader(asc)
		if err != nil {
			nazalog.Errorf("build aac seq header failed. err=%+v", err)
			return
		}
	}

	if r.videoType != base.AVPacketPTUnknown {
		if r.videoType == base.AVPacketPTHEVC {
			bVsh, err = hevc.BuildSeqHeaderFromVPSSPSPPS(vps, sps, pps)
			if err != nil {
				nazalog.Errorf("build hevc seq header failed. err=%+v", err)
				return
			}
		} else {
			bVsh, err = avc.BuildSeqHeaderFromSPSPPS(sps, pps)
			if err != nil {
				nazalog.Errorf("build avc seq header failed. err=%+v", err)
				return
			}
		}
	}

	if r.audioType != base.AVPacketPTUnknown {
		r.emitRTMPAVMsg(true, bAsh, 0)
	}

	if r.videoType != base.AVPacketPTUnknown {
		r.emitRTMPAVMsg(false, bVsh, 0)
	}
}

// @param pkt: 内部不持有该内存块
//
func (r *AVPacket2RTMPRemuxer) FeedAVPacket(pkt base.AVPacket) {
	switch pkt.PayloadType {
	case base.AVPacketPTAVC:
		fallthrough
	case base.AVPacketPTHEVC:
		nals, err := avc.SplitNALUAVCC(pkt.Payload)
		if err != nil {
			nazalog.Errorf("iterate nalu failed. err=%+v", err)
			return
		}

		pos := 5
		maxLength := len(pkt.Payload) + pos
		payload := make([]byte, maxLength)

		for _, nal := range nals {
			if pkt.PayloadType == base.AVPacketPTAVC {
				t := avc.ParseNALUType(nal[0])
				if t == avc.NALUTypeSPS || t == avc.NALUTypePPS {
					// 如果有sps，pps，先把它们抽离出来进行缓存
					if t == avc.NALUTypeSPS {
						r.setSPS(nal)
					} else {
						r.setPPS(nal)
					}

					if r.sps != nil && r.pps != nil {
						// TODO(chef): 是否应该判断sps、pps是连续的，比如rtp seq的关系，或者timestamp是相等的
						// 凑齐了，发送video seq header

						bVsh, err := avc.BuildSeqHeaderFromSPSPPS(r.sps, r.pps)
						if err != nil {
							nazalog.Errorf("build avc seq header failed. err=%+v", err)
							continue
						}
						r.emitRTMPAVMsg(false, bVsh, pkt.Timestamp)
						r.clearVideoSeqHeader()
					}
				} else {
					// 重组实际数据

					if t == avc.NALUTypeIDRSlice {
						payload[0] = base.RTMPAVCKeyFrame
					} else {
						payload[0] = base.RTMPAVCInterFrame
					}
					payload[1] = base.RTMPAVCPacketTypeNALU
					bele.BEPutUint32(payload[pos:], uint32(len(nal)))
					pos += 4
					copy(payload[pos:], nal)
					pos += len(nal)
				}
			} else if pkt.PayloadType == base.AVPacketPTHEVC {
				t := hevc.ParseNALUType(nal[0])
				if t == hevc.NALUTypeVPS || t == hevc.NALUTypeSPS || t == hevc.NALUTypePPS {
					if t == hevc.NALUTypeVPS {
						r.setVPS(nal)
					} else if t == hevc.NALUTypeSPS {
						r.setSPS(nal)
					} else {
						r.setPPS(nal)
					}
					if r.vps != nil && r.sps != nil && r.pps != nil {
						bVsh, err := hevc.BuildSeqHeaderFromVPSSPSPPS(r.vps, r.sps, r.pps)
						if err != nil {
							nazalog.Errorf("build hevc seq header failed. err=%+v", err)
							continue
						}
						r.emitRTMPAVMsg(false, bVsh, pkt.Timestamp)
						r.clearVideoSeqHeader()
					}
				} else {
					if t == hevc.NALUTypeSliceIDR || t == hevc.NALUTypeSliceIDRNLP {
						payload[0] = base.RTMPHEVCKeyFrame
					} else {
						payload[0] = base.RTMPHEVCInterFrame
					}
					payload[1] = base.RTMPHEVCPacketTypeNALU
					bele.BEPutUint32(payload[pos:], uint32(len(nal)))
					pos += 4
					copy(payload[pos:], nal)
					pos += len(nal)
				}
			}
		}

		// 有实际数据
		if pos > 5 {
			r.emitRTMPAVMsg(false, payload[:pos], pkt.Timestamp)
		}

	case base.AVPacketPTAAC:
		length := len(pkt.Payload) + 2
		payload := make([]byte, length)
		// TODO(chef) 处理此处的魔数0xAF
		payload[0] = 0xAF
		payload[1] = base.RTMPAACPacketTypeRaw
		copy(payload[2:], pkt.Payload)
		r.emitRTMPAVMsg(true, payload, pkt.Timestamp)
	default:
		nazalog.Warnf("unsupported packet. type=%d", pkt.PayloadType)
	}
}

func (r *AVPacket2RTMPRemuxer) emitRTMPAVMsg(isAudio bool, payload []byte, timestamp uint32) {
	if !r.hasEmittedMetadata {
		// TODO(chef): 此处简化了从sps中获取宽高写入metadata的逻辑
		audiocodecid := -1
		videocodecid := -1
		if r.audioType == base.AVPacketPTAAC {
			audiocodecid = int(base.RTMPSoundFormatAAC)
		}
		switch r.videoType {
		case base.AVPacketPTAVC:
			videocodecid = int(base.RTMPCodecIDAVC)
		case base.AVPacketPTHEVC:
			videocodecid = int(base.RTMPCodecIDHEVC)
		}
		bMetadata, err := rtmp.BuildMetadata(-1, -1, audiocodecid, videocodecid)
		if err != nil {
			nazalog.Errorf("build metadata failed. err=%+v", err)
			return
		}
		r.onRTMPAVMsg(base.RTMPMsg{
			Header: base.RTMPHeader{
				CSID:         rtmp.CSIDAMF,
				MsgLen:       uint32(len(bMetadata)),
				MsgTypeID:    base.RTMPTypeIDMetadata,
				MsgStreamID:  rtmp.MSID1,
				TimestampAbs: 0,
			},
			Payload: bMetadata,
		})
		r.hasEmittedMetadata = true
	}

	var msg base.RTMPMsg
	msg.Header.MsgStreamID = rtmp.MSID1

	if isAudio {
		msg.Header.CSID = rtmp.CSIDAudio
		msg.Header.MsgTypeID = base.RTMPTypeIDAudio
	} else {
		msg.Header.CSID = rtmp.CSIDVideo
		msg.Header.MsgTypeID = base.RTMPTypeIDVideo
	}

	msg.Header.MsgLen = uint32(len(payload))
	msg.Header.TimestampAbs = timestamp
	msg.Payload = payload

	r.onRTMPAVMsg(msg)
}

func (r *AVPacket2RTMPRemuxer) setVPS(b []byte) {
	r.vps = r.vps[0:0]
	r.vps = append(r.vps, b...)
}

func (r *AVPacket2RTMPRemuxer) setSPS(b []byte) {
	r.sps = r.sps[0:0]
	r.sps = append(r.sps, b...)
}

func (r *AVPacket2RTMPRemuxer) setPPS(b []byte) {
	r.pps = r.pps[0:0]
	r.pps = append(r.pps, b...)
}

func (r *AVPacket2RTMPRemuxer) clearVideoSeqHeader() {
	r.vps = r.vps[0:0]
	r.sps = r.sps[0:0]
	r.pps = r.pps[0:0]
}
