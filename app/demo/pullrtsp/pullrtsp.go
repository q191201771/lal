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
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/lal/pkg/rtsp"
	"github.com/q191201771/naza/pkg/nazalog"
)

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})
	defer nazalog.Sync()
	base.LogoutStartInfo()

	inUrl, outFilename, overTcp := parseFlag()

	var fileWriter httpflv.FlvFileWriter
	err := fileWriter.Open(outFilename)
	nazalog.Assert(nil, err)
	defer fileWriter.Dispose()
	err = fileWriter.WriteRaw(httpflv.FlvHeader)
	nazalog.Assert(nil, err)

	remuxer := remux.NewAvPacket2RtmpRemuxer().WithOnRtmpMsg(func(msg base.RtmpMsg) {
		err = fileWriter.WriteTag(*remux.RtmpMsg2FlvTag(msg))
		nazalog.Assert(nil, err)
	})
	pullSession := rtsp.NewPullSession(remuxer, func(option *rtsp.PullSessionOption) {
		option.PullTimeoutMs = 5000
		option.OverTcp = overTcp != 0
	})

	err = pullSession.Pull(inUrl)
	nazalog.Assert(nil, err)

	go func() {
		for {
			pullSession.UpdateStat(1)
			nazalog.Debugf("stat. pull=%+v", pullSession.GetStat())
			time.Sleep(1 * time.Second)
		}
	}()

	// 临时测试一下主动关闭client session
	//go func() {
	//	time.Sleep(5 * time.Second)
	//	err := pullSession.Dispose()
	//	nazalog.Debugf("< session Dispose. err=%+v", err)
	//}()

	err = <-pullSession.WaitChan()
	nazalog.Infof("< pullSession.Wait(). err=%+v", err)
}

func parseFlag() (inUrl string, outFilename string, overTcp int) {
	i := flag.String("i", "", "specify pull rtsp url")
	o := flag.String("o", "", "specify output flv file")
	t := flag.Int("t", 0, "specify interleaved mode(rtp/rtcp over tcp)")
	flag.Parse()
	if *i == "" || *o == "" {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  %s -i rtsp://localhost:5544/live/test110 -o out.flv -t 0
  %s -i rtsp://localhost:5544/live/test110 -o out.flv -t 1
`, os.Args[0], os.Args[0])
		base.OsExitAndWaitPressIfWindows(1)
	}
	return *i, *o, *t
}
