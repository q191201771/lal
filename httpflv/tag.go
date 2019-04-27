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
	codecIdAvc uint8 = 7
)

var (
	AvcKey   = frameTypeKey<<4 | codecIdAvc
	AvcInter = frameTypeInter<<4 | codecIdAvc
)

var (
	AvcPacketTypeSeqHeader uint8 = 0
	AvcPacketTypeNalu      uint8 = 1
)

var (
	SoundFormatAac uint8 = 10
)

var (
	AacPacketTypeSeqHeader uint8 = 0
	AacPacketTypeRaw       uint8 = 1
)

type TagHeader struct {
	T         uint8 // type
	DataSize  uint32
	Timestamp uint32
	StreamId  uint32 // always 0
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
	h.DataSize = bele.BeUInt24(rawHeader[1:])
	h.Timestamp = (uint32(rawHeader[7]) << 24) + bele.BeUInt24(rawHeader[4:])
	return
}

func (tag *Tag) isMetaData() bool {
	return tag.Header.T == tagTypeMetadata
}

func (tag *Tag) isAvcKeySeqHeader() bool {
	return tag.Header.T == tagTypeVideo && tag.Raw[tagHeaderSize] == AvcKey && tag.Raw[tagHeaderSize+1] == AvcPacketTypeSeqHeader
}

func (tag *Tag) isAvcKeyNalu() bool {
	return tag.Header.T == tagTypeVideo && tag.Raw[tagHeaderSize] == AvcKey && tag.Raw[tagHeaderSize+1] == AvcPacketTypeNalu
}

func (tag *Tag) isAacSeqHeader() bool {
	return tag.Header.T == tagTypeAudio && tag.Raw[tagHeaderSize]>>4 == SoundFormatAac && tag.Raw[tagHeaderSize+1] == AacPacketTypeSeqHeader
}

func (tag *Tag) cloneTag() *Tag {
	res := &Tag{}
	res.Header = tag.Header
	res.Raw = append(res.Raw, tag.Raw...)
	return res
}
