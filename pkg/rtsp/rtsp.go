// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

// 注意，正在学习以及实现rtsp，请不要使用这个package

// rfc2326

const (
	MethodOptions  = "OPTIONS"
	MethodDescribe = "DESCRIBE"
	MethodSetup    = "SETUP"
	MethodPlay     = "PLAY"

	MethodAnnounce = "ANNOUNCE"
)

const (
	HeaderFieldCSeq      = "CSeq"
	HeaderFieldTransport = "Transport"
)

// TODO chef:
// 收集lal中其他可以hack服务名的地方，统一到一处，并增加版本号信息
const serverName = "lalserver"
