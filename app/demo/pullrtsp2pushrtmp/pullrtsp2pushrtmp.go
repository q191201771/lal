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

	"github.com/q191201771/lal/pkg/rtmp"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/rtsp"
	"github.com/q191201771/naza/pkg/nazalog"
)

var rtmpPushSession *rtmp.PushSession

type Observer struct {
}

func (o *Observer) OnRTPPacket(pkt rtprtcp.RTPPacket) {
	// noop
}

func (o *Observer) OnAVConfig(asc, vps, sps, pps []byte) {
	metadata, ash, vsh, err := remux.AVConfig2RTMPMsg(asc, vps, sps, pps)
	nazalog.Assert(nil, err)
	err = rtmpPushSession.AsyncWrite(rtmp.Message2Chunks(metadata.Payload, &metadata.Header))
	nazalog.Assert(nil, err)
	if ash != nil {
		err = rtmpPushSession.AsyncWrite(rtmp.Message2Chunks(ash.Payload, &ash.Header))
		nazalog.Assert(nil, err)
	}
	if vsh != nil {
		err = rtmpPushSession.AsyncWrite(rtmp.Message2Chunks(vsh.Payload, &vsh.Header))
		nazalog.Assert(nil, err)
	}
}

func (o *Observer) OnAVPacket(pkt base.AVPacket) {
	msg, err := remux.AVPacket2RTMPMsg(pkt)
	nazalog.Assert(nil, err)
	err = rtmpPushSession.AsyncWrite(rtmp.Message2Chunks(msg.Payload, &msg.Header))
	nazalog.Assert(nil, err)
}

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})

	inURL, outURL, overTCP := parseFlag()

	rtmpPushSession = rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
		option.PushTimeoutMS = 5000
		option.WriteAVTimeoutMS = 5000
	})

	err := rtmpPushSession.Push(outURL)
	nazalog.Assert(nil, err)

	go func() {
		err := <-rtmpPushSession.Wait()
		nazalog.Infof("push rtmp done. err=%+v", err)
		os.Exit(1)
	}()

	o := &Observer{}
	rtspPullSession := rtsp.NewPullSession(o, func(option *rtsp.PullSessionOption) {
		option.PullTimeoutMS = 5000
		option.OverTCP = overTCP != 0
	})

	go func() {
		time.Sleep(3 * time.Second)
		for {
			rtspPullSession.UpdateStat(1)
			rtspStat := rtspPullSession.GetStat()
			rtmpPushSession.UpdateStat(1)
			rtmpStat := rtmpPushSession.GetStat()
			nazalog.Debugf("bitrate. rtsp pull=%dkbit/s, rtmp push=%dkbit/s", rtspStat.Bitrate, rtmpStat.Bitrate)
			time.Sleep(1 * time.Second)
		}
	}()

	err = rtspPullSession.Pull(inURL)
	nazalog.Assert(nil, err)
	err = <-rtspPullSession.Wait()
	nazalog.Infof("pull rtsp done. err=%+v", err)
}

func parseFlag() (inURL string, outFilename string, overTCP int) {
	i := flag.String("i", "", "specify pull rtsp url")
	o := flag.String("o", "", "specify push rtmp url")
	t := flag.Int("t", 0, "specify interleaved mode(rtp/rtcp over tcp)")
	flag.Parse()
	if *i == "" || *o == "" {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  ./bin/pullrtsp -i rtsp://localhost:5544/live/test110 -o rtmp://localhost:19350/live/test220 -t 0
  ./bin/pullrtsp -i rtsp://localhost:5544/live/test110 -o rtmp://localhost:19350/live/test220 -t 1
`)
		os.Exit(1)
	}
	return *i, *o, *t
}
