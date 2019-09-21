package rtmp

// chunk_composer.go
// @pure
// 读取chunk，并组织chunk，生成message返回给上层

import (
	"github.com/q191201771/nezha/pkg/bele"
	"io"
)

type ChunkComposer struct {
	peerChunkSize uint32
	csid2stream   map[int]*Stream
}

func NewChunkComposer() *ChunkComposer {
	return &ChunkComposer{
		peerChunkSize: defaultChunkSize,
		csid2stream:   make(map[int]*Stream),
	}
}

func (c *ChunkComposer) SetPeerChunkSize(val uint32) {
	c.peerChunkSize = val
}

func (c *ChunkComposer) GetPeerChunkSize() uint32 {
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
			// 包头中为绝对时间戳
			stream.header.Timestamp = bele.BEUint24(bootstrap)
			stream.timestampAbs = stream.header.Timestamp
			stream.header.MsgLen = bele.BEUint24(bootstrap[3:])
			stream.header.MsgTypeID = bootstrap[6]
			stream.header.MsgStreamID = int(bele.LEUint32(bootstrap[7:]))

			stream.msg.reserve(stream.header.MsgLen)
		case 1:
			if _, err := io.ReadAtLeast(reader, bootstrap[:7], 7); err != nil {
				return err
			}
			// 包头中为相对时间戳
			stream.header.Timestamp = bele.BEUint24(bootstrap)
			stream.timestampAbs += stream.header.Timestamp
			stream.header.MsgLen = bele.BEUint24(bootstrap[3:])
			stream.header.MsgTypeID = bootstrap[6]

			stream.msg.reserve(stream.header.MsgLen)
		case 2:
			if _, err := io.ReadAtLeast(reader, bootstrap[:3], 3); err != nil {
				return err
			}
			// 包头中为相对时间戳
			stream.header.Timestamp = bele.BEUint24(bootstrap)
			stream.timestampAbs += stream.header.Timestamp

		case 3:
			// noop
		}

		// 5.3.1.3 Extended Timestamp
		// 使用ffmpeg推流时，发现时间戳超过3字节最大值后，即使是fmt3(即包头大小为0)，依然存在ext ts字段
		// 所以这里我将 `==` 的判断改成了 `>=`
		// TODO chef: 测试其他客户端和ext ts相关的表现
		//if stream.header.Timestamp == maxTimestampInMessageHeader {
		if stream.header.Timestamp >= maxTimestampInMessageHeader {
			if _, err := io.ReadAtLeast(reader, bootstrap[:4], 4); err != nil {
				return err
			}
			stream.header.Timestamp = bele.BEUint32(bootstrap)
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
		//stream.header.CSID = csid
		//log.Debugf("CHEFGREPME tag1 fmt:%d header:%+v csid:%d len:%d ts:%d", fmt, stream.header, csid, stream.header.MsgLen, stream.timestampAbs)

		var neededSize uint32
		if stream.header.MsgLen <= c.peerChunkSize {
			neededSize = stream.header.MsgLen
		} else {
			neededSize = stream.header.MsgLen - stream.msg.len()
			if neededSize > c.peerChunkSize {
				neededSize = c.peerChunkSize
			}
		}

		//stream.msg.reserve(neededSize)
		if _, err := io.ReadAtLeast(reader, stream.msg.buf[stream.msg.e:stream.msg.e+neededSize], int(neededSize)); err != nil {
			return err
		}
		stream.msg.produced(neededSize)

		if stream.msg.len() == stream.header.MsgLen {
			// 对端设置了chunk size
			if stream.header.MsgTypeID == typeidSetChunkSize {
				val := bele.BEUint32(stream.msg.buf)
				c.SetPeerChunkSize(val)
			}

			stream.header.CSID = csid
			//log.Debugf("CHEFGREPME %+v %d %d", stream.header, stream.timestampAbs, stream.header.MsgLen)
			if err := cb(stream); err != nil {
				return err
			}
			stream.msg.clear()
		}
		if stream.msg.len() > stream.header.MsgLen {
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
