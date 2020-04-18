// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"encoding/hex"
	"flag"
	"os"
	"time"

	"github.com/q191201771/lal/pkg/avc"

	"github.com/q191201771/naza/pkg/bele"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/naza/pkg/bitrate"
	"github.com/q191201771/naza/pkg/nazaatomic"
	"github.com/q191201771/naza/pkg/nazalog"
)

// TODO chef: 存储成 flv 文件

func main() {
	url := parseFlag()
	session := httpflv.NewPullSession()
	abr := bitrate.New()
	vbr := bitrate.New()
	prevTs := int64(-1)
	var runFlag nazaatomic.Bool
	runFlag.Store(true)
	go func() {
		for runFlag.Load() {
			time.Sleep(1 * time.Second)
			//log.Infof("bitrate. audio=%fkb/s, video=%fkb/s", abr.Rate(), vbr.Rate())
		}
	}()
	err := session.Pull(url, func(tag httpflv.Tag) {
		now := time.Now().UnixNano() / 1e6
		if prevTs != -1 {
			//log.Infof("%v", now - prevTs)
		}
		prevTs = now

		switch tag.Header.Type {
		case httpflv.TagTypeMetadata:
			//nazalog.Infof("onReadFLVTag. %+v", tag.Header)
			nazalog.Info(hex.Dump(tag.Payload()))
		case httpflv.TagTypeAudio:
			//nazalog.Infof("onReadFLVTag. %+v, isSeqHeader=%t", tag.Header, tag.IsAACSeqHeader())
			abr.Add(len(tag.Raw))
		case httpflv.TagTypeVideo:
			nazalog.Infof("onReadFLVTag. %+v, isSeqHeader=%t, isKeyNalu=%t", tag.Header, tag.IsVideoKeySeqHeader(), tag.IsVideoKeyNalu())
			analysisVideoTag(tag)
			vbr.Add(len(tag.Raw))
		}
	})
	runFlag.Store(false)
	nazalog.FatalIfErrorNotNil(err)
}

func analysisVideoTag(tag httpflv.Tag) {
	body := tag.Raw[11:]
	if body[1] == httpflv.AVCPacketTypeSeqHeader {
		nazalog.Infof("SH")
	} else {
		for i := 5; i != int(tag.Header.DataSize); {
			naluLen := bele.BEUint32(body[i:])
			naluType := body[i+4] & 0x1f
			if naluType == avc.NaluUnitTypeSEI {
				nazalog.Info("SEI")
			} else {
				nazalog.Infof("len=%d, nalu type=%s, slice type=%s", naluLen, avc.NaluUintTypeMapping[naluType], avc.CalcSliceTypeReadable(body[i+4:]))
			}
			//nazalog.Info(hex.Dump(body[i+4 : i+20]))
			i = i + 4 + int(naluLen)
		}
	}

}

func parseFlag() string {
	url := flag.String("i", "", "specify http-flv url")
	flag.Parse()
	if *url == "" {
		flag.Usage()
		os.Exit(1)
	}
	return *url
}
