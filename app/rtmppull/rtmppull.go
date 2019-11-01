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

func main() {
	var (
		w   httpflv.FLVFileWriter
		err error
	)

	url, outFileName := parseFlag()

	err = w.Open(outFileName)
	log.FatalIfErrorNotNil(err)
	defer w.Dispose()
	err = w.WriteRaw(httpflv.FLVHeader)
	log.FatalIfErrorNotNil(err)

	session := rtmp.NewPullSession(func(option *rtmp.PullSessionOption) {
		option.ConnectTimeoutMS = 3000
		option.PullTimeoutMS = 3000
		option.ReadAVTimeoutMS = 10000
	})

	err = session.Pull(url, func(msg rtmp.AVMsg) {
		log.Infof("%+v, abs ts=%d", msg.Header, msg.Header.TimestampAbs)
		tag := logic.Trans.RTMPMsg2FLVTag(msg)
		err := w.WriteTag(*tag)
		log.FatalIfErrorNotNil(err)
	})
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
