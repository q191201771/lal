// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls_test

import (
	"testing"

	"github.com/q191201771/lal/pkg/innertest"

	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/naza/pkg/nazalog"
)

func TestParseFixedTSPacket(t *testing.T) {
	h := hls.ParseTSPacketHeader(hls.FixedFragmentHeader)
	nazalog.Debugf("%+v", h)
	pat := hls.ParsePAT(hls.FixedFragmentHeader[5:])
	nazalog.Debugf("%+v", pat)

	h = hls.ParseTSPacketHeader(hls.FixedFragmentHeader[188:])
	nazalog.Debugf("%+v", h)
	pmt := hls.ParsePMT(hls.FixedFragmentHeader[188+5:])
	nazalog.Debugf("%+v", pmt)
}

func TestHls(t *testing.T) {
	innertest.InnerTestEntry(t)
}
