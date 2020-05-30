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

	"github.com/q191201771/lal/pkg/httpflv"
	log "github.com/q191201771/naza/pkg/nazalog"
)

// 拉取HTTP-FLV的流
//
// TODO
// - 存储成flv文件
// - 拉取HTTP-FLV流进行分析参见另外一个demo：analyseflvts。 这个demo可能可以删除掉了。

func main() {
	url := parseFlag()
	session := httpflv.NewPullSession()
	err := session.Pull(url, func(tag httpflv.Tag) {
		switch tag.Header.Type {
		case httpflv.TagTypeMetadata:
			log.Info(hex.Dump(tag.Payload()))
		case httpflv.TagTypeAudio:
		case httpflv.TagTypeVideo:
		}
	})
	log.Assert(nil, err)
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
