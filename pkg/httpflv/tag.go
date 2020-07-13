// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv

import (
	"io"

	"github.com/q191201771/naza/pkg/bele"
)

type TagHeader struct {
	Type      uint8  // type
	DataSize  uint32 // body大小，不包含 header 和 prev tag size 字段
	Timestamp uint32 // 绝对时间戳，单位毫秒
	StreamID  uint32 // always 0
}

type Tag struct {
	Header TagHeader
	Raw    []byte // 结构为 (11字节的 tag header) + (body) + (4字节的 prev tag size)
}

func (tag *Tag) Payload() []byte {
	return tag.Raw[11 : len(tag.Raw)-4]
}

func (tag *Tag) IsMetadata() bool {
	return tag.Header.Type == TagTypeMetadata
}

func (tag *Tag) IsAVC() bool {
	return tag.Header.Type == TagTypeVideo && (tag.Raw[TagHeaderSize]&0xF == codecIDAVC)
}

func (tag *Tag) IsHEVC() bool {
	return tag.Header.Type == TagTypeVideo && (tag.Raw[TagHeaderSize]&0xF == codecIDHEVC)
}

func (tag *Tag) IsAVCKeySeqHeader() bool {
	return tag.Header.Type == TagTypeVideo && tag.Raw[TagHeaderSize] == AVCKeyFrame && tag.Raw[TagHeaderSize+1] == AVCPacketTypeSeqHeader
}

func (tag *Tag) IsHEVCKeySeqHeader() bool {
	return tag.Header.Type == TagTypeVideo && tag.Raw[TagHeaderSize] == HEVCKeyFrame && tag.Raw[TagHeaderSize+1] == HEVCPacketTypeSeqHeader
}

// AVC或HEVC的seq header
func (tag *Tag) IsVideoKeySeqHeader() bool {
	return tag.IsAVCKeySeqHeader() || tag.IsHEVCKeySeqHeader()
}

func (tag *Tag) IsAVCKeyNALU() bool {
	return tag.Header.Type == TagTypeVideo && tag.Raw[TagHeaderSize] == AVCKeyFrame && tag.Raw[TagHeaderSize+1] == AVCPacketTypeNALU
}

func (tag *Tag) IsHEVCKeyNALU() bool {
	return tag.Header.Type == TagTypeVideo && tag.Raw[TagHeaderSize] == HEVCKeyFrame && tag.Raw[TagHeaderSize+1] == HEVCPacketTypeNALU
}

// AVC或HEVC的关键帧
func (tag *Tag) IsVideoKeyNALU() bool {
	return tag.IsAVCKeyNALU() || tag.IsHEVCKeyNALU()
}

func (tag *Tag) IsAACSeqHeader() bool {
	return tag.Header.Type == TagTypeAudio && tag.Raw[TagHeaderSize]>>4 == SoundFormatAAC && tag.Raw[TagHeaderSize+1] == AACPacketTypeSeqHeader
}

func (tag *Tag) clone() (out Tag) {
	out.Header = tag.Header
	out.Raw = append(out.Raw, tag.Raw...)
	return
}

func (tag *Tag) ModTagTimestamp(timestamp uint32) {
	tag.Header.Timestamp = timestamp

	bele.BEPutUint24(tag.Raw[4:], timestamp&0xffffff)
	tag.Raw[7] = byte(timestamp >> 24)
}

// 打包一个序列化后的 tag 二进制buffer，包含 tag header，body，prev tag size
func PackHTTPFLVTag(t uint8, timestamp uint32, in []byte) []byte {
	out := make([]byte, TagHeaderSize+len(in)+prevTagSizeFieldSize)
	out[0] = t
	bele.BEPutUint24(out[1:], uint32(len(in)))
	bele.BEPutUint24(out[4:], timestamp&0xFFFFFF)
	out[7] = uint8(timestamp >> 24)
	out[8] = 0
	out[9] = 0
	out[10] = 0
	copy(out[11:], in)
	bele.BEPutUint32(out[TagHeaderSize+len(in):], uint32(TagHeaderSize+len(in)))
	return out
}

func parseTagHeader(rawHeader []byte) TagHeader {
	var h TagHeader
	h.Type = rawHeader[0]
	h.DataSize = bele.BEUint24(rawHeader[1:])
	h.Timestamp = (uint32(rawHeader[7]) << 24) + bele.BEUint24(rawHeader[4:])
	return h
}

func readTag(rd io.Reader) (tag Tag, err error) {
	rawHeader := make([]byte, TagHeaderSize)
	if _, err = io.ReadAtLeast(rd, rawHeader, TagHeaderSize); err != nil {
		return
	}
	header := parseTagHeader(rawHeader)

	needed := int(header.DataSize) + prevTagSizeFieldSize
	tag.Header = header
	tag.Raw = make([]byte, TagHeaderSize+needed)
	copy(tag.Raw, rawHeader)

	if _, err = io.ReadAtLeast(rd, tag.Raw[TagHeaderSize:], needed); err != nil {
		return
	}

	return
}
