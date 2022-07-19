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
	"sync/atomic"
	"time"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/remux"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
)

// RTMP推流客户端，读取本地FLV文件，使用RTMP协议推送出去
//
// 支持匀速推送：按照时间戳的间隔时间推送
// 支持循环推送：文件推送完毕后，可循环推送（RTMP push 流并不断开）
// 支持推送多路流：相当于一个RTMP推流压测工具
//
// 连接断开，内部并不做重试。当所有推流连接断开时，程序退出。
//
// Usage of ./bin/pushrtmp:
// -i string
// specify flv file
// -n int
// num of push connection (default 1)
// -o string
// specify rtmp push url
// -r	recursive push if reach end of file
// -v	show bin info
// Example:
// ./bin/pushrtmp -i testdata/test.flv -o rtmp://127.0.0.1:19350/live/test
// ./bin/pushrtmp -i testdata/test.flv -o rtmp://127.0.0.1:19350/live/test -r
// ./bin/pushrtmp -i testdata/test.flv -o rtmp://127.0.0.1:19350/live/test_{i} -r -n 1000

var aliveSessionCount int32

func main() {
	defer nazalog.Sync()
	base.LogoutStartInfo()

	filename, urlTmpl, num, isRecursive, logfile := parseFlag()
	initLog(logfile)

	urls := collect(urlTmpl, num)

	tags, err := httpflv.ReadAllTagsFromFlvFile(filename)
	if err != nil || len(tags) == 0 {
		nazalog.Fatalf("read tags from flv file failed. len=%d, err=%+v", len(tags), err)
		return
	}
	nazalog.Infof("read tags from flv file succ. len of tags=%d", len(tags))

	go func() {
		for {
			nazalog.Debugf("alive session:%d", atomic.LoadInt32(&aliveSessionCount))
			time.Sleep(1 * time.Second)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(len(urls))
	for _, url := range urls {
		go func(u string) {
			push(tags, u, isRecursive)
			wg.Done()
			atomic.AddInt32(&aliveSessionCount, -1)
		}(url)
	}
	wg.Wait()
	time.Sleep(1 * time.Second)
	nazalog.Info("< main.")
}

func push(tags []httpflv.Tag, url string, isRecursive bool) {
	ps := rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
		option.PushTimeoutMs = 5000
		option.WriteAvTimeoutMs = 10000
		option.WriteBufSize = 0
		option.WriteChanSize = 0
	})

	if err := ps.Push(url); err != nil {
		nazalog.Errorf("push failed. err=%v", err)
		return
	}
	atomic.AddInt32(&aliveSessionCount, 1)

	nazalog.Infof("push succ. url=%s", url)

	go func() {
		flvFilePump := httpflv.NewFlvFilePump(func(option *httpflv.FlvFilePumpOption) {
			option.IsRecursive = isRecursive
		})
		_ = flvFilePump.PumpWithTags(tags, func(tag httpflv.Tag) bool {

			if tag.Header.Type == base.RtmpTypeIdMetadata {
				m, err := rtmp.ParseMetadata(tag.Payload())
				nazalog.Debugf("metadata. err=%+v, len=%d, value=%s", err, len(m), m.DebugString())
			}

			chunks := remux.FlvTag2RtmpChunks(tag)

			if err := ps.Write(chunks); err != nil {
				nazalog.Errorf("write data error. err=%v", err)
				return false
			}
			return true
		})
	}()

	// 临时测试一下主动关闭client session
	//go func() {
	//	time.Sleep(5 * time.Second)
	//	nazalog.Debugf("> session Dispose.")
	//	err := ps.Dispose()
	//	nazalog.Debugf("< session Dispose. err=%+v", err)
	//}()

	err := <-ps.WaitChan()
	nazalog.Infof("< session WaitChan. err=%+v", err)
}

func collect(urlTmpl string, num int) (urls []string) {
	for i := 0; i < num; i++ {
		url := strings.Replace(urlTmpl, "{i}", strconv.Itoa(i), -1)
		urls = append(urls, url)
	}
	return
}

func initLog(logfile string) {
	if logfile != "" {
		err := nazalog.Init(func(option *nazalog.Option) {
			option.IsRotateDaily = false
			option.Filename = logfile
			option.IsToStdout = false
		})
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "init nazalog failed. err=%+v", err)
			os.Exit(1)
		}
	} else {
		_ = nazalog.Init(func(option *nazalog.Option) {
			option.AssertBehavior = nazalog.AssertFatal
		})
	}
}

func parseFlag() (filename string, urlTmpl string, num int, isRecursive bool, logfile string) {
	i := flag.String("i", "", "specify flv file")
	o := flag.String("o", "", "specify rtmp push url")
	r := flag.Bool("r", false, "recursive push if reach end of file")
	n := flag.Int("n", 1, "num of push connection")
	l := flag.String("l", "", "specify log file")
	flag.Parse()

	if *i == "" || *o == "" {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  %s -i test.flv -o rtmp://127.0.0.1:1935/live/test
  %s -i test.flv -o rtmp://127.0.0.1:1935/live/test -r
  %s -i test.flv -o rtmp://127.0.0.1:1935/live/test_{i} -r -n 1000
`, os.Args[0], os.Args[0], os.Args[0])
		base.OsExitAndWaitPressIfWindows(1)
	}
	return *i, *o, *n, *r, *l
}
