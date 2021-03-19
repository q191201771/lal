// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package remux

import (
	"github.com/cfeeling/lal/pkg/aac"
	"github.com/cfeeling/lal/pkg/avc"
	"github.com/cfeeling/lal/pkg/base"
	"github.com/cfeeling/lal/pkg/hevc"
	"github.com/cfeeling/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/bele"
)

// @return 返回的内存块为新申请的独立内存块
//
func AVConfig2RTMPMsg(asc, vps, sps, pps []byte) (metadata, ash, vsh *base.RTMPMsg, err error) {
	var bMetadata []byte
	var bVsh []byte
	var bAsh []byte

	hasAudio := asc != nil
	hasVideo := sps != nil && pps != nil
	isHEVC := vps != nil

	if !hasAudio && !hasVideo {
		err = ErrRemux
		return
	}

	audiocodecid := -1
	if hasAudio {
		audiocodecid = int(base.RTMPSoundFormatAAC)
	}
	videocodecid := -1
	width := -1
	height := -1
	if hasVideo {
		if isHEVC {
			videocodecid = int(base.RTMPCodecIDHEVC)
			var ctx hevc.Context
			if err = hevc.ParseSPS(sps, &ctx); err != nil {
				return
			}
			width = int(ctx.PicWidthInLumaSamples)
			height = int(ctx.PicHeightInLumaSamples)
			bVsh, err = hevc.BuildSeqHeaderFromVPSSPSPPS(vps, sps, pps)
			if err != nil {
				return
			}
		} else {
			videocodecid = int(base.RTMPCodecIDAVC)
			var ctx avc.Context
			err = avc.ParseSPS(sps, &ctx)
			if err != nil {
				return
			}
			if ctx.Width != 0 {
				width = int(ctx.Width)
			}
			if ctx.Height != 0 {
				height = int(ctx.Height)
			}
			bVsh, err = avc.BuildSeqHeaderFromSPSPPS(sps, pps)
			if err != nil {
				return
			}
		}
	}

	if hasAudio {
		bAsh, err = aac.BuildAACSeqHeader(asc)
		if err != nil {
			return
		}
	}

	var h base.RTMPHeader

	bMetadata, err = rtmp.BuildMetadata(width, height, audiocodecid, videocodecid)
	if err != nil {
		return
	}

	h.MsgLen = uint32(len(bMetadata))
	h.TimestampAbs = 0
	h.MsgTypeID = base.RTMPTypeIDMetadata
	h.MsgStreamID = rtmp.MSID1
	h.CSID = rtmp.CSIDAMF
	metadata = &base.RTMPMsg{
		Header:  h,
		Payload: bMetadata,
	}

	if hasVideo {
		h.MsgLen = uint32(len(bVsh))
		h.TimestampAbs = 0
		h.MsgTypeID = base.RTMPTypeIDVideo
		h.MsgStreamID = rtmp.MSID1
		h.CSID = rtmp.CSIDVideo
		vsh = &base.RTMPMsg{
			Header:  h,
			Payload: bVsh,
		}
	}

	if hasAudio {
		h.MsgLen = uint32(len(bAsh))
		h.TimestampAbs = 0
		h.MsgTypeID = base.RTMPTypeIDAudio
		h.MsgStreamID = rtmp.MSID1
		h.CSID = rtmp.CSIDAudio
		ash = &base.RTMPMsg{
			Header:  h,
			Payload: bAsh,
		}
	}

	return
}

// @return 返回的内存块为新申请的独立内存块
func AVPacket2RTMPMsg(pkt base.AVPacket) (msg base.RTMPMsg, err error) {
	switch pkt.PayloadType {
	case base.AVPacketPTAVC:
		fallthrough
	case base.AVPacketPTHEVC:
		msg.Header.TimestampAbs = pkt.Timestamp
		msg.Header.MsgStreamID = rtmp.MSID1

		msg.Header.MsgTypeID = base.RTMPTypeIDVideo
		msg.Header.CSID = rtmp.CSIDVideo
		msg.Header.MsgLen = uint32(len(pkt.Payload)) + 5

		msg.Payload = make([]byte, msg.Header.MsgLen)

		// TODO chef: 这段代码应该放在更合适的地方，或者在AVPacket中标识是否包含关键帧
		for i := 0; i != len(pkt.Payload); {
			naluSize := int(bele.BEUint32(pkt.Payload[i:]))

			t := avc.ParseNALUType(pkt.Payload[i+4])
			switch pkt.PayloadType {
			case base.AVPacketPTAVC:
				if t == avc.NALUTypeIDRSlice {
					msg.Payload[0] = base.RTMPAVCKeyFrame
				} else {
					msg.Payload[0] = base.RTMPAVCInterFrame
				}
				msg.Payload[1] = base.RTMPAVCPacketTypeNALU
			case base.AVPacketPTHEVC:
				if t == hevc.NALUTypeSliceIDR || t == hevc.NALUTypeSliceIDRNLP {
					msg.Payload[0] = base.RTMPHEVCKeyFrame
				} else {
					msg.Payload[0] = base.RTMPHEVCInterFrame
				}
				msg.Payload[1] = base.RTMPHEVCPacketTypeNALU
			}

			i += 4 + naluSize
		}

		msg.Payload[2] = 0x0 // cts
		msg.Payload[3] = 0x0
		msg.Payload[4] = 0x0
		copy(msg.Payload[5:], pkt.Payload)
		//nazalog.Debugf("%d %s", len(msg.Payload), hex.Dump(msg.Payload[:32]))
	case base.AVPacketPTAAC:
		msg.Header.TimestampAbs = pkt.Timestamp
		msg.Header.MsgStreamID = rtmp.MSID1

		msg.Header.MsgTypeID = base.RTMPTypeIDAudio
		msg.Header.CSID = rtmp.CSIDAudio
		msg.Header.MsgLen = uint32(len(pkt.Payload)) + 2

		msg.Payload = make([]byte, msg.Header.MsgLen)
		msg.Payload[0] = 0xAF
		msg.Payload[1] = base.RTMPAACPacketTypeRaw
		copy(msg.Payload[2:], pkt.Payload)
	default:
		err = ErrRemux
		return
	}

	return
}
