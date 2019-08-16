package aac

import (
	"encoding/hex"
	"github.com/q191201771/lal/pkg/util/log"
	"io"
)

var adts ADTS

type ADTS struct {
	audioObjectType        uint8
	samplingFrequencyIndex uint8
	channelConfig          uint8

	frameLengthFlag   uint8
	dependOnCoreCoder uint8
	extensionFlag     uint8
	adtsHeader        []byte
}

// @param <payload> rtmp message payload，包含前面2个字节
func (obj *ADTS) PutAACSequenceHeader(payload []byte) {
	soundFormat := payload[0] >> 4        // 10=AAC
	soundRate := (payload[0] >> 2) & 0x03 // 3=44kHz. For AAC: always 3
	soundSize := (payload[0] >> 1) & 0x01 // 0=snd8Bit 1=snd16Bit
	soundType := payload[0] & 0x01        // For AAC: always 1

	//aacPacketType := payload[1] // 0:sequence header 1:AAC raw

	obj.audioObjectType = (payload[2] & 0xf8) >> 3                              // 5bit 编码结构类型
	obj.samplingFrequencyIndex = ((payload[2] & 0x07) << 1) | (payload[3] >> 7) // 4bit 音频采样率索引值
	obj.channelConfig = (payload[3] & 0x78) >> 3                                // 4bit 音频输出声道
	obj.frameLengthFlag = (payload[3] & 0x04) >> 2                              // 1bit
	obj.dependOnCoreCoder = (payload[3] & 0x02) >> 1                            // 1bit
	obj.extensionFlag = payload[3] & 0x01                                       // 1bit

	log.Debugf(hex.Dump(payload[:4]))
	log.Debugf("%d %d %d %d", soundFormat, soundRate, soundSize, soundType)
	log.Debugf("%+v", obj)
}

// @param <length> rtmp message payload长度，包含前面2个字节
func (obj *ADTS) GetADTS(length uint16) []byte {
	// adts fixed header:
	// syncword                           12bit (0: **** ****, 1: **** 0000) 0xfff
	// ID                                  1bit (1: 0000 *000)               0     0:MPEG-4 1:MPEG-2
	// layer                               2bit (1: 0000 0**0)               0
	// protection_absent                   1bit (1: 0000 000*)               1
	// profile                             2bit (2: **00 0000)
	// sampling_frequency_index            4bit (2: 00** **00)
	// private_bit                         1bit (2: 0000 00*0)               0
	// channel_configuration               3bit (2: 0000 000*, 3: **00 0000)
	// origin_copy                         1bit (3: 00*0 0000)               0
	// home                                1bit (3: 000* 0000)               0
	//
	// adts variable_header:
	// copyright_identification_bit        1bit (3: 0000 *000)               0
	// copyright_identification_start      1bit (3: 0000 0*00)               0
	// aac_frame_length                   13bit (3: 0000 00**, 5: ***0 0000)
	// adts_buffer_fullness               11bit (5: 000* ****, 6: **** **00) 0x7ff
	// number_of_raw_data_blocks_in_frame  2bit (6: 0000 00**)               0
	if obj.adtsHeader == nil {
		obj.adtsHeader = make([]byte, 7)
	}
	obj.adtsHeader[0] = 0xff
	obj.adtsHeader[1] = 0xf1
	obj.adtsHeader[2] = ((obj.audioObjectType - 1) << 6) & 0xc0
	obj.adtsHeader[2] |= (obj.samplingFrequencyIndex << 2) & 0x3c
	obj.adtsHeader[2] |= (obj.channelConfig >> 2) & 0x01
	obj.adtsHeader[3] = (obj.channelConfig << 6) & 0xc0

	// 减去前面2个字节，加上adts的7个字节
	length = length - 2 + 7
	obj.adtsHeader[3] += uint8(length >> 11) // TODO chef: 为什么这样做，应该只是使用2个字节，取5个再相加是否会超出？
	obj.adtsHeader[4] = uint8((length & 0x7ff) >> 3)
	obj.adtsHeader[5] = uint8((length & 0x07) << 5)

	obj.adtsHeader[5] |= 0x1f
	obj.adtsHeader[6] = 0xfc

	return obj.adtsHeader
}

// @param <payload> rtmp message payload部分
func CaptureAAC(w io.Writer, payload []byte) {
	if payload[1] == 0 {
		adts.PutAACSequenceHeader(payload)
		return
	}

	_, _ = w.Write(adts.GetADTS(uint16(len(payload))))
	_, _ = w.Write(payload[2:])
}
