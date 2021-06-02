// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

type AVPacketPT int

const (
	AVPacketPTUnknown AVPacketPT = -1
	AVPacketPTAVC     AVPacketPT = RTPPacketTypeAVCOrHEVC
	AVPacketPTHEVC    AVPacketPT = RTPPacketTypeHEVC
	AVPacketPTAAC     AVPacketPT = RTPPacketTypeAAC
)

// 不同场景使用时，字段含义可能不同。
// 使用AVPacket的地方，应注明各字段的含义。
type AVPacket struct {
	Timestamp   uint32
	PayloadType AVPacketPT
	Payload     []byte
}

func (a AVPacketPT) ReadableString() string {
	switch a {
	case AVPacketPTUnknown:
		return "unknown"
	case AVPacketPTAVC:
		return "avc"
	case AVPacketPTHEVC:
		return "hevc"
	case AVPacketPTAAC:
		return "aac"
	}
	return ""
}
