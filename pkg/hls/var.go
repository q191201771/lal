// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import (
	"github.com/q191201771/naza/pkg/mock"
	"github.com/q191201771/naza/pkg/nazalog"
)

var (
	PathStrategy IPathStrategy = &DefaultPathStrategy{}

	Clock = mock.NewStdClock()

	Log = nazalog.GetGlobalLogger()
)
