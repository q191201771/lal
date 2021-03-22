// Copyright 2019, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
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
	"sync/atomic"
	"time"

	"github.com/cfeeling/lal/pkg/remux"

	"github.com/cfeeling/lal/pkg/base"

	"github.com/cfeeling/lal/pkg/httpflv"
	"github.com/cfeeling/lal/pkg/rtmp"
	"github.com/cfeeling/naza/pkg/nazalog"
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
//   ./bin/pullrtmp -i rtmp://127.0.0.1:1935/live/test -o out.flv
//   ./bin/pullrtmp -i rtmp://127.0.0.1:1935/live/test -n 1000
//   ./bin/pullrtmp -i rtmp://127.0.0.1:1935/live/test_{i} -n 1000

var aliveSessionCount int32

func main() {
	urlTmpl, fileNameTmpl, num := parseFlag()
	urls, filenames := collect(urlTmpl, fileNameTmpl, num)

	go func() {
		for {
			nazalog.Debugf("alive session:%d", atomic.LoadInt32(&aliveSessionCount))
			time.Sleep(1 * time.Second)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(num)
	for i := 0; i < num; i++ {
		go func(index int) {
			pull(urls[index], filenames[index])
			wg.Done()
			atomic.AddInt32(&aliveSessionCount, -1)
		}(i)
	}
	wg.Wait()
	time.Sleep(1 * time.Second)
	nazalog.Info("bye.")
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
		option.PullTimeoutMS = 30000
		option.ReadAVTimeoutMS = 10000
	})

	err = session.Pull(
		url,
		func(msg base.RTMPMsg) {
			//nazalog.Debugf("header=%+v", msg.Header)
			if filename != "" {
				tag := remux.RTMPMsg2FLVTag(msg)
				err := w.WriteTag(*tag)
				nazalog.Assert(nil, err)
			}
		})
	if err != nil {
		nazalog.Errorf("pull failed. err=%v", err)
		return
	}
	atomic.AddInt32(&aliveSessionCount, 1)

	err = <-session.Wait()
	nazalog.Debug(err)
}

func collect(urlTmpl string, fileNameTmpl string, num int) (urls []string, filenames []string) {
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
	n := flag.Int("n", 1, "specify num of pull connection")
	flag.Parse()
	if *i == "" {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  %s -i rtmp://127.0.0.1:1935/live/test -o out.flv
  %s -i rtmp://127.0.0.1:1935/live/test -n 1000
  %s -i rtmp://127.0.0.1:1935/live/test_{i} -n 1000
`, os.Args[0], os.Args[0], os.Args[0])
		base.OSExitAndWaitPressIfWindows(1)
	}
	return *i, *o, *n
}
