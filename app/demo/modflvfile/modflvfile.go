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
	"io"
	"time"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/naza/pkg/nazalog"
)

// 修改flv文件的一些信息（比如某些tag的时间戳）后另存文件
//
// Usage:
// ./bin/modflvfile -i /tmp/in.flv -o /tmp/out.flv

var countA int
var countV int
var exitFlag bool

func hookTag(tag *httpflv.Tag) {
	nazalog.Infof("%+v", tag.Header)
	if tag.Header.Timestamp != 0 {
		tag.ModTagTimestamp(tag.Header.Timestamp + uint32(time.Now().Unix()/1e6))
	}
}

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})
	defer nazalog.Sync()
	base.LogoutStartInfo()

	var err error
	inFileName, outFileName := parseFlag()

	var ffr httpflv.FlvFileReader
	err = ffr.Open(inFileName)
	nazalog.Assert(nil, err)
	defer ffr.Dispose()
	nazalog.Infof("open input flv file succ.")

	var ffw httpflv.FlvFileWriter
	err = ffw.Open(outFileName)
	nazalog.Assert(nil, err)
	defer ffw.Dispose()
	nazalog.Infof("open output flv file succ.")

	flvHeader, err := ffr.ReadFlvHeader()
	nazalog.Assert(nil, err)

	err = ffw.WriteRaw(flvHeader)
	nazalog.Assert(nil, err)

	for {
		tag, err := ffr.ReadTag()
		if err == io.EOF {
			nazalog.Infof("EOF.")
			break
		}
		nazalog.Assert(nil, err)
		hookTag(&tag)
		err = ffw.WriteRaw(tag.Raw)
		nazalog.Assert(nil, err)
	}
}

func parseFlag() (string, string) {
	i := flag.String("i", "", "specify input flv file")
	o := flag.String("o", "", "specify output flv file")
	flag.Parse()
	if *i == "" || *o == "" {
		flag.Usage()
		base.OsExitAndWaitPressIfWindows(1)
	}
	return *i, *o
}
