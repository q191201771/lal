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
// ADTS
// e.g. ts
//
// StreamMuxConfig
//

var ErrAAC = errors.New("lal.aac: fxxk")

// Audio Data Transport Stream
type ADTS struct {
	audioObjectType        uint8
	samplingFrequencyIndex uint8
	channelConfiguration   uint8

	adtsHeader []byte
}

type SequenceHeader struct {
	soundFormat   uint8
	soundRate     uint8
	soundSize     uint8
	soundType     uint8
	aacPacketType uint8
}

// @param <asc> 2字节的AAC Audio Specifc Config
//              函数调用结束后，内部不持有<asc>内存块
//              注意，如果是rtmp/flv的message/tag，应去除Seq Header头部的2个字节
//
func (a *ADTS) InitWithAACAudioSpecificConfig(asc []byte) error {
	if len(asc) < 2 {
		nazalog.Warnf("aac seq header length invalid. len=%d", len(asc))
		return ErrAAC
	}

	// <ISO_IEC_14496-3.pdf>
	// <1.6.2.1 AudioSpecificConfig>, <page 33/110>
	// <1.5.1.1 Audio Object type definition>, <page 23/110>
	// <1.6.3.3 samplingFrequencyIndex>, <page 35/110>
	// <1.6.3.4 channelConfiguration>
	// --------------------------------------------------------
	// audio object type      [5b] 2=AAC LC
	// samplingFrequencyIndex [4b] 3=48000 4=44100
	// channelConfiguration   [4b] 2=left, right front speakers
	br := nazabits.NewBitReader(asc)
	a.audioObjectType, _ = br.ReadBits8(5)
	a.samplingFrequencyIndex, _ = br.ReadBits8(4)
	a.channelConfiguration, _ = br.ReadBits8(4)
	//nazalog.Debugf("%+v", a)

	if a.adtsHeader == nil {
		a.adtsHeader = make([]byte, 7)
	}

	return nil
}

// 获取ADTS头，由于ADTS头中的字段依赖包的长度，而每个包的长度不同，所以生成的每个包的ADTS头也不同
//
// @param <length> raw aac frame的大小
//                 注意，如果是rtmp/flv的message/tag，应去除Seq Header头部的2个字节
// @return 返回的内存块，内部会继续持有，重复使用
//
func (a *ADTS) CalcADTSHeader(length uint16) ([]byte, error) {
	if !a.HasInited() {
		nazalog.Warn("calc adts header but asc not inited.")
		return nil, ErrAAC
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

	// 加上自身adts头的7个字节
	length += 7

	bw := nazabits.NewBitWriter(a.adtsHeader)
	// Syncword 0(8) 1(4)
	bw.WriteBits16(12, 0xFFF)
	// ID, Layer, protection_absent 1(4)
	bw.WriteBits8(4, 0x1)
	// 2(2)
	bw.WriteBits8(2, a.audioObjectType-1)
	// 2(4)
	bw.WriteBits8(4, a.samplingFrequencyIndex)
	// private_bit 2(1)
	bw.WriteBits8(1, 0)
	// 2(1) 3(2)
	bw.WriteBits8(3, a.channelConfiguration)
	// origin/copy, home, copyright_identification_bit, copyright_identification_start 3(4)
	bw.WriteBits8(4, 0)
	// 3(2) 4(8) 5(3)
	bw.WriteBits16(13, length)
	// adts_buffer_fullness 5(5) 6(6)
	bw.WriteBits16(11, 0x7FF)
	// no_raw_data_blocks_in_frame 6(2)
	bw.WriteBits8(2, 0)

	return a.adtsHeader, nil
}

// 可用于判断，是否调用过ADTS.InitWithAACAudioSpecificConfig
func (a *ADTS) HasInited() bool {
	return a.adtsHeader != nil
}

// @param <b> rtmp/flv的message/tag的payload部分，包含前面2个字节
//            函数调用结束后，内部不持有<b>内存块
func ParseAACSeqHeader(b []byte) (sh SequenceHeader, adts ADTS, err error) {
	if len(b) < 4 {
		nazalog.Warnf("aac seq header length invalid. len=%d", len(b))
		err = ErrAAC
		return
	}

	// <spec-video_file_format_spec_v10.pdf>, <Audio tags, AUDIODATA>, <page 10/48>
	// ----------------------------------------------------------------------------
	// soundFormat    [4b] 10=AAC
	// soundRate      [2b] 3=44kHz. AAC always 3
	// soundSize      [1b] 0=snd8Bit, 1=snd16Bit
	// soundType      [1b] 0=sndMono, 1=sndStereo. AAC always 1
	// aacPackageType [8b] 0=seq header, 1=AAC raw
	br := nazabits.NewBitReader(b)
	sh.soundFormat, _ = br.ReadBits8(4)
	sh.soundRate, _ = br.ReadBits8(2)
	sh.soundSize, _ = br.ReadBits8(1)
	sh.soundType, _ = br.ReadBits8(1)
	sh.aacPacketType, _ = br.ReadBits8(8)
	//nazalog.Debugf("%s %+v", hex.Dump(payload[:4]), sh)

	err = adts.InitWithAACAudioSpecificConfig(b[2:])
	return
}

// @param <asc> 函数调用结束后，内部不继续持有<asc>内存块
//
// @return      返回的内存块为新申请的独立内存块
//
func BuildAACSeqHeader(asc []byte) ([]byte, error) {
	if len(asc) < 2 {
		return nil, ErrAAC
	}

	ret := make([]byte, 4)
	// <spec-video_file_format_spec_v10.pdf>, <Audio tags, AUDIODATA>, <page 10/48>
	ret[0] = 0xaf
	ret[1] = 0
	ret[2] = asc[0]
	ret[3] = asc[1]
	return ret, nil
}
