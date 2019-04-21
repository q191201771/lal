package httpflv

import (
	"github.com/q191201771/lal/bele"
	"io"
	"log"
)

var tagHeaderSize = 11

type TagHeader struct {
	t         uint8 // type
	dataSize  uint32
	timestamp uint32
	streamId  uint32 // always 0
}

func readTagHeader(rd io.Reader) (h *TagHeader, raw []byte, err error) {
	raw = make([]byte, tagHeaderSize)
	if _, err = io.ReadAtLeast(rd, raw, tagHeaderSize); err != nil {
		log.Println(err)
		return
	}

	h = &TagHeader{}
	h.t = raw[0]
	h.dataSize = bele.BeUInt24(raw[1:])
	h.timestamp = (uint32(raw[7]) << 24) + bele.BeUInt24(raw[4:])
	return
}
