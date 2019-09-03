package rtmp

// chunk_divider.go
// @pure
// 将message切割成chunk

import (
	"github.com/apex/log"
	"github.com/q191201771/nezha/pkg/bele"
)

// TODO chef: 新的message的第一个chunk始终使用fmt0格式，不参考前一个message
func Message2Chunks(message []byte, header *Header, chunkSize int) []byte {
	return message2Chunks(message, header, nil, chunkSize)
}

// TODO chef: 返回值直接传入
func calcHeader(header *Header, prevHeader *Header) []byte {
	var index int
	out := make([]byte, 16)

	// 计算fmt和timestamp
	fmt := uint8(0)
	var timestamp uint32
	if prevHeader == nil {
		timestamp = header.Timestamp
	} else {
		if header.MsgStreamID == prevHeader.MsgStreamID {
			fmt++
			if header.MsgLen == prevHeader.MsgLen && header.MsgTypeID == prevHeader.MsgTypeID {
				fmt++
				if header.Timestamp == prevHeader.Timestamp {
					fmt++
				}
			}
			timestamp = header.Timestamp - prevHeader.Timestamp
		} else {
			timestamp = header.Timestamp
		}
	}

	// 设置fmt
	out[index] = fmt << 6

	// 设置csid
	if header.CSID >= 2 && header.CSID <= 63 {
		out[index] |= uint8(header.CSID)
		index++
	} else if header.CSID >= 64 && header.CSID <= 319 {
		// value 0
		index++
		out[index] = uint8(header.CSID - 64)
		index++
	} else {
		out[index] |= 1
		index++
		out[index] = uint8(header.CSID - 64)
		index++
		out[index] = uint8((header.CSID - 64) >> 8)
		index++
	}

	// 设置timestamp msgLen msgTypeID msgStreamID
	if fmt <= 2 {
		if timestamp > maxTimestampInMessageHeader {
			bele.BEPutUint24(out[index:], maxTimestampInMessageHeader)
		} else {
			bele.BEPutUint24(out[index:], timestamp)
		}
		index += 3

		if fmt <= 1 {
			bele.BEPutUint24(out[index:], uint32(header.MsgLen))
			index += 3
			out[index] = uint8(header.MsgTypeID)
			index++

			if fmt == 0 {
				bele.LEPutUint32(out[index:], uint32(header.MsgStreamID))
				index += 4
			}
		}
	}

	// 设置扩展时间戳
	if timestamp > maxTimestampInMessageHeader {
		log.Debugf("CHEFERASEME %+v %+v %d %d", header, prevHeader, timestamp, index)
		bele.BEPutUint32(out[index:], timestamp)
		index += 4
	}

	return out[0:index]
}

func message2Chunks(message []byte, header *Header, prevHeader *Header, chunkSize int) []byte {
	//if header.CSID < minCSID || header.CSID > maxCSID {
	//	return nil, rtmpErr
	//}

	// 计算chunk数量，最后一个chunk的大小
	numOfChunk := len(message) / chunkSize
	lastChunkSize := chunkSize
	if len(message)%chunkSize != 0 {
		numOfChunk++
		lastChunkSize = len(message) % chunkSize
	}

	maxNeededLen := (chunkSize + maxHeaderSize) * numOfChunk
	out := make([]byte, maxNeededLen)

	var index int

	// NOTICE 和srs交互时，发现srs要求message中的非第一个chunk不能使用fmt0
	// 将message切割成chunk放入chunk body中
	for i := 0; i < numOfChunk; i++ {
		head := calcHeader(header, prevHeader)
		copy(out[index:], head)
		index += len(head)

		if i != numOfChunk-1 {
			copy(out[index:], message[i*chunkSize:i*chunkSize+chunkSize])
			index += chunkSize
		} else {
			copy(out[index:], message[i*chunkSize:i*chunkSize+lastChunkSize])
			index += lastChunkSize
		}
		prevHeader = header
	}

	return out[:index]
}
