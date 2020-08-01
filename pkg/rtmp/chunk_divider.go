// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

// chunk_divider.go
// @pure
// 将message切割成chunk

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/bele"
)

type ChunkDivider struct {
	localChunkSize int
}

var defaultChunkDivider = ChunkDivider{
	localChunkSize: LocalChunkSize,
}

// @param  header 注意，内部使用TimestampAbs而非Timestamp
// @return 返回的内存块由内部申请，不依赖参数<message>内存块
func Message2Chunks(message []byte, header *base.RTMPHeader) []byte {
	return defaultChunkDivider.Message2Chunks(message, header)
}

// TODO chef: 新的 message 的第一个 chunk 始终使用 fmt0 格式，没有参考前一个 message
func (d *ChunkDivider) Message2Chunks(message []byte, header *base.RTMPHeader) []byte {
	return message2Chunks(message, header, nil, d.localChunkSize)
}

// @param 返回头的大小
func calcHeader(header *base.RTMPHeader, prevHeader *base.RTMPHeader, out []byte) int {
	var index int

	// 计算fmt和timestamp
	fmt := uint8(0)
	var timestamp uint32
	if prevHeader == nil {
		timestamp = header.TimestampAbs
	} else {
		if header.MsgStreamID == prevHeader.MsgStreamID {
			fmt++
			if header.MsgLen == prevHeader.MsgLen && header.MsgTypeID == prevHeader.MsgTypeID {
				fmt++
				if header.TimestampAbs == prevHeader.TimestampAbs {
					fmt++
				}
			}
			if header.TimestampAbs > maxTimestampInMessageHeader {
				// 将数据打包成rtmp chunk发送给vlc，时间戳超过3字节最大范围时，
				// vlc认为fmt0和fmt3两种格式，都需要携带扩展时间戳字段，并且该时间戳字段必须使用绝对时间戳。
				timestamp = header.TimestampAbs
			} else {
				timestamp = header.TimestampAbs - prevHeader.TimestampAbs
			}
		} else {
			timestamp = header.TimestampAbs
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
			bele.BEPutUint24(out[index:], header.MsgLen)
			index += 3
			out[index] = header.MsgTypeID
			index++

			if fmt == 0 {
				bele.LEPutUint32(out[index:], uint32(header.MsgStreamID))
				index += 4
			}
		}
	}

	// 设置扩展时间戳
	if timestamp > maxTimestampInMessageHeader {
		//log.Debugf("CHEFERASEME %+v %+v %d %d", header, prevHeader, timestamp, index)
		bele.BEPutUint32(out[index:], timestamp)
		index += 4
	}

	return index
}

func message2Chunks(message []byte, header *base.RTMPHeader, prevHeader *base.RTMPHeader, chunkSize int) []byte {
	//if header.CSID < minCSID || header.CSID > maxCSID {
	//	return nil, ErrRTMP
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
		headLen := calcHeader(header, prevHeader, out[index:])
		index += headLen

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
