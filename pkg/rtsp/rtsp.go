// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

// 注意，正在学习以及实现rtsp，请不要使用这个package

// TODO chef
// - rtp和rtcp作为独立package

// rfc2326

const (
	MethodOptions  = "OPTIONS"
	MethodAnnounce = "ANNOUNCE"
	MethodSetup    = "SETUP"
	MethodRecord   = "RECORD"
	MethodDescribe = "DESCRIBE"
	MethodPlay     = "PLAY"
)

const (
	HeaderFieldCSeq      = "CSeq"
	HeaderFieldTransport = "Transport"
)

var (
	// TODO chef:
	// 收集lal中其他可以hack服务名的地方，统一到一处，并增加版本号信息
	serverName = "lalserver"
	sessionID  = "191201771"

	minServerPort = uint16(8000)
	maxServerPort = uint16(16000)

	udpMaxPacketLength = 1500
)
