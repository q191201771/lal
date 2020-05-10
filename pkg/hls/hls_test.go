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

	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/naza/pkg/nazalog"
)

func TestParseTSPacketHeader(t *testing.T) {
	h := hls.ParseTSPacketHeader(hls.TSHeader)
	nazalog.Debugf("%+v", h)
	pat := hls.ParsePAT(hls.TSHeader[5:])
	nazalog.Debugf("%+v", pat)

	h = hls.ParseTSPacketHeader(hls.TSHeader[188:])
	nazalog.Debugf("%+v", h)
	pmt := hls.ParsePMT(hls.TSHeader[188+5:])
	nazalog.Debugf("%+v", pmt)
}

// TODO chef: 把下面这部分内容整理成 hls pull session

//var (
//	pat     PAT
//	pmt     PMT
//	streams map[uint16]*Stream
//)
//
//// 将单个 ts 文件的内容分隔为多个 ts packet
//func splitTS(content []byte) (ret [][]byte) {
//	for {
//		if len(content) < 188 {
//			if len(content) != 0 {
//				nazalog.Fatal(len(content))
//			}
//			break
//		}
//		ret = append(ret, content[0:188])
//		content = content[188:]
//	}
//	return
//}
//
//func handlePacket(packet []byte) {
//	h, err := parseTSPacketHeader(packet)
//	nazalog.Assert(nil, err)
//	nazalog.Debugf("handlePacket. h=%+v", h)
//	index := 4
//	var tsaf TSAdaptationField
//	switch h.afc {
//	case adaptationFieldControlNo:
//		// noop
//	case adaptationFieldControlFollowed:
//		tsaf, err = parseTSAdaptationField(packet[index:])
//	default:
//		nazalog.Warn(h.afc)
//	}
//
//	if tsaf.afl != 0 {
//		index += int(tsaf.afl)
//	}
//	if h.pusi == 1 {
//		index++
//	}
//
//	if h.pid == pidPAT {
//		nazalog.Info("PAT.")
//		pat, err = parsePAT(packet[index:])
//		nazalog.Assert(nil, err)
//		nazalog.Debugf("pat=%+v", pat)
//		return
//	}
//	if pat.searchPID(h.pid) {
//		nazalog.Info("PMT.")
//		pmt, err = parsePMT(packet[index:])
//		nazalog.Assert(nil, err)
//		nazalog.Debugf("pmt=%+v", pmt)
//		for _, ppe := range pmt.ppes {
//			switch ppe.st {
//			case streamTypeAAC:
//				streams[ppe.epid] = NewStream("/tmp/out.aac")
//			case streamTypeAVC:
//				streams[ppe.epid] = NewStream("/tmp/out.h264")
//			default:
//				nazalog.Warn(ppe)
//			}
//		}
//		return
//	}
//	if stream, ok := streams[h.pid]; ok {
//		if packet[index] == 0x0 && packet[index+1] == 0x0 && packet[index+2] == 0x1 {
//			nazalog.Infof("PES. %s", hex.Dump(packet[:188]))
//			pes := parsePES(packet[index:])
//			nazalog.Debugf("pes=%+v", pes)
//			nazalog.Assert(uint8(128), pes.pad1)
//			nazalog.Assert(uint8(0), pes.pad2)
//		}
//		stream.Feed(packet[index:])
//		return
//	}
//	nazalog.Warn(h)
//}
//
//func main() {
//	streams = make(map[uint16]*Stream)
//
//	rawTS, err := ioutil.ReadFile("/tmp/test0.ts")
//	nazalog.Assert(nil, err)
//	nazalog.Debugf("file size:%d", len(rawTS))
//
//	tsPackets := splitTS(rawTS)
//	nazalog.Debugf("num of packet:%d", len(tsPackets))
//
//	for _, packet := range tsPackets {
//		handlePacket(packet)
//	}
//
//	for _, stream := range streams {
//		stream.Close()
//	}
//}

