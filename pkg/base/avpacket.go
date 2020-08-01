// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

var (
	AVPacketPTAVC = 96
	AVPacketPTAAC = 97
)

// 目前供package rtsp使用。以后可能被多个package使用。不排除不同package使用时，字段含义也不同的情况出现
type AVPacket struct {
	Timestamp   uint32 // 绝对时间戳，单位毫秒
	Payload     []byte
	PayloadType int
}
