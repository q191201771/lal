package httpflv

import (
	"github.com/q191201771/lal/pkg/util/bele"
	"io"
)

// TODO chef: make these const
const tagHeaderSize int = 11
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
	T         uint8 // type
	DataSize  uint32
	Timestamp uint32
	StreamID  uint32 // always 0
}

type Tag struct {
	Header TagHeader
	Raw    []byte
}

func (tag *Tag) IsMetadata() bool {
	return tag.Header.T == TagTypeMetadata
}

func (tag *Tag) IsAVCKeySeqHeader() bool {
	return tag.Header.T == TagTypeVideo && tag.Raw[tagHeaderSize] == AVCKey && tag.Raw[tagHeaderSize+1] == isAVCKeySeqHeader
}

func (tag *Tag) IsAVCKeyNalu() bool {
	return tag.Header.T == TagTypeVideo && tag.Raw[tagHeaderSize] == AVCKey && tag.Raw[tagHeaderSize+1] == AVCPacketTypeNalu
}

func (tag *Tag) IsAACSeqHeader() bool {
	return tag.Header.T == TagTypeAudio && tag.Raw[tagHeaderSize]>>4 == SoundFormatAAC && tag.Raw[tagHeaderSize+1] == AACPacketTypeSeqHeader
}

func IsMetadata(tag []byte) bool {
	return tag[0] == TagTypeMetadata
}

func IsAVCKeySeqHeader(tag []byte) bool {
	return tag[0] == TagTypeVideo && tag[tagHeaderSize] == AVCKey && tag[tagHeaderSize+1] == isAVCKeySeqHeader
}

func IsAVCKeyNalu(tag []byte) bool {
	return tag[0] == TagTypeVideo && tag[tagHeaderSize] == AVCKey && tag[tagHeaderSize+1] == AVCPacketTypeNalu
}

func IsAACSeqHeader(tag []byte) bool {
	return tag[0] == TagTypeAudio && tag[tagHeaderSize]>>4 == SoundFormatAAC && tag[tagHeaderSize+1] == AACPacketTypeSeqHeader
}

func PackHTTPFlvTag(t uint8, timestamp int, in []byte) []byte {
	out := make([]byte, tagHeaderSize+len(in)+prevTagSizeFieldSize)
	out[0] = t
	bele.BEPutUint24(out[1:], uint32(len(in)))
	bele.BEPutUint24(out[4:], uint32(timestamp&0xFFFFFF))
	out[7] = uint8(timestamp >> 24)
	out[8] = 0
	out[9] = 0
	out[10] = 0
	copy(out[11:], in)
	bele.BEPutUint32(out[tagHeaderSize+len(in):], uint32(tagHeaderSize+len(in)))
	return out
}

func readTagHeader(rd io.Reader) (h TagHeader, rawHeader []byte, err error) {
	rawHeader = make([]byte, tagHeaderSize)
	if _, err = io.ReadAtLeast(rd, rawHeader, tagHeaderSize); err != nil {
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
