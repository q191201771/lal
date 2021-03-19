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
	"time"

	"github.com/cfeeling/lal/pkg/base"
	"github.com/cfeeling/lal/pkg/httpflv"
	"github.com/cfeeling/lal/pkg/remux"
	"github.com/cfeeling/lal/pkg/rtprtcp"
	"github.com/cfeeling/lal/pkg/rtsp"
	"github.com/q191201771/naza/pkg/nazalog"
)

var fileWriter httpflv.FLVFileWriter

type Observer struct {
}

func (o *Observer) OnRTPPacket(pkt rtprtcp.RTPPacket) {
	// noop
}

func (o *Observer) OnAVConfig(asc, vps, sps, pps []byte) {
	metadata, ash, vsh, err := remux.AVConfig2FLVTag(asc, vps, sps, pps)
	nazalog.Assert(nil, err)

	err = fileWriter.WriteTag(*metadata)
	nazalog.Assert(nil, err)

	if ash != nil {
		err = fileWriter.WriteTag(*ash)
		nazalog.Assert(nil, err)
	}

	if vsh != nil {
		err = fileWriter.WriteTag(*vsh)
		nazalog.Assert(nil, err)
	}
}

func (o *Observer) OnAVPacket(pkt base.AVPacket) {
	tag, err := remux.AVPacket2FLVTag(pkt)
	nazalog.Assert(nil, err)
	err = fileWriter.WriteTag(tag)
	nazalog.Assert(nil, err)
}

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})
	defer nazalog.Sync()

	inURL, outFilename, overTCP := parseFlag()

	err := fileWriter.Open(outFilename)
	nazalog.Assert(nil, err)
	defer fileWriter.Dispose()
	err = fileWriter.WriteRaw(httpflv.FLVHeader)
	nazalog.Assert(nil, err)

	o := &Observer{}
	pullSession := rtsp.NewPullSession(o, func(option *rtsp.PullSessionOption) {
		option.PullTimeoutMS = 5000
		option.OverTCP = overTCP != 0
	})

	err = pullSession.Pull(inURL)
	nazalog.Assert(nil, err)
	defer pullSession.Dispose()

	go func() {
		for {
			pullSession.UpdateStat(1)
			nazalog.Debugf("stat. pull=%+v", pullSession.GetStat())
			time.Sleep(1 * time.Second)
		}
	}()

	err = <-pullSession.Wait()
	nazalog.Infof("< pullSession.Wait(). err=%+v", err)
}

func parseFlag() (inURL string, outFilename string, overTCP int) {
	i := flag.String("i", "", "specify pull rtsp url")
	o := flag.String("o", "", "specify ouput flv file")
	t := flag.Int("t", 0, "specify interleaved mode(rtp/rtcp over tcp)")
	flag.Parse()
	if *i == "" || *o == "" {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  %s -i rtsp://localhost:5544/live/test110 -o out.flv -t 0
  %s -i rtsp://localhost:5544/live/test110 -o out.flv -t 1
`, os.Args[0], os.Args[0])
		base.OSExitAndWaitPressIfWindows(1)
	}
	return *i, *o, *t
}
