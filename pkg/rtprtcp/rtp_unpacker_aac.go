// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtprtcp

import "github.com/cfeeling/lal/pkg/base"

// AAC格式的流，尝试合成一个完整的帧
func (r *RTPUnpacker) unpackOneAAC() bool {
	first := r.list.head.next
	if first == nil {
		return false
	}

	// TODO chef:
	// 2. 只处理了一个RTP包含多个音频包的情况，没有处理一个音频包跨越多个RTP包的情况（是否有这种情况）

	// rfc3640 2.11.  Global Structure of Payload Format
	//
	// +---------+-----------+-----------+---------------+
	// | RTP     | AU Header | Auxiliary | Access Unit   |
	// | Header  | Section   | Section   | Data Section  |
	// +---------+-----------+-----------+---------------+
	//
	//           <----------RTP Packet Payload----------->
	//
	// rfc3640 3.2.1.  The AU Header Section
	//
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+- .. -+-+-+-+-+-+-+-+-+-+
	// |AU-headers-length|AU-header|AU-header|      |AU-header|padding|
	// |                 |   (1)   |   (2)   |      |   (n)   | bits  |
	// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+- .. -+-+-+-+-+-+-+-+-+-+
	//
	// rfc3640 3.3.6.  High Bit-rate AAC
	//

	b := first.packet.Raw[first.packet.Header.payloadOffset:]
	//nazalog.Debugf("%d, %d, %s", len(pkt.Raw), pkt.Header.timestamp, hex.Dump(b))

	// AU Header Section
	var auHeaderLength uint32
	auHeaderLength = uint32(b[0])<<8 + uint32(b[1])
	auHeaderLength = (auHeaderLength + 7) / 8
	//nazalog.Debugf("auHeaderLength=%d", auHeaderLength)

	// no Auxiliary Section

	pauh := uint32(2)                 // AU Header pos
	pau := uint32(2) + auHeaderLength // AU pos
	auNum := uint32(auHeaderLength) / 2
	for i := uint32(0); i < auNum; i++ {
		var auSize uint32
		auSize = uint32(b[pauh])<<8 | uint32(b[pauh+1]&0xF8) // 13bit
		auSize /= 8

		//auIndex := b[pauh+1] & 0x7

		// raw AAC frame
		// pau, auSize
		//nazalog.Debugf("%d %d %s", auSize, auIndex, hex.Dump(b[pau:pau+auSize]))
		var outPkt base.AVPacket
		outPkt.Timestamp = first.packet.Header.Timestamp / uint32(r.clockRate/1000)
		outPkt.Timestamp += i * uint32((1024*1000)/r.clockRate)
		outPkt.Payload = b[pau : pau+auSize]
		outPkt.PayloadType = r.payloadType

		r.onAVPacket(outPkt)

		pauh += 2
		pau += auSize
	}

	r.unpackedFlag = true
	r.unpackedSeq = first.packet.Header.Seq
	r.list.head.next = first.next
	r.list.size--
	return true
}
