// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

const (
	// 注意，AVC和HEVC都可能使用96，所以不能直接通过96判断是AVC还是HEVC
	RTPPacketTypeAVCOrHEVC = 96
	RTPPacketTypeAAC       = 97
)
