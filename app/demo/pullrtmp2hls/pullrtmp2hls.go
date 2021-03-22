// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
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
	"strings"

	"github.com/cfeeling/lal/pkg/base"
	"github.com/cfeeling/lal/pkg/hls"
	"github.com/cfeeling/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
)

func main() {
	url, hlsOutPath, fragmentDurationMS, fragmentNum := parseFlag()

	hlsMuxerConfig := hls.MuxerConfig{
		Enable:             true,
		OutPath:            hlsOutPath,
		FragmentDurationMS: fragmentDurationMS,
		FragmentNum:        fragmentNum,
	}

	index := strings.LastIndexByte(url, '/')
	if index == -1 {
		nazalog.Error("rtmp url invalid.")
		os.Exit(1)
	}
	streamName := url[index:]

	pullSession := rtmp.NewPullSession()
	hlsMuexer := hls.NewMuxer(streamName, &hlsMuxerConfig, nil, nil)
	hlsMuexer.Start()

	err := pullSession.Pull(url, func(msg base.RTMPMsg) {
		hlsMuexer.FeedRTMPMessage(msg)
	})
	if err != nil {
		nazalog.Errorf("pull error. err=%+v", err)
		os.Exit(1)
	}
	err = <-pullSession.Wait()
	nazalog.Errorf("pull error. err=%+v", err)
}

func parseFlag() (url string, hlsOutPath string, fragmentDurationMS int, fragmentNum int) {
	i := flag.String("i", "", "specify pull rtmp url")
	o := flag.String("o", "", "specify ouput hls file")
	d := flag.Int("d", 3000, "specify duration of each ts file in millisecond")
	n := flag.Int("n", 6, "specify num of ts file in live m3u8 list")
	flag.Parse()
	if *i == "" {
		flag.Usage()
		eo := filepath.FromSlash("./pullrtmp2hls/")
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  %s -i rtmp://127.0.0.1:19350/live/test110 -o %s
  %s -i rtmp://127.0.0.1:19350/live/test110 -o %s -d 5000 -n 5
`, os.Args[0], eo, os.Args[0], eo)
		base.OSExitAndWaitPressIfWindows(1)
	}
	return *i, *o, *d, *n
}
