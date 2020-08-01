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
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
)

// RTMP拉流客户端，从远端服务器拉取RTMP流，存储为本地FLV文件
//
// 另外，作为一个RTMP拉流压测工具，已经支持：
// 1. 对一路流拉取n份
// 2. 拉取n路流
//
// Usage of ./bin/pullrtmp:
//   -i string
//     	specify pull rtmp url
//   -n int
//     	num of pull connection (default 1)
//   -o string
//     	specify ouput flv file
// Example:
//   ./bin/pullrtmp -i rtmp://127.0.0.1:19350/live/test -o out.flv
//   ./bin/pullrtmp -i rtmp://127.0.0.1:19350/live/test -n 1000
//   ./bin/pullrtmp -i rtmp://127.0.0.1:19350/live/test_{i} -n 1000

func main() {
	urlTmpl, fileNameTmpl, num := parseFlag()
	urls, filenames := connect(urlTmpl, fileNameTmpl, num)

	var wg sync.WaitGroup
	wg.Add(num)
	for i := 0; i < num; i++ {
		go func(index int) {
			pull(urls[index], filenames[index])
			wg.Done()
		}(i)
	}
	wg.Wait()
}

func pull(url string, filename string) {
	var (
		w   httpflv.FLVFileWriter
		err error
	)

	if filename != "" {
		err = w.Open(filename)
		nazalog.Assert(nil, err)
		defer w.Dispose()
		err = w.WriteRaw(httpflv.FLVHeader)
		nazalog.Assert(nil, err)
	}

	session := rtmp.NewPullSession(func(option *rtmp.PullSessionOption) {
		option.ConnectTimeoutMS = 3000
		option.PullTimeoutMS = 3000
		option.ReadAVTimeoutMS = 10000
	})

	err = session.Pull(
		url,
		func(msg base.RTMPMsg) {
			nazalog.Debugf("header=%+v", msg.Header)
			if filename != "" {
				tag := logic.Trans.RTMPMsg2FLVTag(msg)
				err := w.WriteTag(*tag)
				nazalog.Assert(nil, err)
			}
		})
	nazalog.Assert(nil, err)
	err = <-session.Done()
	nazalog.Debug(err)
}

func connect(urlTmpl string, fileNameTmpl string, num int) (urls []string, filenames []string) {
	for i := 0; i < num; i++ {
		url := strings.Replace(urlTmpl, "{i}", strconv.Itoa(i), -1)
		urls = append(urls, url)
		filename := strings.Replace(fileNameTmpl, "{i}", strconv.Itoa(i), -1)
		filenames = append(filenames, filename)
	}
	return
}

func parseFlag() (urlTmpl string, fileNameTmpl string, num int) {
	i := flag.String("i", "", "specify pull rtmp url")
	o := flag.String("o", "", "specify ouput flv file")
	n := flag.Int("n", 1, "num of pull connection")
	flag.Parse()
	if *i == "" {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  ./bin/pullrtmp -i rtmp://127.0.0.1:19350/live/test -o out.flv
  ./bin/pullrtmp -i rtmp://127.0.0.1:19350/live/test -n 1000
  ./bin/pullrtmp -i rtmp://127.0.0.1:19350/live/test_{i} -n 1000
`)
		os.Exit(1)
	}
	return *i, *o, *n
}
