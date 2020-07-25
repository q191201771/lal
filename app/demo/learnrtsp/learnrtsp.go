// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"os"

	"github.com/q191201771/lal/pkg/aac"

	"github.com/q191201771/lal/pkg/rtsp"
	"github.com/q191201771/naza/pkg/nazalog"
)

var sps = []byte{
	0x67, 0x64, 0x00, 0x20, 0xAC, 0xD9, 0x40, 0xC0, 0x29, 0xB0, 0x11, 0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x00, 0x03, 0x00, 0x32, 0x0F, 0x18, 0x31, 0x96,
}

var pps = []byte{
	0x68, 0xEB, 0xEC, 0xB2, 0x2C,
}

var asc = []byte{
	0x12, 0x10,
}

// TODO chef: 验证ES流是正常的，然后组织代码

type Obs struct {
}

func (obs *Obs) OnNewRTSPPubSession(session *rtsp.PubSession) {
	nazalog.Debugf("OnNewRTSPPubSession. %+v", session)

	avcFp, err := os.Create("/tmp/rtsp.h264")
	nazalog.Assert(nil, err)
	_, _ = avcFp.Write([]byte{0, 0, 0, 1})
	avcFp.Write(sps)
	_, _ = avcFp.Write([]byte{0, 0, 0, 1})
	avcFp.Write(pps)

	aacFp, err := os.Create("/tmp/rtsp.aac")
	nazalog.Assert(nil, err)

	session.SetOnAVPacket(func(pkt rtsp.AVPacket) {
		nazalog.Debugf("type=%d, ts=%d, len=%d", pkt.PayloadType, pkt.Timestamp, len(pkt.Payload))

		switch pkt.PayloadType {
		case rtsp.RTPPacketTypeAVC:
			_, _ = avcFp.Write([]byte{0, 0, 0, 1})
			_, _ = avcFp.Write(pkt.Payload)
			_ = avcFp.Sync()
		case rtsp.RTPPacketTypeAAC:
			var a aac.ADTS
			_ = a.InitWithAACAudioSpecificConfig(asc)
			h, _ := a.CalcADTSHeader(uint16(len(pkt.Payload)))
			_, _ = aacFp.Write(h)
			_, _ = aacFp.Write(pkt.Payload)
			_ = aacFp.Sync()
		}
	})
}

func (obs *Obs) OnDelRTSPPubSession(session *rtsp.PubSession) {
	nazalog.Debugf("OnDelRTSPPubSession. %+v", session)
}

func main() {
	var obs Obs
	s := rtsp.NewServer(":5544", &obs)
	err := s.Listen()
	nazalog.Assert(nil, err)
	err = s.RunLoop()
	nazalog.Error(err)
}
