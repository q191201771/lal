package main

import (
	"flag"
	"fmt"
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/nezha/pkg/bininfo"
	"github.com/q191201771/nezha/pkg/log"
	"io"
	"os"
	"time"
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

	ps := rtmp.NewPushSession(rtmp.PushSessionTimeout{
		ConnectTimeoutMS: 3000,
		PushTimeoutMS:    5000,
		WriteAVTimeoutMS: 10000,
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

		var ffr httpflv.FlvFileReader
		err = ffr.Open(flvFileName)
		log.FatalIfErrorNotNil(err)
		log.Infof("open succ. filename=%s", flvFileName)

		flvHeader, err := ffr.ReadFlvHeader()
		log.FatalIfErrorNotNil(err)
		log.Infof("read flv header succ. %v", flvHeader)

		hasReadThisBaseTS = false

		for {
			tag, err := ffr.ReadTag()
			if err == io.EOF {
				log.Info("EOF")
				break
			}
			log.FatalIfErrorNotNil(err)

			// TODO chef: 转换代码放入lal某个包中
			var h rtmp.Header
			h.MsgLen = int(tag.Header.DataSize) //len(tag.Raw)-httpflv.TagHeaderSize

			h.MsgTypeID = int(tag.Header.T)
			h.MsgStreamID = rtmp.MSID1
			switch tag.Header.T {
			case httpflv.TagTypeMetadata:
				h.CSID = rtmp.CSIDAMF
			case httpflv.TagTypeAudio:
				h.CSID = rtmp.CSIDAudio
			case httpflv.TagTypeVideo:
				h.CSID = rtmp.CSIDVideo
			}

			if tag.Header.T == httpflv.TagTypeMetadata {
				if totalBaseTS == 0 {
					// 第一个metadata直接发送
					//log.Debugf("CHEFERASEME write metadata.")
					chunks := rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h, rtmp.LocalChunkSize)
					err = ps.TmpWrite(chunks)
					log.FatalIfErrorNotNil(err)
				} else {
					// noop
				}
				continue
			}

			if hasReadThisBaseTS {
				// 之前已经读到了这轮读文件的base值，ts要减去base
				//log.Debugf("CHEFERASEME %+v %d %d %d.", tag.Header, tag.Header.Timestamp, thisBaseTS, totalBaseTS)
				h.Timestamp = tag.Header.Timestamp - thisBaseTS + totalBaseTS
			} else {
				// 设置base，ts设置为上一轮读文件的值
				//log.Debugf("CHEFERASEME %+v %d %d %d.", tag.Header, tag.Header.Timestamp, thisBaseTS, totalBaseTS)
				thisBaseTS = tag.Header.Timestamp
				h.Timestamp = totalBaseTS
				hasReadThisBaseTS = true
			}

			if h.Timestamp < prevTS {
				// ts比上一个包的还小，直接设置为上一包的值，并且不sleep直接发送
				h.Timestamp = prevTS
			}

			chunks := rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h, rtmp.LocalChunkSize)

			if hasTraceFirstTagTS {
				n := time.Now().UnixNano() / 1000000
				diffTick := n - firstTagTick
				diffTS := h.Timestamp - firstTagTS
				//log.Infof("%d %d %d %d", n, diffTick, diffTS, int64(diffTS) - diffTick)
				if diffTick < int64(diffTS) {
					time.Sleep(time.Duration(int64(diffTS)-diffTick) * time.Millisecond)
				}
			} else {
				firstTagTick = time.Now().UnixNano() / 1000000
				firstTagTS = h.Timestamp
				hasTraceFirstTagTS = true
			}

			err = ps.TmpWrite(chunks)
			log.FatalIfErrorNotNil(err)

			prevTS = h.Timestamp
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
