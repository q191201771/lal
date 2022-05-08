// Copyright 2021, Chef.  All rights reserved.
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

	"github.com/q191201771/lal/pkg/rtprtcp"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/lal/pkg/rtsp"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/nazalog"
)

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})
	base.LogoutStartInfo()

	inRtmpUrl, outRtspUrl, overTcp := parseFlag()

	pushSession := rtsp.NewPushSession(func(option *rtsp.PushSessionOption) {
		option.OverTcp = overTcp == 1
	})

	remuxer := remux.NewRtmp2RtspRemuxer(
		func(sdpCtx sdp.LogicContext) {
			// remuxer完成前期工作，生成sdp并开始push
			nazalog.Info("start push.")
			err := pushSession.Push(outRtspUrl, sdpCtx)
			nazalog.Assert(nil, err)
			nazalog.Info("push succ.")

		},
		func(pkt rtprtcp.RtpPacket) {
			_ = pushSession.WriteRtpPacket(pkt) // remuxer的数据给push发送
		},
	)

	pullSession := rtmp.NewPullSession().WithOnReadRtmpAvMsg(remuxer.FeedRtmpMsg)

	nazalog.Info("start pull.")
	err := pullSession.Pull(inRtmpUrl)
	nazalog.Assert(nil, err)
	nazalog.Info("pull succ.")

	select {
	case err := <-pullSession.WaitChan():
		nazalog.Fatalf("pull stopped. err=%+v", err)
	case err := <-pushSession.WaitChan():
		nazalog.Fatalf("push stopped. err=%+v", err)
	}
}

func parseFlag() (inRtmpUrl string, outRtspUrl string, overTcp int) {
	i := flag.String("i", "", "specify pull rtmp url")
	o := flag.String("o", "", "specify push rtsp url")
	t := flag.Int("t", 0, "specify rtsp interleaved mode(rtp/rtcp over tcp)")
	flag.Parse()
	if *i == "" || *o == "" {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  %s -i rtmp://localhost:1935/live/test110 -o rtsp://localhost:5544/live/test220 -t 0
  %s -i rtmp://localhost:1935/live/test110 -o rtsp://localhost:5544/live/test220 -t 1
`, os.Args[0], os.Args[0])
		base.OsExitAndWaitPressIfWindows(1)
	}
	return *i, *o, *t
}
