// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"flag"
	"os"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/lal/pkg/rtmp"
	log "github.com/q191201771/naza/pkg/nazalog"
)

type Obs struct {
	w httpflv.FlvFileWriter
}

func (obs *Obs) ReadRTMPAVMsgCB(header rtmp.Header, timestampAbs uint32, message []byte) {
	log.Infof("%+v, abs ts=%d", header, timestampAbs)
	tag := logic.Trans.RTMPMsg2FlvTag(header, timestampAbs, message)
	err := obs.w.WriteTag(tag)
	log.FatalIfErrorNotNil(err)
}

func main() {
	url, outFileName := parseFlag()
	var obs Obs
	session := rtmp.NewPullSession(&obs, rtmp.PullSessionTimeout{
		ConnectTimeoutMS: 3000,
		PullTimeoutMS:    5000,
		ReadAVTimeoutMS:  10000,
	})
	err := session.Pull(url)
	log.FatalIfErrorNotNil(err)

	err = obs.w.Open(outFileName)
	log.FatalIfErrorNotNil(err)
	//defer obs.w.Dispose()
	err = obs.w.WriteRaw(httpflv.FlvHeader)
	log.FatalIfErrorNotNil(err)

	err = session.WaitLoop()
	log.FatalIfErrorNotNil(err)
}

func parseFlag() (string, string) {
	i := flag.String("i", "", "specify pull rtmp url")
	o := flag.String("o", "", "specify ouput flv file")
	flag.Parse()
	if *i == "" || *o == "" {
		flag.Usage()
		os.Exit(1)
	}
	return *i, *o
}
