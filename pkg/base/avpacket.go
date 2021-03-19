// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
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

// 目前供package rtsp使用。以后可能被多个package使用。
// 不排除不同package使用时，字段含义也不同的情况出现。
// 使用AVPacket的地方，应注明各字段的含义。
type AVPacket struct {
	Timestamp   uint32
	PayloadType AVPacketPT
	Payload     []byte
}
