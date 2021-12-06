// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package aac

import (
	"errors"

	"github.com/q191201771/naza/pkg/nazabits"
	"github.com/q191201771/naza/pkg/nazalog"
)

// AudioSpecificConfig(asc)
// keywords: Seq Header,
// e.g.  rtmp, flv
//
// ADTS(Audio Data Transport Stream)
// e.g. es, ts
//
// StreamMuxConfig
//

var ErrAac = errors.New("lal.aac: fxxk")

const (
	AdtsHeaderLength = 7

	AscSamplingFrequencyIndex48000 = 3
	AscSamplingFrequencyIndex44100 = 4
)

const (
	minAscLength = 2
)

// <ISO_IEC_14496-3.pdf>
// <1.6.2.1 AudioSpecificConfig>, <page 33/110>
// <1.5.1.1 Audio Object type definition>, <page 23/110>
// <1.6.3.3 samplingFrequencyIndex>, <page 35/110>
// <1.6.3.4 channelConfiguration>
// --------------------------------------------------------
// audio object type      [5b] 1=AAC MAIN  2=AAC LC
// samplingFrequencyIndex [4b] 3=48000  4=44100  6=24000  5=32000  11=11025
// channelConfiguration   [4b] 1=center front speaker  2=left, right front speakers
type AscContext struct {
	AudioObjectType        uint8 // [5b]
	SamplingFrequencyIndex uint8 // [4b]
	ChannelConfiguration   uint8 // [4b]
}

func NewAscContext(asc []byte) (*AscContext, error) {
	var ascCtx AscContext
	if err := ascCtx.Unpack(asc); err != nil {
		return nil, err
	}
	return &ascCtx, nil
}

// @param asc: 2字节的AAC Audio Specifc Config
//             注意，如果是rtmp/flv的message/tag，应去除Seq Header头部的2个字节
//             函数调用结束后，内部不持有该内存块
//
func (ascCtx *AscContext) Unpack(asc []byte) error {
	if len(asc) < minAscLength {
		nazalog.Warnf("aac seq header length invalid. len=%d", len(asc))
		return ErrAac
	}

	br := nazabits.NewBitReader(asc)
	ascCtx.AudioObjectType, _ = br.ReadBits8(5)
	ascCtx.SamplingFrequencyIndex, _ = br.ReadBits8(4)
	ascCtx.ChannelConfiguration, _ = br.ReadBits8(4)
	return nil
}

// @return asc: 内存块为独立新申请；函数调用结束后，内部不持有该内存块
//
func (ascCtx *AscContext) Pack() (asc []byte) {
	asc = make([]byte, minAscLength)
	bw := nazabits.NewBitWriter(asc)
	bw.WriteBits8(5, ascCtx.AudioObjectType)
	bw.WriteBits8(4, ascCtx.SamplingFrequencyIndex)
	bw.WriteBits8(4, ascCtx.ChannelConfiguration)
	return
}

// 获取ADTS头，由于ADTS头中的字段依赖包的长度，而每个包的长度可能不同，所以每个包的ADTS头都需要独立生成
//
// @param frameLength: raw aac frame的大小
//                     注意，如果是rtmp/flv的message/tag，应去除Seq Header头部的2个字节
//
// @return h: 内存块为独立新申请；函数调用结束后，内部不持有该内存块
//
func (ascCtx *AscContext) PackAdtsHeader(frameLength int) (out []byte) {
	out = make([]byte, AdtsHeaderLength)
	_ = ascCtx.PackToAdtsHeader(out, frameLength)
	return
}

// @param out: 函数调用结束后，内部不持有该内存块
//
func (ascCtx *AscContext) PackToAdtsHeader(out []byte, frameLength int) error {
	if len(out) < AdtsHeaderLength {
		return ErrAac
	}

	// <ISO_IEC_14496-3.pdf>
	// <1.A.2.2.1 Fixed Header of ADTS>, <page 75/110>
	// <1.A.2.2.2 Variable Header of ADTS>, <page 76/110>
	// <1.A.3.2.1 Definitions: Bitstream elements for ADTS>
	// ----------------------------------------------------
	// Syncword                 [12b] '1111 1111 1111'
	// ID                       [1b]  1=MPEG-2 AAC 0=MPEG-4
	// Layer                    [2b]
	// protection_absent        [1b]  1=no crc check
	// Profile_ObjectType       [2b]
	// sampling_frequency_index [4b]
	// private_bit              [1b]
	// channel_configuration    [3b]
	// origin/copy              [1b]
	// home                     [1b]
	// Emphasis???
	// ------------------------------------
	// copyright_identification_bit   [1b]
	// copyright_identification_start [1b]
	// aac_frame_length               [13b]
	// adts_buffer_fullness           [11b]
	// no_raw_data_blocks_in_frame    [2b]

	bw := nazabits.NewBitWriter(out)
	// Syncword 0(8) 1(4)
	bw.WriteBits16(12, 0xFFF)
	// ID, Layer, protection_absent 1(4)
	bw.WriteBits8(4, 0x1)
	// 2(2)
	bw.WriteBits8(2, ascCtx.AudioObjectType-1)
	// 2(4)
	bw.WriteBits8(4, ascCtx.SamplingFrequencyIndex)
	// private_bit 2(1)
	bw.WriteBits8(1, 0)
	// 2(1) 3(2)
	bw.WriteBits8(3, ascCtx.ChannelConfiguration)
	// origin/copy, home, copyright_identification_bit, copyright_identification_start 3(4)
	bw.WriteBits8(4, 0)
	// 3(2) 4(8) 5(3)
	bw.WriteBits16(13, uint16(frameLength+AdtsHeaderLength))
	// adts_buffer_fullness 5(5) 6(6)
	bw.WriteBits16(11, 0x7FF)
	// no_raw_data_blocks_in_frame 6(2)
	bw.WriteBits8(2, 0)
	return nil
}

func (ascCtx *AscContext) GetSamplingFrequency() (int, error) {
	switch ascCtx.SamplingFrequencyIndex {
	case AscSamplingFrequencyIndex48000:
		return 48000, nil
	case AscSamplingFrequencyIndex44100:
		return 44100, nil
	}
	nazalog.Errorf("GetSamplingFrequency failed. ascCtx=%+v", ascCtx)
	return -1, ErrAac
}

type AdtsHeaderContext struct {
	AscCtx AscContext

	AdtsLength uint16 // 字段中的值，包含了adts header + adts frame
}

func NewAdtsHeaderContext(adtsHeader []byte) (*AdtsHeaderContext, error) {
	var ctx AdtsHeaderContext
	if err := ctx.Unpack(adtsHeader); err != nil {
		return nil, err
	}
	return &ctx, nil
}

// @param adtsHeader: 函数调用结束后，内部不持有该内存块
//
func (ctx *AdtsHeaderContext) Unpack(adtsHeader []byte) error {
	if len(adtsHeader) < AdtsHeaderLength {
		return ErrAac
	}

	br := nazabits.NewBitReader(adtsHeader)
	_ = br.SkipBits(16)
	v, _ := br.ReadBits8(2)
	ctx.AscCtx.AudioObjectType = v + 1
	ctx.AscCtx.SamplingFrequencyIndex, _ = br.ReadBits8(4)
	_ = br.SkipBits(1)
	ctx.AscCtx.ChannelConfiguration, _ = br.ReadBits8(3)
	_ = br.SkipBits(4)
	ctx.AdtsLength, _ = br.ReadBits16(13)
	return nil
}

// @param adtsHeader: 函数调用结束后，内部不持有该内存块
//
// @return asc: 内存块为独立新申请；函数调用结束后，内部不持有该内存块
//
func MakeAscWithAdtsHeader(adtsHeader []byte) (asc []byte, err error) {
	var ctx *AdtsHeaderContext
	if ctx, err = NewAdtsHeaderContext(adtsHeader); err != nil {
		return nil, err
	}
	return ctx.AscCtx.Pack(), nil
}
