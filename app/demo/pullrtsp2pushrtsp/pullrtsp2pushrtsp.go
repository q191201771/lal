// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/rtsp"
	"github.com/q191201771/naza/pkg/nazalog"
)

var rtpPacketChan = make(chan rtprtcp.RTPPacket, 1024)

type Observer struct {
}

func (o *Observer) OnRTPPacket(pkt rtprtcp.RTPPacket) {
	rtpPacketChan <- pkt
}

func (o *Observer) OnAVConfig(asc, vps, sps, pps []byte) {
	// noop
}

func (o *Observer) OnAVPacket(pkt base.AVPacket) {
	// noop
}

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})

	inURL, outURL := parseFlag()

	o := &Observer{}
	rtspPullSession := rtsp.NewPullSession(o, func(option *rtsp.PullSessionOption) {
		option.PullTimeoutMS = 5000
		option.OverTCP = false
	})

	rtspPushSession := rtsp.NewPushSession(func(option *rtsp.PushSessionOption) {
		option.PushTimeoutMS = 5000
		option.OverTCP = false
	})

	go func() {
		time.Sleep(3 * time.Second)
		for {
			rtspPullSession.UpdateStat(1)
			rtspStat := rtspPullSession.GetStat()
			nazalog.Debugf("bitrate. rtsp pull=%dkbit/s, rtsp push=", rtspStat.Bitrate)
			time.Sleep(1 * time.Second)
		}
	}()

	err := rtspPullSession.Pull(inURL)
	nazalog.Assert(nil, err)
	rawSDP, sdpLogicCtx := rtspPullSession.GetSDP()

	err = rtspPushSession.Push(outURL, rawSDP, sdpLogicCtx)
	nazalog.Assert(nil, err)

	for {
		select {
		case err = <-rtspPullSession.Wait():
			nazalog.Infof("pull rtsp done. err=%+v", err)
			return
		case err = <-rtspPushSession.Wait():
			nazalog.Infof("push rtsp done. err=%+v", err)
			return
		case pkt := <-rtpPacketChan:
			rtspPushSession.WriteRTPPacket(pkt)
		}
	}
}

func parseFlag() (inURL string, outURL string) {
	i := flag.String("i", "", "specify pull rtsp url")
	o := flag.String("o", "", "specify push rtmp url")
	flag.Parse()
	if *i == "" || *o == "" {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  ./bin/pullrtsp2pushrtsp -i rtsp://localhost:5544/live/test110 -o rtsp://localhost:5544/live/test220
`)
		os.Exit(1)
	}
	return *i, *o
}
