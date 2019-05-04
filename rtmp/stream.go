package rtmp

import "github.com/q191201771/lal/log"

var initMsgLen = 4096

type Header struct {
	csid        int
	timestamp   int
	msgLen      int
	msgTypeId   int
	msgStreamId int
}

type StreamMsg struct {
	buf []byte
	b   int
	e   int
}

type Stream struct {
	header       Header
	msgLen       int // TODO chef: needed? dup with Header's
	timestampAbs int

	msg StreamMsg
}

func NewStream() *Stream {
	return &Stream{
		msg: StreamMsg{
			buf: make([]byte, initMsgLen),
		},
	}
}

func (msg *StreamMsg) reserve(n int) {
	nn := cap(msg.buf)
	if nn > n {
		return
	}
	for nn < n {
		nn <<= 1
	}
	msg.buf = make([]byte, nn)
	log.Debugf("reserve. %d %d", n, nn)
}

func (msg *StreamMsg) len() int {
	return msg.e - msg.b
}

func (msg *StreamMsg) produced(n int) {
	msg.e += n
}

func (msg *StreamMsg) consumed(n int) {
	msg.b += n
}

func (msg *StreamMsg) clear() {
	msg.b = 0
	msg.e = 0
}
