package rtmp

import (
	"github.com/q191201771/lal/pkg/util/bele"
	"io"
)

// 读取chunk，并组织chunk，生成message返回给上层

type ChunkComposer struct {
	peerChunkSize int
	csid2stream   map[int]*Stream
}

func NewChunkComposer() *ChunkComposer {
	return &ChunkComposer{
		peerChunkSize: defaultChunkSize,
		csid2stream:   make(map[int]*Stream),
	}
}

func (c *ChunkComposer) SetPeerChunkSize(val int) {
	c.peerChunkSize = val
}

func (c *ChunkComposer) GetPeerChunkSize() int {
	return c.peerChunkSize
}

type CompleteMessageCB func(stream *Stream) error

func (c *ChunkComposer) RunLoop(reader io.Reader, cb CompleteMessageCB) error {
	bootstrap := make([]byte, 11)

	for {
		if _, err := io.ReadAtLeast(reader, bootstrap[:1], 1); err != nil {
			return err
		}

		// 5.3.1.1. Chunk Basic Header
		fmt := (bootstrap[0] >> 6) & 0x03
		csid := int(bootstrap[0] & 0x3f)

		switch csid {
		case 0:
			if _, err := io.ReadAtLeast(reader, bootstrap[:1], 1); err != nil {
				return err
			}
			csid = 64 + int(bootstrap[0])
		case 1:
			if _, err := io.ReadAtLeast(reader, bootstrap[:2], 2); err != nil {
				return err
			}
			csid = 64 + int(bootstrap[0]) + int(bootstrap[1])*256
		default:
			// noop
		}

		stream := c.getOrCreateStream(csid)

		// 5.3.1.2. Chunk Message Header
		switch fmt {
		case 0:
			if _, err := io.ReadAtLeast(reader, bootstrap[:11], 11); err != nil {
				return err
			}
			stream.header.Timestamp = int(bele.BEUint24(bootstrap))
			stream.timestampAbs = stream.header.Timestamp
			stream.msgLen = int(bele.BEUint24(bootstrap[3:]))
			stream.header.MsgTypeID = int(bootstrap[6])
			stream.header.MsgStreamID = int(bele.LEUint32(bootstrap[7:]))

			stream.msg.reserve(stream.msgLen)
		case 1:
			if _, err := io.ReadAtLeast(reader, bootstrap[:7], 7); err != nil {
				return err
			}
			stream.header.Timestamp = int(bele.BEUint24(bootstrap))
			stream.timestampAbs += stream.header.Timestamp
			stream.msgLen = int(bele.BEUint24(bootstrap[3:]))
			stream.header.MsgTypeID = int(bootstrap[6])

			stream.msg.reserve(stream.msgLen)
		case 2:
			if _, err := io.ReadAtLeast(reader, bootstrap[:3], 3); err != nil {
				return err
			}
			stream.header.Timestamp = int(bele.BEUint24(bootstrap))
			stream.timestampAbs += stream.header.Timestamp

		case 3:
			// noop
		}

		// 5.3.1.3 Extended Timestamp
		if stream.header.Timestamp == maxTimestampInMessageHeader {
			if _, err := io.ReadAtLeast(reader, bootstrap[:4], 4); err != nil {
				return err
			}
			stream.header.Timestamp = int(bele.BEUint32(bootstrap))
			switch fmt {
			case 0:
				stream.timestampAbs = stream.header.Timestamp
			case 1:
				fallthrough
			case 2:
				stream.timestampAbs = stream.timestampAbs - maxTimestampInMessageHeader + stream.header.Timestamp
			case 3:
				// noop
			}
		}

		var neededSize int
		if stream.msgLen <= c.peerChunkSize {
			neededSize = stream.msgLen
		} else {
			neededSize = stream.msgLen - stream.msg.len()
			if neededSize > c.peerChunkSize {
				neededSize = c.peerChunkSize
			}
		}

		//stream.msg.reserve(neededSize)
		if _, err := io.ReadAtLeast(reader, stream.msg.buf[stream.msg.e:stream.msg.e+neededSize], neededSize); err != nil {
			return err
		}
		stream.msg.produced(neededSize)

		if stream.msg.len() == stream.msgLen {
			// 对端设置了chunk size
			if stream.header.MsgTypeID == typeidSetChunkSize {
				val := int(bele.BEUint32(stream.msg.buf))
				c.SetPeerChunkSize(val)
			}

			stream.header.CSID = csid
			stream.header.MsgLen = stream.msgLen
			if err := cb(stream); err != nil {
				return err
			}
			stream.msg.clear()
		}
		if stream.msg.len() > stream.msgLen {
			panic(0)
		}
	}
}

func (c *ChunkComposer) getOrCreateStream(csid int) *Stream {
	stream, exist := c.csid2stream[csid]
	if !exist {
		stream = NewStream()
		c.csid2stream[csid] = stream
	}
	return stream
}
