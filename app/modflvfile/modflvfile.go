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

	"github.com/q191201771/lal/pkg/httpflv"
	log "github.com/q191201771/naza/pkg/nazalog"
)

// 修改flv文件的一些信息（比如某些tag的时间戳）后另存文件
//
// Usage:
// ./bin/modflvfile -i /tmp/in.flv -o /tmp/out.flv

var countA int
var countV int
var exitFlag bool

func hookTag(tag *httpflv.Tag) {
	log.Infof("%+v", tag.Header)
	if tag.Header.Type == httpflv.TagTypeAudio {

		//if countA < 3 {
		//	httpflv.ModTagTimestamp(tag, 16777205)
		//}
		//countA++
		if tag.IsAACSeqHeader() {
			log.Info("aac header.")
		}
	}
	if tag.Header.Type == httpflv.TagTypeVideo {
		//if countV < 3 {
		//	httpflv.ModTagTimestamp(tag, 16777205)
		//}
		//countV++
		if tag.IsAVCKeySeqHeader() {
			log.Info("key seq header.")
		}
		if tag.IsAVCKeyNalu() {
			log.Info("key nalu.")
		}
	}
}

func main() {
	var err error
	inFileName, outFileName := parseFlag()

	var ffr httpflv.FLVFileReader
	err = ffr.Open(inFileName)
	log.FatalIfErrorNotNil(err)
	defer ffr.Dispose()
	log.Infof("open input flv file succ.")

	var ffw httpflv.FLVFileWriter
	err = ffw.Open(outFileName)
	log.FatalIfErrorNotNil(err)
	defer ffw.Dispose()
	log.Infof("open output flv file succ.")

	flvHeader, err := ffr.ReadFLVHeader()
	log.FatalIfErrorNotNil(err)

	err = ffw.WriteRaw(flvHeader)
	log.FatalIfErrorNotNil(err)

	//for i:=0; i < 10; i++{
	for {
		tag, err := ffr.ReadTag()
		if err == io.EOF {
			log.Infof("EOF.")
			break
		}
		log.FatalIfErrorNotNil(err)
		if tag.Header.Type == httpflv.TagTypeVideo && tag.Header.DataSize == 68 && tag.Header.Timestamp == 677764 {
			break
		}

		//log.Infof("> hook. %+v", tag)
		hookTag(&tag)
		//log.Infof("< hook. %+v", tag)
		err = ffw.WriteRaw(tag.Raw)
		log.FatalIfErrorNotNil(err)
	}
}

func parseFlag() (string, string) {
	i := flag.String("i", "", "specify input flv file")
	o := flag.String("o", "", "specify ouput flv file")
	flag.Parse()
	if *i == "" || *o == "" {
		flag.Usage()
		os.Exit(1)
	}
	return *i, *o
}
