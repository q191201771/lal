// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package gb28181

import (
	"errors"
	"github.com/q191201771/naza/pkg/nazalog"
)

// TODO(chef): [feat] http api start_rtp_pub 202207
// TODO(chef): [feat] http api stop_rtp_pub 202207
// TODO(chef): [feat] http api /api/stat/all_rtp_pub，不过这个可以用已有的all_group代替 202207
// TODO(chef): [feat] pub接入group 202207
// TODO(chef): [feat] 超时自动关闭 202207
// TODO(chef): [test] 保存rtp数据，用于回放分析 202206
// TODO(chef): [perf] 优化ps解析，内存块 202207

var (
	Log = nazalog.GetGlobalLogger()
)

// ErrGb28181 TODO(chef): [refactor] move to pkg base 202207
//
var ErrGb28181 = errors.New("lal.gb28181: fxxk")

var maxUnpackRtpListSize = 1024
