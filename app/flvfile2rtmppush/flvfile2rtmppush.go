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
	"time"

	"github.com/q191201771/lal/pkg/logic"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/bininfo"
	log "github.com/q191201771/naza/pkg/nazalog"
)

//rtmp推流客户端，输入是本地flv文件，文件推送完毕后，可循环推送（rtmp push流并不断开）
//
// -r 为1时表示当文件推送完毕后，是否循环推送（rtmp push流并不断开）
//
// Usage:
// ./bin/flvfile2rtmppush -r 1 -i /tmp/test.flv -o rtmp://push.xxx.com/live/testttt

func main() {
	var err error

	flvFileName, rtmpPushURL, isRecursive := parseFlag()

	log.Info(bininfo.StringifySingleLine())

	ps := rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
		option.ConnectTimeoutMS = 3000
		option.PushTimeoutMS = 5000
		option.WriteAVTimeoutMS = 10000
	})
	err = ps.Push(rtmpPushURL)
	log.FatalIfErrorNotNil(err)
	log.Infof("push succ. url=%s", rtmpPushURL)

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

		var ffr httpflv.FLVFileReader
		err = ffr.Open(flvFileName)
		log.FatalIfErrorNotNil(err)
		log.Infof("open succ. filename=%s", flvFileName)

		hasReadThisBaseTS = false

		for {
			tag, err := ffr.ReadTag()
			if err == io.EOF {
				log.Info("EOF")
				break
			}
			log.FatalIfErrorNotNil(err)

			h := logic.Trans.FLVTagHeader2RTMPHeader(tag.Header)

			if tag.IsMetadata() {
				if totalBaseTS == 0 {
					// 第一个metadata直接发送
					//log.Debugf("CHEFERASEME write metadata.")
					h.TimestampAbs = 0
					chunks := rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h)
					err = ps.AsyncWrite(chunks)
					log.FatalIfErrorNotNil(err)
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

			err = ps.AsyncWrite(chunks)
			log.FatalIfErrorNotNil(err)

			prevTS = h.TimestampAbs
		}

		totalBaseTS = prevTS + 1
		ffr.Dispose()

		if !isRecursive {
			break
		}
	}
}

func parseFlag() (string, string, bool) {
	v := flag.Bool("v", false, "show bin info")
	i := flag.String("i", "", "specify flv file")
	o := flag.String("o", "", "specify rtmp push url")
	r := flag.Bool("r", false, "recursive push if reach end of file")
	flag.Parse()
	if *v {
		_, _ = fmt.Fprint(os.Stderr, bininfo.StringifyMultiLine())
		os.Exit(1)
	}
	if *i == "" || *o == "" {
		flag.Usage()
		os.Exit(1)
	}
	return *i, *o, *r
}
