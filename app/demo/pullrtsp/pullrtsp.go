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

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/rtsp"
	"github.com/q191201771/naza/pkg/nazalog"
)

var w httpflv.FLVFileWriter

type Observer struct {
}

func (o *Observer) OnRTPPacket(pkt rtprtcp.RTPPacket) {
	// noop
}

func (o *Observer) OnAVConfig(asc, vps, sps, pps []byte) {
	metadata, vsh, ash, err := remux.AVConfig2FLVTag(asc, vps, sps, pps)
	nazalog.Assert(nil, err)
	err = w.WriteTag(*metadata)
	nazalog.Assert(nil, err)
	err = w.WriteTag(*vsh)
	nazalog.Assert(nil, err)
	err = w.WriteTag(*ash)
	nazalog.Assert(nil, err)
}

func (o *Observer) OnAVPacket(pkt base.AVPacket) {
	tag, err := remux.AVPacket2FLVTag(pkt)
	nazalog.Assert(nil, err)
	err = w.WriteTag(tag)
	nazalog.Assert(nil, err)
}

func main() {
	inURL, outFilename := parseFlag()
	err := w.Open(outFilename)
	nazalog.Assert(nil, err)
	defer w.Dispose()
	err = w.WriteRaw(httpflv.FLVHeader)
	nazalog.Assert(nil, err)

	o := &Observer{}
	s := rtsp.NewPullSession(o, func(option *rtsp.PullSessionOption) {
		option.PullTimeoutMS = 5000
	})
	err = s.Pull(inURL)
	nazalog.Error(err)
}

func parseFlag() (inURL string, outFilename string) {
	i := flag.String("i", "", "specify pull rtsp url")
	o := flag.String("o", "", "specify ouput flv file")
	flag.Parse()
	if *i == "" || *o == "" {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  ./bin/pullrtsp -i rtsp://localhost:5544/live/test110 -o out.flv
`)
		os.Exit(1)
	}
	return *i, *o
}
