// Copyright 2020, Chef.  All rights reserved.
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
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazalog"
)

// @param asc 如果为nil，则没有音频
// @param vps 如果为nil，则是H264，如果不为nil，则是H265
// @return 返回的内存块为新申请的独立内存块
func AVConfig2FLVTag(asc, vps, sps, pps []byte) (metadata, ash, vsh *httpflv.Tag, err error) {
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
			err = hevc.ParseSPS(sps, &ctx)
			if err == nil {
				width = int(ctx.PicWidthInLumaSamples)
				height = int(ctx.PicHeightInLumaSamples)
			} else {
				nazalog.Warnf("parse hevc sps failed. err=%+v", err)
			}
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

	var h httpflv.TagHeader
	var tagRaw []byte

	bMetadata, err = rtmp.BuildMetadata(width, height, audiocodecid, videocodecid)
	if err != nil {
		return
	}

	h.Type = base.RTMPTypeIDMetadata
	h.DataSize = uint32(len(bMetadata))
	h.Timestamp = 0
	tagRaw = httpflv.PackHTTPFLVTag(h.Type, h.Timestamp, bMetadata)
	metadata = &httpflv.Tag{
		Header: h,
		Raw:    tagRaw,
	}

	if hasVideo {
		h.Type = base.RTMPTypeIDVideo
		h.DataSize = uint32(len(bVsh))
		h.Timestamp = 0
		tagRaw = httpflv.PackHTTPFLVTag(h.Type, h.Timestamp, bVsh)
		vsh = &httpflv.Tag{
			Header: h,
			Raw:    tagRaw,
		}
	}

	if hasAudio {
		h.Type = base.RTMPTypeIDAudio
		h.DataSize = uint32(len(bAsh))
		h.Timestamp = 0
		tagRaw = httpflv.PackHTTPFLVTag(h.Type, h.Timestamp, bAsh)
		ash = &httpflv.Tag{
			Header: h,
			Raw:    tagRaw,
		}
	}

	return
}

// @return 返回的内存块为新申请的独立内存块
func AVPacket2FLVTag(pkt base.AVPacket) (tag httpflv.Tag, err error) {
	switch pkt.PayloadType {
	case base.AVPacketPTAVC:
		fallthrough
	case base.AVPacketPTHEVC:
		tag.Header.Type = base.RTMPTypeIDVideo
		tag.Header.DataSize = uint32(len(pkt.Payload)) + 5
		tag.Header.Timestamp = pkt.Timestamp
		tag.Raw = make([]byte, httpflv.TagHeaderSize+int(tag.Header.DataSize)+httpflv.PrevTagSizeFieldSize)
		tag.Raw[0] = tag.Header.Type
		bele.BEPutUint24(tag.Raw[1:], tag.Header.DataSize)
		bele.BEPutUint24(tag.Raw[4:], tag.Header.Timestamp&0xFFFFFF)
		tag.Raw[7] = uint8(tag.Header.Timestamp >> 24)
		tag.Raw[8] = 0
		tag.Raw[9] = 0
		tag.Raw[10] = 0

		// TODO chef: 这段代码应该放在更合适的地方，或者在AVPacket中标识是否包含关键帧
		for i := 0; i != len(pkt.Payload); {
			naluSize := int(bele.BEUint32(pkt.Payload[i:]))

			switch pkt.PayloadType {
			case base.AVPacketPTAVC:
				t := avc.ParseNALUType(pkt.Payload[i+4])
				if t == avc.NALUTypeIDRSlice {
					tag.Raw[httpflv.TagHeaderSize] = base.RTMPAVCKeyFrame
				} else {
					tag.Raw[httpflv.TagHeaderSize] = base.RTMPAVCInterFrame
				}
				tag.Raw[httpflv.TagHeaderSize+1] = base.RTMPAVCPacketTypeNALU
			case base.AVPacketPTHEVC:
				t := hevc.ParseNALUType(pkt.Payload[i+4])
				if t == hevc.NALUTypeSliceIDR || t == hevc.NALUTypeSliceIDRNLP {
					tag.Raw[httpflv.TagHeaderSize] = base.RTMPHEVCKeyFrame
				} else {
					tag.Raw[httpflv.TagHeaderSize] = base.RTMPHEVCInterFrame
				}
				tag.Raw[httpflv.TagHeaderSize+1] = base.RTMPHEVCPacketTypeNALU
			}

			i += 4 + naluSize
		}

		tag.Raw[httpflv.TagHeaderSize+2] = 0x0 // cts
		tag.Raw[httpflv.TagHeaderSize+3] = 0x0
		tag.Raw[httpflv.TagHeaderSize+4] = 0x0
		copy(tag.Raw[httpflv.TagHeaderSize+5:], pkt.Payload)
		bele.BEPutUint32(tag.Raw[httpflv.TagHeaderSize+int(tag.Header.DataSize):], uint32(httpflv.TagHeaderSize)+tag.Header.DataSize)
		//nazalog.Debugf("%d %s", len(msg.Payload), hex.Dump(msg.Payload[:32]))
	case base.AVPacketPTAAC:
		tag.Header.Type = base.RTMPTypeIDAudio
		tag.Header.DataSize = uint32(len(pkt.Payload)) + 2
		tag.Header.Timestamp = pkt.Timestamp
		tag.Raw = make([]byte, httpflv.TagHeaderSize+int(tag.Header.DataSize)+httpflv.PrevTagSizeFieldSize)
		tag.Raw[0] = tag.Header.Type
		bele.BEPutUint24(tag.Raw[1:], tag.Header.DataSize)
		bele.BEPutUint24(tag.Raw[4:], tag.Header.Timestamp&0xFFFFFF)
		tag.Raw[7] = uint8(tag.Header.Timestamp >> 24)
		tag.Raw[8] = 0
		tag.Raw[9] = 0
		tag.Raw[10] = 0
		tag.Raw[httpflv.TagHeaderSize] = 0xAF
		tag.Raw[httpflv.TagHeaderSize+1] = base.RTMPAACPacketTypeRaw
		copy(tag.Raw[httpflv.TagHeaderSize+2:], pkt.Payload)
		bele.BEPutUint32(tag.Raw[httpflv.TagHeaderSize+int(tag.Header.DataSize):], uint32(httpflv.TagHeaderSize)+tag.Header.DataSize)
	default:
		err = ErrRemux
		return
	}

	return
}
