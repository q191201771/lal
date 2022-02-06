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
	"os"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/avc"

	"github.com/q191201771/lal/pkg/aac"
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/naza/pkg/nazalog"
)

// 将本地FLV文件分离成H264/AVC和AAC的ES流文件
//
// TODO chef 做HEVC的支持

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})
	defer nazalog.Sync()
	base.LogoutStartInfo()

	var err error
	flvFileName, aacFileName, avcFileName := parseFlag()

	var ffr httpflv.FlvFileReader
	err = ffr.Open(flvFileName)
	nazalog.Assert(nil, err)
	defer ffr.Dispose()
	nazalog.Infof("open flv file succ.")

	afp, err := os.Create(aacFileName)
	nazalog.Assert(nil, err)
	defer afp.Close()
	nazalog.Infof("open es aac file succ.")

	vfp, err := os.Create(avcFileName)
	nazalog.Assert(nil, err)
	defer vfp.Close()
	nazalog.Infof("open es h264 file succ.")

	var ascCtx aac.AscContext

	for {
		tag, err := ffr.ReadTag()
		if err == io.EOF {
			nazalog.Infof("EOF.")
			break
		}
		nazalog.Assert(nil, err)

		payload := tag.Payload()

		switch tag.Header.Type {
		case httpflv.TagTypeAudio:
			if payload[1] == 0 {
				err = ascCtx.Unpack(payload[2:])
				nazalog.Assert(nil, err)
			}

			d := ascCtx.PackAdtsHeader(len(payload) - 2)
			_, _ = afp.Write(d)
			_, _ = afp.Write(payload[2:])
		case httpflv.TagTypeVideo:
			_ = avc.CaptureAvcc2Annexb(vfp, payload)
		}
	}
}

func parseFlag() (string, string, string) {
	flv := flag.String("i", "", "specify flv file")
	a := flag.String("a", "", "specify es aac file")
	v := flag.String("v", "", "specify es h264 file")
	flag.Parse()
	if *flv == "" || *a == "" || *v == "" {
		flag.Usage()
		base.OsExitAndWaitPressIfWindows(1)
	}
	return *flv, *a, *v
}
