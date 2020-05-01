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
	"github.com/q191201771/lal/pkg/hevc"
	"github.com/q191201771/naza/pkg/bele"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/naza/pkg/bitrate"
	"github.com/q191201771/naza/pkg/nazaatomic"
	log "github.com/q191201771/naza/pkg/nazalog"
)

// TODO chef: 存储成 flv 文件

func main() {
	url := parseFlag()
	session := httpflv.NewPullSession()
	abr := bitrate.New()
	vbr := bitrate.New()
	var runFlag nazaatomic.Bool
	runFlag.Store(true)
	go func() {
		for runFlag.Load() {
			time.Sleep(1 * time.Second)
		}
	}()
	err := session.Pull(url, func(tag httpflv.Tag) {
		switch tag.Header.Type {
		case httpflv.TagTypeMetadata:
			log.Info(hex.Dump(tag.Payload()))
		case httpflv.TagTypeAudio:
			abr.Add(len(tag.Raw))
		case httpflv.TagTypeVideo:
			log.Infof("onReadFLVTag. %+v, isSeqHeader=%t, isKeyNalu=%t", tag.Header, tag.IsVideoKeySeqHeader(), tag.IsVideoKeyNalu())
			analysisVideoTag(tag)
			vbr.Add(len(tag.Raw))
		}
	})
	runFlag.Store(false)
	log.Assert(nil, err)
}

const (
	typeUnknown uint8 = 1
	typeAVC     uint8 = 2
	typeHEVC    uint8 = 3
)

var t uint8 = typeUnknown

func analysisVideoTag(tag httpflv.Tag) {
	if tag.IsVideoKeySeqHeader() {
		if tag.IsAVCKeySeqHeader() {
			t = typeAVC
			log.Info("AVC SH")
		} else if tag.IsHEVCKeySeqHeader() {
			t = typeHEVC
			log.Info("HEVC SH")
		}
	} else {
		body := tag.Raw[11:]

		for i := 5; i != int(tag.Header.DataSize); {
			naluLen := bele.BEUint32(body[i:])
			switch t {
			case typeAVC:
				log.Infof("%s %s", avc.CalcNaluTypeReadable(body[i+4:]), hex.Dump(body[i+4:i+8]))
			case typeHEVC:
				log.Infof("%s %s", hevc.CalcNaluTypeReadable(body[i+4:]), hex.Dump(body[i+4:i+8]))
			}
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
