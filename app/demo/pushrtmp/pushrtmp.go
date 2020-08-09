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
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/q191201771/naza/pkg/bitrate"

	"github.com/q191201771/lal/pkg/logic"

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

var pss []*rtmp.PushSession
var br bitrate.Bitrate

func main() {
	filename, urlTmpl, num, isRecursive, logfile := parseFlag()
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
	}

	urls := collect(urlTmpl, num)

	tags := readAllTag(filename)

	br = bitrate.New()

	go func() {
		for {
			time.Sleep(1 * time.Second)
			rate := br.Rate()
			nazalog.Debugf("bitrate=%.3fkbit/s", rate)
			if rate > 1024*10 {
				nazalog.Errorf("bitrate too large. bitrate=%.3fkbit/s", rate)
				os.Exit(1)
			}
		}
	}()

	push(tags, urls, isRecursive)
	nazalog.Info("bye.")
}

// readAllTag 预读取 flv 文件中的所有 tag，缓存在内存中
func readAllTag(filename string) (ret []httpflv.Tag) {
	var ffr httpflv.FLVFileReader
	err := ffr.Open(filename)
	if err != nil {
		nazalog.Errorf("open file failed. file=%s, err=%v", filename, err)
		os.Exit(1)
	}
	nazalog.Infof("open succ. filename=%s", filename)

	for {
		tag, err := ffr.ReadTag()
		if err == io.EOF {
			nazalog.Info("EOF")
			break
		}
		if err != nil {
			nazalog.Errorf("read file tag error. tag num=%d, err=%v", len(ret), err)
			break
		}
		if tag.IsMetadata() {
			nazalog.Debugf("M %d", tag.Header.Timestamp)
		} else if tag.IsVideoKeySeqHeader() {
			nazalog.Debugf("V SH %d", tag.Header.Timestamp)
		} else if tag.IsVideoKeyNALU() {
			nazalog.Debugf("V K %d", tag.Header.Timestamp)
		} else if tag.IsAACSeqHeader() {
			nazalog.Debugf("A SH %d", tag.Header.Timestamp)
		}
		ret = append(ret, tag)
	}
	nazalog.Infof("read all tag done. tag num=%d", len(ret))
	return
}

func push(tags []httpflv.Tag, urls []string, isRecursive bool) {
	if len(tags) == 0 || len(urls) == 0 {
		return
	}

	var err error

	for i := range urls {
		ps := rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
			option.ConnectTimeoutMS = 3000
			option.PushTimeoutMS = 5000
			option.WriteAVTimeoutMS = 10000
		})

		err = ps.Push(urls[i])
		if err != nil {
			nazalog.Errorf("push failed. err=%v", err)
			continue
		}

		nazalog.Infof("push succ. url=%s", urls[i])
		pss = append(pss, ps)
	}
	check()

	var totalBaseTS uint32 // 每轮最后更新
	var prevTS uint32      // 上一个tag
	var hasReadThisBaseTS bool
	var thisBaseTS uint32 // 每轮第一个tag
	var hasTraceFirstTagTS bool
	var firstTagTS uint32  // 所有轮第一个tag
	var firstTagTick int64 // 所有轮第一个tag的物理发送时间

	// 1. 保证metadata只在最初发送一次
	// 2. 多轮，时间戳会翻转，需要处理，让它线性增长

	// 多轮，一个循环代表一次完整文件的发送
	for i := 0; ; i++ {
		nazalog.Infof(" > round. i=%d, totalBaseTS=%d, prevTS=%d, thisBaseTS=%d",
			i, totalBaseTS, prevTS, thisBaseTS)

		hasReadThisBaseTS = false

		// 一轮，遍历文件的所有tag数据
		for _, tag := range tags {
			h := logic.Trans.FLVTagHeader2RTMPHeader(tag.Header)

			// metadata只发送一次
			if tag.IsMetadata() {
				if totalBaseTS == 0 {
					h.TimestampAbs = 0
					chunks := rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h)
					send(chunks)
				} else {
					// noop
				}
				continue
			}

			if hasReadThisBaseTS {
				// 本轮非第一个tag

				// 之前已经读到了这轮读文件的base值，ts要减去base
				h.TimestampAbs = tag.Header.Timestamp - thisBaseTS + totalBaseTS
			} else {
				// 本轮第一个tag

				// 设置base，ts设置为上一轮读文件的值
				thisBaseTS = tag.Header.Timestamp
				h.TimestampAbs = totalBaseTS
				hasReadThisBaseTS = true
			}

			if h.TimestampAbs < prevTS {
				// ts比上一个包的还小，直接设置为上一包的值，并且不sleep直接发送
				h.TimestampAbs = prevTS
				nazalog.Errorf("this tag timestamp less than prev timestamp. h.TimestampAbs=%d, prevTS=%d", h.TimestampAbs, prevTS)
			}

			chunks := rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h)

			if hasTraceFirstTagTS {
				// 所有轮的非第一个tag

				// 当前距离第一个tag的物理发送时间，以及距离第一个tag的时间戳
				// 如果物理时间短，就睡眠相应的时间
				n := time.Now().UnixNano() / 1000000
				diffTick := n - firstTagTick
				diffTS := h.TimestampAbs - firstTagTS
				if diffTick < int64(diffTS) {
					time.Sleep(time.Duration(int64(diffTS)-diffTick) * time.Millisecond)
				}
			} else {
				// 所有轮的第一个tag

				// 记录所有轮的第一个tag的物理发送时间，以及数据的时间戳
				firstTagTick = time.Now().UnixNano() / 1000000
				firstTagTS = h.TimestampAbs
				hasTraceFirstTagTS = true
			}

			send(chunks)

			prevTS = h.TimestampAbs
		} // tags for loop

		totalBaseTS = prevTS + 1

		if !isRecursive {
			break
		}
	}
}

func send(b []byte) {
	br.Add(len(b))

	var s []*rtmp.PushSession
	for _, ps := range pss {
		if err := ps.AsyncWrite(b); err != nil {
			nazalog.Errorf("write data error. err=%v", err)
			continue
		}
		s = append(s, ps)
	}
	pss = s

	check()
}

func check() {
	if len(pss) == 0 {
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
  ./bin/pushrtmp -i testdata/test.flv -o rtmp://127.0.0.1:19350/live/test
  ./bin/pushrtmp -i testdata/test.flv -o rtmp://127.0.0.1:19350/live/test -r
  ./bin/pushrtmp -i testdata/test.flv -o rtmp://127.0.0.1:19350/live/test_{i} -r -n 1000
`)
		os.Exit(1)
	}
	return *i, *o, *n, *r, *l
}
