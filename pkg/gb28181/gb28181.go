// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package gb28181

import "errors"

// TODO(chef): gb28181 package处于开发中阶段，请不使用
// TODO(chef): [opt] rtp排序 202206
// TODO(chef): [test] 保存rtp数据，用于回放分析 202206
// TODO(chef): [perf] 优化ps解析，内存块 202207

// ErrGb28181 TODO(chef): [refactor] move to pkg base 202207
//
var ErrGb28181 = errors.New("lal.gb28181: fxxk")

var maxUnpackRtpListSize = 1024
