package rtmp

import "github.com/q191201771/lal/log"

var initMsgLen = 4096

type Header struct {
	csid   int
	msgLen int

	timestamp int // NOTICE 是header中的时间戳，可能是绝对的，也可能是相对的。
	// 如果需要绝对时间戳，应该使用Stream中的timestampAbs
	MsgTypeID   int // 8 audio 9 video 18 metadata
	msgStreamID int
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
	nn := cap(msg.buf) - msg.e
	if nn > n {
		return
	}
	for nn < n {
		nn <<= 1
	}
	nb := make([]byte, cap(msg.buf)+nn)
	copy(nb, msg.buf[msg.b:msg.e])
	msg.buf = nb
	log.Debugf("reserve. need:%d left:%d %d %d", n, nn, len(msg.buf), cap(msg.buf))
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

func (msg *StreamMsg) peekStringWithType() (string, error) {
	str, _, err := AMF0.ReadString(msg.buf[msg.b:msg.e])
	return str, err
}

func (msg *StreamMsg) readStringWithType() (string, error) {
	str, l, err := AMF0.ReadString(msg.buf[msg.b:msg.e])
	if err == nil {
		msg.consumed(l)
	}
	return str, err
}

func (msg *StreamMsg) readNumberWithType() (int, error) {
	val, l, err := AMF0.ReadNumber(msg.buf[msg.b:msg.e])
	if err == nil {
		msg.consumed(l)
	}
	return int(val), err
}

func (msg *StreamMsg) readObjectWithType() (map[string]interface{}, error) {
	obj, l, err := AMF0.ReadObject(msg.buf[msg.b:msg.e])
	if err == nil {
		msg.consumed(l)
	}
	return obj, err
}

func (msg *StreamMsg) readNull() error {
	l, err := AMF0.ReadNull(msg.buf[msg.b:msg.e])
	if err == nil {
		msg.consumed(l)
	}
	return err
}
