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

	"github.com/q191201771/lal/pkg/avc"

	"github.com/q191201771/lal/pkg/aac"
	"github.com/q191201771/lal/pkg/httpflv"
	log "github.com/q191201771/naza/pkg/nazalog"
)

// 将本地FLV文件分离成H264/AVC和AAC的ES流文件
//
// TODO chef 做HEVC的支持

func main() {
	var err error
	flvFileName, aacFileName, avcFileName := parseFlag()

	var ffr httpflv.FLVFileReader
	err = ffr.Open(flvFileName)
	log.Assert(nil, err)
	defer ffr.Dispose()
	log.Infof("open flv file succ.")

	afp, err := os.Create(aacFileName)
	log.Assert(nil, err)
	defer afp.Close()
	log.Infof("open es aac file succ.")

	vfp, err := os.Create(avcFileName)
	log.Assert(nil, err)
	defer vfp.Close()
	log.Infof("open es h264 file succ.")

	var adts aac.ADTS

	for {
		tag, err := ffr.ReadTag()
		if err == io.EOF {
			log.Infof("EOF.")
			break
		}
		log.Assert(nil, err)

		payload := tag.Payload()

		switch tag.Header.Type {
		case httpflv.TagTypeAudio:
			if payload[1] == 0 {
				err = adts.PutAACSequenceHeader(payload)
				log.Assert(nil, err)
				return
			}

			d, err := adts.GetADTS(uint16(len(payload)))
			log.Assert(nil, err)
			_, _ = afp.Write(d)
			_, _ = afp.Write(payload[2:])
		case httpflv.TagTypeVideo:
			_ = avc.CaptureAVC(vfp, payload)
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
		os.Exit(1)
	}
	return *flv, *a, *v
}
