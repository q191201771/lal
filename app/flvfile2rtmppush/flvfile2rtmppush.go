package main

import (
	"flag"
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/nezha/pkg/errors"
	"github.com/q191201771/nezha/pkg/log"
	"io"
	"os"
	"time"
)

// 将flv文件通过rtmp协议推送至rtmp服务器
//
// -r 表示当文件推送完毕后，是否循环推送
//
// Usage:
// ./bin/flvfile2rtmppush -r 1 -i /tmp/test.flv -o rtmp://push.xxx.com/live/testttt

func main() {
	var err error

	flvFileName, rtmpPushURL, isRecursive := parseFlag()

	ps := rtmp.NewPushSession(5000)
	err = ps.Push(rtmpPushURL)
	errors.PanicIfErrorOccur(err)
	log.Infof("push succ.")

	var baseTS int
	var prevTS int

	for {
		var ffr httpflv.FlvFileReader
		err = ffr.Open(flvFileName)
		errors.PanicIfErrorOccur(err)
		log.Infof("open succ.")

		flvHeader, err := ffr.ReadFlvHeader()
		errors.PanicIfErrorOccur(err)
		log.Infof("read flv header succ. %v", flvHeader)

		for {
			tag, err := ffr.ReadTag()
			if err == io.EOF {
				log.Info("EOF")
				break
			}
			errors.PanicIfErrorOccur(err)

			// TODO chef: 转换代码放入lal某个包中
			var h rtmp.Header
			h.MsgLen = int(tag.Header.DataSize) //len(tag.Raw)-httpflv.TagHeaderSize
			h.Timestamp = int(tag.Header.Timestamp) + int(baseTS)
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

			var diff int
			if h.Timestamp >= prevTS {
				diff = int(h.Timestamp) - prevTS
			} else {
				h.Timestamp = prevTS
			}

			chunks := rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h, rtmp.LocalChunkSize)

			log.Debugf("before send. diff=%d, ts=%d, prevTS=%d", diff, h.Timestamp, prevTS)
			time.Sleep(time.Duration(diff) * time.Millisecond)
			log.Debug("send")
			err = ps.TmpWrite(chunks)
			errors.PanicIfErrorOccur(err)
			prevTS = h.Timestamp
		}

		baseTS = prevTS + 1
		ffr.Dispose()

		if !isRecursive {
			break
		}
	}
}

func parseFlag() (string, string, bool) {
	i := flag.String("i", "", "specify flv file")
	o := flag.String("o", "", "specify rtmp push url")
	r := flag.Bool("r", false, "recursive push if reach end of file")
	flag.Parse()
	if *i == "" || *o == "" {
		flag.Usage()
		os.Exit(1)
	}
	return *i, *o, *r
}
