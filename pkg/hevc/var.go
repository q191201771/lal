// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hevc

import "github.com/q191201771/naza/pkg/nazalog"

var Log = nazalog.GetGlobalLogger()

// StrategyTryAnnexbWhenParseVspFromSeqHeaderFailed 从seq header中解析vps/sps/pps失败时，尝试按annexb格式解析
//
// https://github.com/q191201771/lal/pull/353
var StrategyTryAnnexbWhenParseVspFromSeqHeaderFailed = true
