// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package mpegts_test

import (
	"testing"

	"github.com/cfeeling/lal/pkg/mpegts"
	"github.com/cfeeling/naza/pkg/nazalog"
)

func TestParseFixedTSPacket(t *testing.T) {
	h := mpegts.ParseTSPacketHeader(mpegts.FixedFragmentHeader)
	nazalog.Debugf("%+v", h)
	pat := mpegts.ParsePAT(mpegts.FixedFragmentHeader[5:])
	nazalog.Debugf("%+v", pat)

	h = mpegts.ParseTSPacketHeader(mpegts.FixedFragmentHeader[188:])
	nazalog.Debugf("%+v", h)
	pmt := mpegts.ParsePMT(mpegts.FixedFragmentHeader[188+5:])
	nazalog.Debugf("%+v", pmt)
}
