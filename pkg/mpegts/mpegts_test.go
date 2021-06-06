// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package mpegts_test

import (
	"testing"

	"github.com/q191201771/lal/pkg/mpegts"
	"github.com/q191201771/naza/pkg/nazalog"
)

func TestParseFixedTsPacket(t *testing.T) {
	h := mpegts.ParseTsPacketHeader(mpegts.FixedFragmentHeader)
	nazalog.Debugf("%+v", h)
	pat := mpegts.ParsePat(mpegts.FixedFragmentHeader[5:])
	nazalog.Debugf("%+v", pat)

	h = mpegts.ParseTsPacketHeader(mpegts.FixedFragmentHeaderHevc[188:])
	nazalog.Debugf("%+v", h)
	pmt := mpegts.ParsePmt(mpegts.FixedFragmentHeader[188+5:])
	nazalog.Debugf("%+v", pmt)
}
