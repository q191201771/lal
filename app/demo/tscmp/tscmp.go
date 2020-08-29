// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"

	"github.com/q191201771/lal/pkg/mpegts"

	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/naza/pkg/nazalog"
)

// 临时小工具，比较两个TS文件。注意，该程序还没有写完。

var filename1 = "/tmp/lal/hls/innertest/innertest-7.ts"
var filename2 = "/tmp/lal/hls/innertest.bak/innertest-7.ts"

func skipPacketFilter(tss [][]byte) (ret [][]byte) {
	for _, ts := range tss {
		h := mpegts.ParseTSPacketHeader(ts)
		if h.Pid == mpegts.PidAudio {
			continue
		}
		ret = append(ret, ts)
	}
	return
}

func parsePacket(packet []byte) {
	h := mpegts.ParseTSPacketHeader(packet)
	nazalog.Debugf("%+v", h)
	index := 4

	var adaptation mpegts.TSPacketAdaptation
	switch h.Adaptation {
	case mpegts.AdaptationFieldControlNo:
		// noop
	case mpegts.AdaptationFieldControlFollowed:
		adaptation = mpegts.ParseTSPacketAdaptation(packet[4:])
		index++
	default:
		nazalog.Warn(h.Adaptation)
	}
	index += int(adaptation.Length)

	if h.PayloadUnitStart == 1 && h.Pid == 256 {
		pes, length := mpegts.ParsePES(packet[index:])
		nazalog.Debugf("%+v, %d", pes, length)
	}
}

func main() {
	content1, err := ioutil.ReadFile(filename1)
	nazalog.Assert(nil, err)

	content2, err := ioutil.ReadFile(filename2)
	nazalog.Assert(nil, err)

	tss1, _ := hls.SplitFragment2TSPackets(content1)
	tss2, _ := hls.SplitFragment2TSPackets(content2)

	nazalog.Debugf("num of ts1=%d, num of ts2=%d", len(tss1), len(tss2))

	//tss1 = skipPacketFilter(tss1)
	//tss2 = skipPacketFilter(tss2)

	nazalog.Debugf("after skip. num of ts1=%d, num of ts2=%d", len(tss1), len(tss2))

	m := len(tss1)
	if m > len(tss2) {
		m = len(tss2)
	}

	for i := 0; i < m; i++ {
		if !bytes.Equal(tss1[i], tss2[i]) {
			nazalog.Debug(i)
			parsePacket(tss1[i])
			parsePacket(tss2[i])
			nazalog.Debugf("\n%s", hex.Dump(tss1[i]))
			nazalog.Debugf("\n%s", hex.Dump(tss2[i]))
		}
	}

}
