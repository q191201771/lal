package rtmp

import (
	"github.com/q191201771/lal/pkg/bele"
)

// TODO chef: 这里所有的chunk的格式判断是参考的前一个message的字段。实际上应该参考当前message的前一个chunk的字段吧？
func Message2Chunks(message []byte, header *Header, prevHeader *Header, chunkSize int) ([]byte, error) {
	if header.csid < minCSID || header.csid > maxCSID {
		return nil, rtmpErr
	}

	numOfChunk := len(message) / chunkSize
	lastChunkSize := chunkSize
	if len(message)%chunkSize != 0 {
		numOfChunk++
		lastChunkSize = len(message) % chunkSize
	}

	maxNeededLen := (chunkSize + maxHeaderSize) * numOfChunk
	out := make([]byte, maxNeededLen)
	index := 0

	fmt := uint8(0)
	var timestamp int
	if prevHeader == nil {
		timestamp = header.timestamp
	} else {
		if header.msgStreamID == prevHeader.msgStreamID {
			fmt++
			if header.msgLen == prevHeader.msgLen && header.MsgTypeID == prevHeader.MsgTypeID {
				fmt++
				if header.timestamp == prevHeader.timestamp {
					fmt++
				}
			}
			timestamp = header.timestamp - prevHeader.timestamp
		} else {
			timestamp = header.timestamp
		}
	}

	out[index] = fmt << 6

	if header.csid >= 2 && header.csid <= 63 {
		out[index] |= uint8(header.csid)
		index++
	} else if header.csid >= 64 && header.csid <= 319 {
		// value 0
		index++
		out[index] = uint8(header.csid - 64)
		index++
	} else {
		out[index] |= 1
		index++
		out[index] = uint8(header.csid - 64)
		index++
		out[index] = uint8((header.csid - 64) >> 8)
		index++
	}

	if fmt <= 2 {
		if timestamp > maxTimestampInMessageHeader {
			bele.BEPutUint24(out[index:], uint32(maxTimestampInMessageHeader))
		} else {
			bele.BEPutUint24(out[index:], uint32(timestamp))
		}
		index += 3

		if fmt <= 1 {
			bele.BEPutUint24(out[index:], uint32(header.msgLen))
			index += 3
			out[index] = uint8(header.MsgTypeID)
			index++

			if fmt == 0 {
				bele.BEPutUint24(out[index:], uint32(header.msgStreamID))
				index += 3
			}
		}
	}

	if timestamp > maxTimestampInMessageHeader {
		bele.BEPutUint32(out[index:], uint32(timestamp))
		index += 4
	}

	headLen := index
	for i := 0; i < numOfChunk; i++ {
		if i != 0 {
			copy(out[index:], out[0:headLen])
			index += headLen
		}
		if i != numOfChunk-1 {
			copy(out[index:], message[i*chunkSize:i*chunkSize+chunkSize])
			index += chunkSize
		} else {
			copy(out[index:], message[i*chunkSize:i*chunkSize+lastChunkSize])
			index += lastChunkSize
		}
	}

	return out[:index], nil
}
