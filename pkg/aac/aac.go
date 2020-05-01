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
	"io"

	"github.com/q191201771/naza/pkg/nazabits"

	log "github.com/q191201771/naza/pkg/nazalog"
)

// TODO chef: 把Seq Header头两字节的解析和ADTS的内容分离开

var adts ADTS

// Audio Data Transport Stream
type ADTS struct {
	audioObjectType        uint8
	samplingFrequencyIndex uint8
	channelConfiguration   uint8

	adtsHeader []byte
}

// 传入AAC Sequence Header，调用GetADTS时需要使用
// @param <payload> rtmp message payload，包含前面2个字节
func (obj *ADTS) PutAACSequenceHeader(payload []byte) {
	// <spec-video_file_format_spec_v10.pdf>, <Audio tags, AUDIODATA>, <page 10/48>
	// ----------------------------------------------------------------------------
	// soundFormat    [4b] 10=AAC
	// soundRate      [2b] 3=44kHz. AAC always 3
	// soundSize      [1b] 0=snd8Bit, 1=snd16Bit
	// soundType      [1b] 0=sndMono, 1=sndStereo. AAC always 1
	// aacPackageType [8b] 0=seq header, 1=AAC raw
	soundFormat := nazabits.GetBits8(payload[0], 4, 4)
	soundRate := nazabits.GetBits8(payload[0], 2, 2)
	soundSize := nazabits.GetBit8(payload[0], 1)
	soundType := nazabits.GetBit8(payload[0], 0)
	aacPacketType := payload[1]
	log.Debugf("%s %d %d %d %d %d", hex.Dump(payload[:4]), soundFormat, soundRate, soundSize, soundType, aacPacketType)

	// <ISO_IEC_14496-3.pdf>
	// <1.6.2.1 AudioSpecificConfig>, <page 33/110>
	// <1.5.1.1 Audio Object type definition>, <page 23/110>
	// <1.6.3.3 samplingFrequencyIndex>, <page 35/110>
	// <1.6.3.4 channelConfiguration>
	// --------------------------------------------------------
	// audio object type      [5b] 2=AAC LC
	// samplingFrequencyIndex [4b] 3=48000
	// channelConfiguration   [4b] 2=left, right front speakers
	obj.audioObjectType = uint8(nazabits.GetBits16(payload[2:], 11, 5))
	obj.samplingFrequencyIndex = uint8(nazabits.GetBits16(payload[2:], 7, 4))
	obj.channelConfiguration = uint8(nazabits.GetBits16(payload[2:], 3, 4))
	log.Debugf("%+v", obj)
}

// 获取 ADTS 头，注意，由于ADTS头依赖包的长度，而每个包的长度不同，所以生成的每个包的 ADTS 头也不同
// @param <length> rtmp message payload长度，包含前面2个字节
func (obj *ADTS) GetADTS(length uint16) []byte {
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

	if obj.adtsHeader == nil {
		obj.adtsHeader = make([]byte, 7)
	}
	obj.adtsHeader[0] = 0xff
	obj.adtsHeader[1] = 0xf1
	obj.adtsHeader[2] = ((obj.audioObjectType - 1) << 6) & 0xc0
	obj.adtsHeader[2] |= (obj.samplingFrequencyIndex << 2) & 0x3c
	obj.adtsHeader[2] |= (obj.channelConfiguration >> 2) & 0x01
	obj.adtsHeader[3] = (obj.channelConfiguration << 6) & 0xc0

	// 减去前面2个字节，再加上加上adts的7个字节
	length += 5
	obj.adtsHeader[3] += uint8(length >> 11) // TODO chef: 为什么这样做，应该只是使用2个字节，取5个再相加是否会超出？
	obj.adtsHeader[4] = uint8((length & 0x7ff) >> 3)
	obj.adtsHeader[5] = uint8((length & 0x07) << 5)

	obj.adtsHeader[5] |= 0x1f
	obj.adtsHeader[6] = 0xfc

	return obj.adtsHeader
}

// 将 rtmp AAC 传入，输出带 ADTS 头的 AAC ES流
// @param <payload> rtmp message payload部分
func CaptureAAC(w io.Writer, payload []byte) {
	if payload[1] == 0 {
		adts.PutAACSequenceHeader(payload)
		return
	}

	_, _ = w.Write(adts.GetADTS(uint16(len(payload))))
	_, _ = w.Write(payload[2:])
}
