// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package aac

import (
	"encoding/hex"
	"errors"

	"github.com/q191201771/naza/pkg/nazabits"

	"github.com/q191201771/naza/pkg/nazalog"
)

var ErrAAC = errors.New("lal.aac: fxxk")

// Audio Data Transport Stream
type ADTS struct {
	audioObjectType        uint8
	samplingFrequencyIndex uint8
	channelConfiguration   uint8

	adtsHeader []byte
}

// 传入AAC Sequence Header，调用GetADTS时需要使用
// @param <payload> rtmp message payload，包含前面2个字节
func (a *ADTS) PutAACSequenceHeader(payload []byte) error {
	if len(payload) < 4 {
		nazalog.Warnf("aac seq header length invalid. len=%d", len(payload))
		return ErrAAC
	}

	// TODO chef: 把Seq Header头两字节的解析和ADTS的内容分离开
	// <spec-video_file_format_spec_v10.pdf>, <Audio tags, AUDIODATA>, <page 10/48>
	// ----------------------------------------------------------------------------
	// soundFormat    [4b] 10=AAC
	// soundRate      [2b] 3=44kHz. AAC always 3
	// soundSize      [1b] 0=snd8Bit, 1=snd16Bit
	// soundType      [1b] 0=sndMono, 1=sndStereo. AAC always 1
	// aacPackageType [8b] 0=seq header, 1=AAC raw
	br := nazabits.NewBitReader(payload)
	soundFormat, _ := br.ReadBits8(4)
	soundRate, _ := br.ReadBits8(2)
	soundSize, _ := br.ReadBits8(1)
	soundType, _ := br.ReadBits8(1)
	aacPacketType, _ := br.ReadBits8(8)
	nazalog.Debugf("%s %d %d %d %d %d", hex.Dump(payload[:4]), soundFormat, soundRate, soundSize, soundType, aacPacketType)

	// <ISO_IEC_14496-3.pdf>
	// <1.6.2.1 AudioSpecificConfig>, <page 33/110>
	// <1.5.1.1 Audio Object type definition>, <page 23/110>
	// <1.6.3.3 samplingFrequencyIndex>, <page 35/110>
	// <1.6.3.4 channelConfiguration>
	// --------------------------------------------------------
	// audio object type      [5b] 2=AAC LC
	// samplingFrequencyIndex [4b] 3=48000 4=44100
	// channelConfiguration   [4b] 2=left, right front speakers
	a.audioObjectType, _ = br.ReadBits8(5)
	a.samplingFrequencyIndex, _ = br.ReadBits8(4)
	a.channelConfiguration, _ = br.ReadBits8(4)
	nazalog.Debugf("%+v", a)

	if a.adtsHeader == nil {
		a.adtsHeader = make([]byte, 7)
	}

	return nil
}

// 获取ADTS头，注意，由于ADTS头依赖包的长度，而每个包的长度不同，所以生成的每个包的ADTS头也不同
// @param <length> rtmp message payload长度，包含前面2个字节
func (a *ADTS) GetADTS(length uint16) []byte {
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

	// 减去头两字节，再加上自身adts头的7个字节
	length += 5

	bw := nazabits.NewBitWriter(a.adtsHeader)
	bw.WriteBits16(12, 0xFFF)                  // Syncword 0(8) 1(4)
	bw.WriteBits8(4, 0x1)                      // ID, Layer, protection_absent 1(4)
	bw.WriteBits8(2, a.audioObjectType-1)      // 2(2)
	bw.WriteBits8(4, a.samplingFrequencyIndex) // 2(4)
	bw.WriteBits8(1, 0)                        // private_bit 2(1)
	bw.WriteBits8(3, a.channelConfiguration)   // 2(1) 3(2)
	bw.WriteBits8(4, 0)                        // origin/copy, home, copyright_identification_bit, copyright_identification_start 3(4)
	bw.WriteBits16(13, length)                 // 3(2) 4(8) 5(3)
	bw.WriteBits16(11, 0x7FF)                  // adts_buffer_fullness 5(5) 6(6)
	bw.WriteBits8(2, 0)                        // no_raw_data_blocks_in_frame 6(2)

	return a.adtsHeader
}

func (a *ADTS) IsNil() bool {
	return a.adtsHeader == nil
}
