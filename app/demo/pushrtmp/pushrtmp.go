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
	filename, urlTmpl, num, isRecursive, logfile := parseFlag()
	initLog(logfile)

	urls := collect(urlTmpl, num)

	tags, err := httpflv.ReadAllTagsFromFlvFile(filename)
	if err != nil {
		nazalog.Fatalf("read tags from flv file failed. err=%+v", err)
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
			push(tags, []string{u}, isRecursive)
			wg.Done()
			atomic.AddInt32(&aliveSessionCount, -1)
		}(url)
	}
	wg.Wait()
	time.Sleep(1 * time.Second)
	nazalog.Info("< main.")
}

func push(tags []httpflv.Tag, urls []string, isRecursive bool) {
	var sessionList []*rtmp.PushSession

	if len(tags) == 0 || len(urls) == 0 {
		return
	}

	var err error

	for i := range urls {
		ps := rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
			option.PushTimeoutMs = 5000
			option.WriteAvTimeoutMs = 10000
		})

		err = ps.Push(urls[i])
		if err != nil {
			nazalog.Errorf("push failed. err=%v", err)
			continue
		}
		atomic.AddInt32(&aliveSessionCount, 1)

		nazalog.Infof("push succ. url=%s", urls[i])
		sessionList = append(sessionList, ps)
	}
	check(sessionList)

	var totalBaseTs uint32 // 每轮最后更新
	var prevTs uint32      // 上一个tag
	var hasReadThisBaseTs bool
	var thisBaseTs uint32 // 每轮第一个tag
	var hasTraceFirstTagTs bool
	var firstTagTs uint32  // 所有轮第一个tag
	var firstTagTick int64 // 所有轮第一个tag的物理发送时间

	// 1. 保证metadata只在最初发送一次
	// 2. 多轮，时间戳会翻转，需要处理，让它线性增长

	// 多轮，一个循环代表一次完整文件的发送
	for i := 0; ; i++ {
		nazalog.Infof(" > round. i=%d, totalBaseTs=%d, prevTs=%d, thisBaseTs=%d",
			i, totalBaseTs, prevTs, thisBaseTs)

		hasReadThisBaseTs = false

		// 一轮，遍历文件的所有tag数据
		for _, tag := range tags {
			h := remux.FlvTagHeader2RtmpHeader(tag.Header)

			// metadata只发送一次
			if tag.IsMetadata() {
				if totalBaseTs == 0 {
					h.TimestampAbs = 0
					chunks := rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h)
					send(sessionList, chunks)
				} else {
					// noop
				}
				continue
			}

			if hasReadThisBaseTs {
				// 本轮非第一个tag

				// 之前已经读到了这轮读文件的base值，ts要减去base
				h.TimestampAbs = tag.Header.Timestamp - thisBaseTs + totalBaseTs
			} else {
				// 本轮第一个tag

				// 设置base，ts设置为上一轮读文件的值
				thisBaseTs = tag.Header.Timestamp
				h.TimestampAbs = totalBaseTs
				hasReadThisBaseTs = true
			}

			if h.TimestampAbs < prevTs {
				// ts比上一个包的还小，直接设置为上一包的值，并且不sleep直接发送
				h.TimestampAbs = prevTs
				nazalog.Errorf("this tag timestamp less than prev timestamp. h.TimestampAbs=%d, prevTs=%d", h.TimestampAbs, prevTs)
			}

			chunks := rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h)

			if hasTraceFirstTagTs {
				// 所有轮的非第一个tag

				// 当前距离第一个tag的物理发送时间，以及距离第一个tag的时间戳
				// 如果物理时间短，就睡眠相应的时间
				n := time.Now().UnixNano() / 1000000
				diffTick := n - firstTagTick
				diffTs := h.TimestampAbs - firstTagTs
				if diffTick < int64(diffTs) {
					time.Sleep(time.Duration(int64(diffTs)-diffTick) * time.Millisecond)
				}
			} else {
				// 所有轮的第一个tag

				// 记录所有轮的第一个tag的物理发送时间，以及数据的时间戳
				firstTagTick = time.Now().UnixNano() / 1000000
				firstTagTs = h.TimestampAbs
				hasTraceFirstTagTs = true
			}

			send(sessionList, chunks)

			prevTs = h.TimestampAbs
		} // tags for loop

		totalBaseTs = prevTs + 1

		if !isRecursive {
			break
		}
	}
}

func send(sessionList []*rtmp.PushSession, b []byte) {
	var s []*rtmp.PushSession
	for _, ps := range sessionList {
		if err := ps.Write(b); err != nil {
			nazalog.Errorf("write data error. err=%v", err)
			continue
		}
		s = append(s, ps)
	}
	sessionList = s

	check(sessionList)
}

func check(sessionList []*rtmp.PushSession) {
	if len(sessionList) == 0 {
		nazalog.Errorf("all push session dead.")
		os.Exit(1)
	}
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
