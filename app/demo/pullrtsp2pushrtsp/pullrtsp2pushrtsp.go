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

	"github.com/q191201771/lal/pkg/sdp"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/rtsp"
	"github.com/q191201771/naza/pkg/nazalog"
)

var rtpPacketChan = make(chan rtprtcp.RtpPacket, 1024)

type Observer struct {
}

func (o *Observer) OnRtpPacket(pkt rtprtcp.RtpPacket) {
	rtpPacketChan <- pkt
}

func (o *Observer) OnSdp(sdpCtx sdp.LogicContext) {
	// noop
}

func (o *Observer) OnAvPacket(pkt base.AvPacket) {
	// noop
}

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})
	defer nazalog.Sync()

	inUrl, outUrl, pullOverTcp, pushOverTcp := parseFlag()

	o := &Observer{}
	pullSession := rtsp.NewPullSession(o, func(option *rtsp.PullSessionOption) {
		option.PullTimeoutMs = 5000
		option.OverTcp = pullOverTcp != 0
	})

	err := pullSession.Pull(inUrl)
	nazalog.Assert(nil, err)
	defer pullSession.Dispose()
	rawSdp, sdpLogicCtx := pullSession.GetSdp()

	pushSession := rtsp.NewPushSession(func(option *rtsp.PushSessionOption) {
		option.PushTimeoutMs = 5000
		option.OverTcp = pushOverTcp != 0
	})

	err = pushSession.Push(outUrl, rawSdp, sdpLogicCtx)
	nazalog.Assert(nil, err)
	defer pushSession.Dispose()

	go func() {
		for {
			pullSession.UpdateStat(1)
			pullStat := pullSession.GetStat()
			pushSession.UpdateStat(1)
			pushStat := pushSession.GetStat()
			nazalog.Debugf("stat. pull=%+v, push=%+v", pullStat, pushStat)
			time.Sleep(1 * time.Second)
		}
	}()

	// 只是为了测试主动关闭session
	//go func() {
	//	time.Sleep(5 * time.Second)
	//	pullSession.Dispose()
	//}()

	for {
		select {
		case err = <-pullSession.WaitChan():
			nazalog.Infof("< pullSession.Wait(). err=%+v", err)
			time.Sleep(1 * time.Second) // 不让程序立即退出，只是为了测试session内部资源是否正常及时释放
			return
		case err = <-pushSession.WaitChan():
			nazalog.Infof("< pushSession.Wait(). err=%+v", err)
			time.Sleep(1 * time.Second)
			return
		case pkt := <-rtpPacketChan:
			pushSession.WriteRtpPacket(pkt)
		}
	}

}

func parseFlag() (inUrl string, outUrl string, pullOverTcp int, pushOverTcp int) {
	i := flag.String("i", "", "specify pull rtsp url")
	o := flag.String("o", "", "specify push rtsp url")
	t := flag.Int("t", 0, "specify pull interleaved mode(rtp/rtcp over tcp)")
	y := flag.Int("y", 0, "specify push interleaved mode(rtp/rtcp over tcp)")
	flag.Parse()
	if *i == "" || *o == "" {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  %s -i rtsp://localhost:5544/live/test110 -o rtsp://localhost:5544/live/test220
  %s -i rtsp://localhost:5544/live/test110 -o rtsp://localhost:5544/live/test220 -t 1 -y 1
`, os.Args[0], os.Args[0])
		base.OsExitAndWaitPressIfWindows(1)
	}
	return *i, *o, *t, *y
}
