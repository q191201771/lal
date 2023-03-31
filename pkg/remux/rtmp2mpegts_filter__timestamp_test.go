// Copyright 2023, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package remux

import (
	"github.com/q191201771/lal/pkg/mpegts"
	"github.com/q191201771/naza/pkg/assert"
	"testing"
)

func TestRtmp2MpegtsTimestampFilter_Do(t *testing.T) {

	var f Rtmp2MpegtsTimestampFilter
	f.Init("test")
	in := []*mpegts.Frame{
		{Sid: mpegts.StreamIdVideo, Dts: 0, Cts: 80},
		{Sid: mpegts.StreamIdVideo, Dts: 3600, Cts: 80},
		{Sid: mpegts.StreamIdVideo, Dts: 7200, Cts: 200},
		{Sid: mpegts.StreamIdVideo, Dts: 10800, Cts: 80},
		{Sid: mpegts.StreamIdVideo, Dts: 14400, Cts: 0},
		{Sid: mpegts.StreamIdAudio, Dts: 3060, Cts: 0},
		{Sid: mpegts.StreamIdVideo, Dts: 18000, Cts: 40},
		{Sid: mpegts.StreamIdVideo, Dts: 21600, Cts: 200},
		{Sid: mpegts.StreamIdVideo, Dts: 25200, Cts: 80},
		{Sid: mpegts.StreamIdVideo, Dts: 28800, Cts: 0},
		{Sid: mpegts.StreamIdAudio, Dts: 17730, Cts: 0},
		{Sid: mpegts.StreamIdVideo, Dts: 32400, Cts: 40},
	}
	expected := []*mpegts.Frame{
		{Sid: mpegts.StreamIdVideo, Dts: 0, Cts: 80, Pts: 7200},
		{Sid: mpegts.StreamIdVideo, Dts: 3600, Cts: 80, Pts: 10800},
		{Sid: mpegts.StreamIdVideo, Dts: 7200, Cts: 200, Pts: 25200},
		{Sid: mpegts.StreamIdVideo, Dts: 10800, Cts: 80, Pts: 18000},
		{Sid: mpegts.StreamIdVideo, Dts: 14400, Cts: 0, Pts: 14400},
		{Sid: mpegts.StreamIdAudio, Dts: 0, Cts: 0, Pts: 0},
		{Sid: mpegts.StreamIdVideo, Dts: 18000, Cts: 40, Pts: 21600},
		{Sid: mpegts.StreamIdVideo, Dts: 21600, Cts: 200, Pts: 39600},
		{Sid: mpegts.StreamIdVideo, Dts: 25200, Cts: 80, Pts: 32400},
		{Sid: mpegts.StreamIdVideo, Dts: 28800, Cts: 0, Pts: 28800},
		{Sid: mpegts.StreamIdAudio, Dts: 14670, Cts: 0, Pts: 14670},
		{Sid: mpegts.StreamIdVideo, Dts: 32400, Cts: 0, Pts: 36000},
	}

	for i := range in {
		//nazalog.Debugf("%d", i)
		f.Do(in[i])
		assert.Equal(t, expected[i].Dts, in[i].Dts)
		assert.Equal(t, expected[i].Pts, in[i].Pts)
	}
}
