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
	log "github.com/q191201771/naza/pkg/nazalog"
)

// TODO chef: 存储成 flv 文件

func main() {
	url := parseFlag()
	session := httpflv.NewPullSession()
	err := session.Pull(url, func(tag httpflv.Tag) {
		log.Infof("onReadFLVTag. %+v %t %t", tag.Header, tag.IsAVCKeySeqHeader(), tag.IsAVCKeyNalu())
	})
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
