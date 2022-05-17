// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

const (
	// RtpPacketTypeAvcOrHevc 注意，一般情况下，AVC使用96，AAC使用97，HEVC使用98
	// 但是我还遇到过：
	// HEVC使用96
	// AVC使用105
	RtpPacketTypeAvcOrHevc = 96
	RtpPacketTypeAac       = 97
	RtpPacketTypeHevc      = 98
)
