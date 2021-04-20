// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtprtcp

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazalog"
)

// 传入RTP包，合成帧数据，并回调返回
// 一路音频或一路视频各对应一个对象

var (
	_ IRTPUnpacker         = &RTPUnpackContainer{}
	_ IRTPUnpackContainer  = &RTPUnpackContainer{}
	_ IRTPUnpackerProtocol = &RTPUnpackerAAC{}
	_ IRTPUnpackerProtocol = &RTPUnpackerAVCHEVC{}
)

type IRTPUnpacker interface {
	IRTPUnpackContainer
}

type IRTPUnpackContainer interface {
	Feed(pkt RTPPacket)
}

type IRTPUnpackerProtocol interface {
	// 计算rtp包处于帧中的位置
	CalcPositionIfNeeded(pkt *RTPPacket)

	// 尝试合成一个完整帧
	//
	// 从当前队列的第一个包开始合成
	// 如果一个rtp包对应一个完整帧，则合成一帧
	// 如果一个rtp包对应多个完整帧，则合成多帧
	// 如果多个rtp包对应一个完整帧，则尝试合成一帧
	//
	// @return unpackedFlag 本次调用是否成功合成
	// @return unpackedSeq  如果成功合成，合成使用的最后一个seq号；如果失败，则为0
	TryUnpackOne(list *RTPPacketList) (unpackedFlag bool, unpackedSeq uint16)
}

// @param pkt: pkt.Timestamp   RTP包头中的时间戳(pts)经过clockrate换算后的时间戳，单位毫秒
//                             注意，不支持带B帧的视频流，pts和dts永远相同
//             pkt.PayloadType base.AVPacketPTXXX
//             pkt.Payload     如果是AAC，返回的是raw frame，一个AVPacket只包含一帧
//                             如果是AVC或HEVC，一个AVPacket可能包含多个NAL(受STAP-A影响)，所以NAL前包含4字节的长度信息
//                             AAC引用的是接收到的RTP包中的内存块
//                             AVC或者HEVC是新申请的内存块，回调结束后，内部不再使用该内存块
type OnAVPacket func(pkt base.AVPacket)

// 目前支持AVC，HEVC和AAC MPEG4-GENERIC/44100/2，业务方也可以自己实现IRTPUnpackerProtocol，甚至是IRTPUnpackContainer
func DefaultRTPUnpackerFactory(payloadType base.AVPacketPT, clockRate int, maxSize int, onAVPacket OnAVPacket) IRTPUnpacker {
	var protocol IRTPUnpackerProtocol
	switch payloadType {
	case base.AVPacketPTAAC:
		protocol = NewRTPUnpackerAAC(payloadType, clockRate, onAVPacket)
	case base.AVPacketPTAVC:
		fallthrough
	case base.AVPacketPTHEVC:
		protocol = NewRTPUnpackerAVCHEVC(payloadType, clockRate, onAVPacket)
	default:
		nazalog.Fatalf("payload type not support yet. payloadType=%d", payloadType)
	}
	return NewRTPUnpackContainer(maxSize, protocol)
}
