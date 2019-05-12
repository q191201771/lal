package httpflv

import (
	"github.com/q191201771/lal/bele"
	"io"
)

// TODO chef: make these const
const tagHeaderSize int = 11

var (
	tagTypeMetadata uint8 = 18
	tagTypeVideo    uint8 = 9
	tagTypeAudio    uint8 = 8
)

var (
	frameTypeKey   uint8 = 1
	frameTypeInter uint8 = 2
)

var (
	codecIDAVC uint8 = 7
)

var (
	AVCKey   = frameTypeKey<<4 | codecIDAVC
	AVCInter = frameTypeInter<<4 | codecIDAVC
)

var (
	isAVCKeySeqHeader uint8 = 0
	AVCPacketTypeNalu uint8 = 1
)

var (
	SoundFormatAAC uint8 = 10
)

var (
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

func (tag *Tag) IsMetadata() bool {
	return tag.Header.T == tagTypeMetadata
}

func (tag *Tag) IsAVCKeySeqHeader() bool {
	return tag.Header.T == tagTypeVideo && tag.Raw[tagHeaderSize] == AVCKey && tag.Raw[tagHeaderSize+1] == isAVCKeySeqHeader
}

func (tag *Tag) IsAVCKeyNalu() bool {
	return tag.Header.T == tagTypeVideo && tag.Raw[tagHeaderSize] == AVCKey && tag.Raw[tagHeaderSize+1] == AVCPacketTypeNalu
}

func (tag *Tag) IsAACSeqHeader() bool {
	return tag.Header.T == tagTypeAudio && tag.Raw[tagHeaderSize]>>4 == SoundFormatAAC && tag.Raw[tagHeaderSize+1] == AACPacketTypeSeqHeader
}

func (tag *Tag) cloneTag() *Tag {
	res := &Tag{}
	res.Header = tag.Header
	res.Raw = append(res.Raw, tag.Raw...)
	return res
}
