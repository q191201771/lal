// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtprtcp_test

import (
	"testing"
	"time"

	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/naza/pkg/nazalog"
)

func TestMSWLSW2UnixNano(t *testing.T) {
	u := rtprtcp.MSWLSW2UnixNano(3805600902, 2181843386)
	nazalog.Debug(u)
	tt := time.Unix(int64(u/1e9), int64(u%1e9))
	nazalog.Debug(tt.String())
}
