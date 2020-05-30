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

	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/naza/pkg/nazalog"
)

// 临时小工具，比较两个TS文件。注意，该程序还没有写完。

var filename1 = "/Volumes/Data/tmp/lal-4.ts"
var filename2 = "/Volumes/Data/tmp/nrm-4.ts"

func skipPacketFilter(tss [][]byte) (ret [][]byte) {
	for _, ts := range tss {
		h := hls.ParseTSPacketHeader(ts)
		if h.Pid == hls.PidAudio {
			continue
		}
		ret = append(ret, ts)
	}
	return
}

func parsePacket(packet []byte) {
	h := hls.ParseTSPacketHeader(packet)
	nazalog.Debugf("%+v", h)
	index := 4

	var adaptation hls.TSPacketAdaptation
	switch h.Adaptation {
	case hls.AdaptationFieldControlNo:
		// noop
	case hls.AdaptationFieldControlFollowed:
		adaptation = hls.ParseTSPacketAdaptation(packet[4:])
		index++
	default:
		nazalog.Warn(h.Adaptation)
	}
	index += int(adaptation.Length)

	if h.PayloadUnitStart == 1 && h.Pid == 256 {
		pes, length := hls.ParsePES(packet[index:])
		nazalog.Debugf("%+v, %d", pes, length)
	}
}

func main() {
	content1, err := ioutil.ReadFile(filename1)
	nazalog.Assert(nil, err)

	content2, err := ioutil.ReadFile(filename2)
	nazalog.Assert(nil, err)

	tss1 := hls.SplitFragment2TSPackets(content1)
	tss2 := hls.SplitFragment2TSPackets(content2)

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
			//break
		}
	}

}
