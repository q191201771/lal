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
	"time"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/naza/pkg/bitrate"
	"github.com/q191201771/naza/pkg/nazaatomic"
	log "github.com/q191201771/naza/pkg/nazalog"
)

// TODO chef: 存储成 flv 文件

func main() {
	url := parseFlag()
	session := httpflv.NewPullSession()
	abr := bitrate.NewBitrate()
	vbr := bitrate.NewBitrate()
	var runFlag nazaatomic.Bool
	runFlag.Store(true)
	go func() {
		for runFlag.Load() {
			time.Sleep(1 * time.Second)
			log.Infof("bitrate. audio=%dkb/s, video=%dkb/s", abr.Rate(), vbr.Rate())
		}
	}()
	err := session.Pull(url, func(tag httpflv.Tag) {
		//log.Infof("onReadFLVTag. %+v %t %t", tag.Header, tag.IsAVCKeySeqHeader(), tag.IsAVCKeyNalu())
		switch tag.Header.Type {
		case httpflv.TagTypeAudio:
			abr.Add(len(tag.Raw))
		case httpflv.TagTypeVideo:
			vbr.Add(len(tag.Raw))
		}
	})
	runFlag.Store(false)
	if err != nil {
		log.Error(err)
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
