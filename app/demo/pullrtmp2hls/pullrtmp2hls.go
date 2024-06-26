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
	"path/filepath"

	"github.com/q191201771/lal/pkg/remux"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
)

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})
	defer nazalog.Sync()
	base.LogoutStartInfo()

	url, hlsOutPath, fragmentDurationMs, fragmentNum := parseFlag()
	nazalog.Infof("parse flag succ. url=%s, hlsOutPath=%s, fragmentDurationMs=%d, fragmentNum=%d",
		url, hlsOutPath, fragmentDurationMs, fragmentNum)

	hlsMuxerConfig := hls.MuxerConfig{
		OutPath:            hlsOutPath,
		FragmentDurationMs: fragmentDurationMs,
		FragmentNum:        fragmentNum,
	}

	ctx, err := base.ParseRtmpUrl(url)
	if err != nil {
		nazalog.Fatalf("parse rtmp url failed. url=%s, err=%+v", url, err)
	}
	streamName := ctx.LastItemOfPath

	hlsMuexer := hls.NewMuxer(streamName, &hlsMuxerConfig, nil)
	hlsMuexer.Start()

	rtmp2Mpegts := remux.NewRtmp2MpegtsRemuxer(hlsMuexer)

	pullSession := rtmp.NewPullSession(func(option *rtmp.PullSessionOption) {
		option.PullTimeoutMs = 10000
		option.ReadAvTimeoutMs = 10000
	}).WithOnReadRtmpAvMsg(rtmp2Mpegts.FeedRtmpMessage)
	err = pullSession.Start(url)

	if err != nil {
		nazalog.Fatalf("pull rtmp failed. err=%+v", err)
	}
	err = <-pullSession.WaitChan()
	nazalog.Errorf("< session.Wait [%s] err=%+v", pullSession.UniqueKey(), err)
}

func parseFlag() (url string, hlsOutPath string, fragmentDurationMs int, fragmentNum int) {
	i := flag.String("i", "", "specify pull rtmp url")
	o := flag.String("o", "", "specify output hls file")
	d := flag.Int("d", 3000, "specify duration of each ts file in millisecond")
	n := flag.Int("n", 6, "specify num of ts file in live m3u8 list")
	flag.Parse()
	if *i == "" {
		flag.Usage()
		eo := filepath.FromSlash("./pullrtmp2hls/")
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  %s -i rtmp://127.0.0.1:1935/live/test110 -o %s
  %s -i rtmp://127.0.0.1:1935/live/test110 -o %s -d 5000 -n 5
`, os.Args[0], eo, os.Args[0], eo)
		base.OsExitAndWaitPressIfWindows(1)
	}
	return *i, *o, *d, *n
}
