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

	"github.com/q191201771/lal/pkg/logic"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/bininfo"
	log "github.com/q191201771/naza/pkg/nazalog"
)

// rtmp 推流客户端，读取本地 flv 文件，使用 rtmp 协议推送出去
//
// 支持循环推送：文件推送完毕后，可循环推送（rtmp push 流并不断开）
// 支持推送多路流：相当于一个 rtmp 推流压测工具
//
// Usage of ./bin/flvfile2rtmppush:
// -i string
// specify flv file
// -n int
// num of push connection (default 1)
// -o string
// specify rtmp push url
// -r	recursive push if reach end of file
// -v	show bin info
// Example:
// ./bin/flvfile2rtmppush -i testdata/test.flv -o rtmp://127.0.0.1:19350/live/test
// ./bin/flvfile2rtmppush -i testdata/test.flv -o rtmp://127.0.0.1:19350/live/test -r
// ./bin/flvfile2rtmppush -i testdata/test.flv -o rtmp://127.0.0.1:19350/live/test_{i} -r -n 1000

func main() {
	log.Info(bininfo.StringifySingleLine())

	filename, urlTmpl, num, isRecursive := parseFlag()
	urls := collect(urlTmpl, num)

	tags := readAllTag(filename)
	log.Debug(urls, num)

	push(tags, urls, isRecursive)
	log.Info("bye.")
}

// readAllTag 预读取 flv 文件中的所有 tag，缓存在内存中
func readAllTag(filename string) (ret []httpflv.Tag) {
	var ffr httpflv.FLVFileReader
	err := ffr.Open(filename)
	log.FatalIfErrorNotNil(err)
	log.Infof("open succ. filename=%s", filename)

	for {
		tag, err := ffr.ReadTag()
		if err == io.EOF {
			log.Info("EOF")
			break
		}
		log.FatalIfErrorNotNil(err)
		ret = append(ret, tag)
	}
	log.Infof("read all tag done. num=%d", len(ret))
	return
}

func push(tags []httpflv.Tag, urls []string, isRecursive bool) {
	if len(tags) == 0 || len(urls) == 0 {
		return
	}

	var err error
	var psList []*rtmp.PushSession

	for i := range urls {
		ps := rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
			option.ConnectTimeoutMS = 3000
			option.PushTimeoutMS = 5000
			option.WriteAVTimeoutMS = 10000
		})
		err = ps.Push(urls[i])
		log.FatalIfErrorNotNil(err)
		log.Infof("push succ. url=%s", urls[i])
		psList = append(psList, ps)
	}

	var totalBaseTS uint32
	var prevTS uint32
	var hasReadThisBaseTS bool
	var thisBaseTS uint32
	var hasTraceFirstTagTS bool
	var firstTagTS uint32
	var firstTagTick int64

	for i := 0; ; i++ {
		log.Infof(" > round. i=%d, totalBaseTS=%d, prevTS=%d, thisBaseTS=%d",
			i, totalBaseTS, prevTS, thisBaseTS)

		hasReadThisBaseTS = false

		for _, tag := range tags {
			h := logic.Trans.FLVTagHeader2RTMPHeader(tag.Header)

			if tag.IsMetadata() {
				if totalBaseTS == 0 {
					// 第一个metadata直接发送
					//log.Debugf("CHEFERASEME write metadata.")
					h.TimestampAbs = 0
					chunks := rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h)
					for _, ps := range psList {
						err = ps.AsyncWrite(chunks)
						log.FatalIfErrorNotNil(err)
					}
				} else {
					// noop
				}
				continue
			}

			if hasReadThisBaseTS {
				// 之前已经读到了这轮读文件的base值，ts要减去base
				//log.Debugf("CHEFERASEME %+v %d %d %d.", tag.Header, tag.Header.Timestamp, thisBaseTS, totalBaseTS)
				h.TimestampAbs = tag.Header.Timestamp - thisBaseTS + totalBaseTS
			} else {
				// 设置base，ts设置为上一轮读文件的值
				//log.Debugf("CHEFERASEME %+v %d %d %d.", tag.Header, tag.Header.Timestamp, thisBaseTS, totalBaseTS)
				thisBaseTS = tag.Header.Timestamp
				h.TimestampAbs = totalBaseTS
				hasReadThisBaseTS = true
			}

			if h.TimestampAbs < prevTS {
				// ts比上一个包的还小，直接设置为上一包的值，并且不sleep直接发送
				h.TimestampAbs = prevTS
			}

			chunks := rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h)

			if hasTraceFirstTagTS {
				n := time.Now().UnixNano() / 1000000
				diffTick := n - firstTagTick
				diffTS := h.TimestampAbs - firstTagTS
				//log.Infof("%d %d %d %d", n, diffTick, diffTS, int64(diffTS) - diffTick)
				if diffTick < int64(diffTS) {
					time.Sleep(time.Duration(int64(diffTS)-diffTick) * time.Millisecond)
				}
			} else {
				firstTagTick = time.Now().UnixNano() / 1000000
				firstTagTS = h.TimestampAbs
				hasTraceFirstTagTS = true
			}

			for _, ps := range psList {
				err = ps.AsyncWrite(chunks)
				log.FatalIfErrorNotNil(err)
			}

			prevTS = h.TimestampAbs
		}

		totalBaseTS = prevTS + 1

		if !isRecursive {
			break
		}
	}
}

func collect(urlTmpl string, num int) (urls []string) {
	for i := 0; i < num; i++ {
		url := strings.Replace(urlTmpl, "{i}", strconv.Itoa(i), -1)
		urls = append(urls, url)
	}
	return
}

func parseFlag() (filename string, urlTmpl string, num int, isRecursive bool) {
	v := flag.Bool("v", false, "show bin info")
	i := flag.String("i", "", "specify flv file")
	o := flag.String("o", "", "specify rtmp push url")
	r := flag.Bool("r", false, "recursive push if reach end of file")
	n := flag.Int("n", 1, "num of push connection")
	flag.Parse()
	if *v {
		_, _ = fmt.Fprint(os.Stderr, bininfo.StringifyMultiLine())
		os.Exit(1)
	}
	if *i == "" || *o == "" {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  ./bin/flvfile2rtmppush -i testdata/test.flv -o rtmp://127.0.0.1:19350/live/test
  ./bin/flvfile2rtmppush -i testdata/test.flv -o rtmp://127.0.0.1:19350/live/test -r
  ./bin/flvfile2rtmppush -i testdata/test.flv -o rtmp://127.0.0.1:19350/live/test_{i} -r -n 1000
`)
		os.Exit(1)
	}
	return *i, *o, *n, *r
}
