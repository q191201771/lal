package httpflv

import (
	"github.com/q191201771/nezha/pkg/bele"
	"io"
)

// TODO chef: make these const
const TagHeaderSize int = 11
const prevTagSizeFieldSize int = 4

const (
	TagTypeMetadata uint8 = 18
	TagTypeVideo    uint8 = 9
	TagTypeAudio    uint8 = 8
)

const (
	frameTypeKey   uint8 = 1
	frameTypeInter uint8 = 2
)

const (
	codecIDAVC uint8 = 7
)

const (
	AVCKey   = frameTypeKey<<4 | codecIDAVC
	AVCInter = frameTypeInter<<4 | codecIDAVC
)

const (
	isAVCKeySeqHeader uint8 = 0
	AVCPacketTypeNalu uint8 = 1
)

const (
	SoundFormatAAC uint8 = 10
)

const (
	AACPacketTypeSeqHeader uint8 = 0
	AACPacketTypeRaw       uint8 = 1
)

type TagHeader struct {
	T         uint8  // type
	DataSize  uint32 // body大小，不包含header和prev tag size字段
	Timestamp uint32
	StreamID  uint32 // always 0
}

type Tag struct {
	Header TagHeader
	Raw    []byte // 结构为 (11字节的 tag header) + (body) + (4字节的 prev tag size)
}

func (tag *Tag) IsMetadata() bool {
	return tag.Header.T == TagTypeMetadata
}

func (tag *Tag) IsAVCKeySeqHeader() bool {
	return tag.Header.T == TagTypeVideo && tag.Raw[TagHeaderSize] == AVCKey && tag.Raw[TagHeaderSize+1] == isAVCKeySeqHeader
}

func (tag *Tag) IsAVCKeyNalu() bool {
	return tag.Header.T == TagTypeVideo && tag.Raw[TagHeaderSize] == AVCKey && tag.Raw[TagHeaderSize+1] == AVCPacketTypeNalu
}

func (tag *Tag) IsAACSeqHeader() bool {
	return tag.Header.T == TagTypeAudio && tag.Raw[TagHeaderSize]>>4 == SoundFormatAAC && tag.Raw[TagHeaderSize+1] == AACPacketTypeSeqHeader
}

func (tag *Tag) Payload() []byte {
	return tag.Raw[11 : len(tag.Raw)-4]
}

func IsMetadata(tag []byte) bool {
	return tag[0] == TagTypeMetadata
}

func IsAVCKeySeqHeader(tag []byte) bool {
	return tag[0] == TagTypeVideo && tag[TagHeaderSize] == AVCKey && tag[TagHeaderSize+1] == isAVCKeySeqHeader
}

func IsAVCKeyNalu(tag []byte) bool {
	return tag[0] == TagTypeVideo && tag[TagHeaderSize] == AVCKey && tag[TagHeaderSize+1] == AVCPacketTypeNalu
}

func IsAACSeqHeader(tag []byte) bool {
	return tag[0] == TagTypeAudio && tag[TagHeaderSize]>>4 == SoundFormatAAC && tag[TagHeaderSize+1] == AACPacketTypeSeqHeader
}

func PackHTTPFlvTag(t uint8, timestamp int, in []byte) []byte {
	out := make([]byte, TagHeaderSize+len(in)+prevTagSizeFieldSize)
	out[0] = t
	bele.BEPutUint24(out[1:], uint32(len(in)))
	bele.BEPutUint24(out[4:], uint32(timestamp&0xFFFFFF))
	out[7] = uint8(timestamp >> 24)
	out[8] = 0
	out[9] = 0
	out[10] = 0
	copy(out[11:], in)
	bele.BEPutUint32(out[TagHeaderSize+len(in):], uint32(TagHeaderSize+len(in)))
	return out
}

func ModTagTimestamp(tag *Tag, timestamp uint32) {
	tag.Header.Timestamp = timestamp

	bele.BEPutUint24(tag.Raw[4:], timestamp&0xffffff)
	tag.Raw[7] = byte(timestamp >> 24)
}

// 调用方需确保len(rawHeader) >= TagHeaderSize
func parseTagHeader(rawHeader []byte) TagHeader {
	var h TagHeader
	h.T = rawHeader[0]
	h.DataSize = bele.BEUint24(rawHeader[1:])
	h.Timestamp = (uint32(rawHeader[7]) << 24) + bele.BEUint24(rawHeader[4:])
	return h
}

func readTagHeader(rd io.Reader) (h TagHeader, rawHeader []byte, err error) {
	rawHeader = make([]byte, TagHeaderSize)
	if _, err = io.ReadAtLeast(rd, rawHeader, TagHeaderSize); err != nil {
		return
	}

	h.T = rawHeader[0]
	h.DataSize = bele.BEUint24(rawHeader[1:])
	h.Timestamp = (uint32(rawHeader[7]) << 24) + bele.BEUint24(rawHeader[4:])
	return
}

func (tag *Tag) cloneTag() *Tag {
	res := &Tag{}
	res.Header = tag.Header
	res.Raw = append(res.Raw, tag.Raw...)
	return res
}
