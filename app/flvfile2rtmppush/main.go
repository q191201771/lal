package main

import (
	"flag"
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/lal/pkg/util/log"
	"os"
	"time"
)

// 将flv文件通过rtmp协议推送至rtmp服务器
//
// Usage:
// ./bin/flvfile2rtmppush -i /tmp/test.flv -o rtmp://push.xxx.com/live/testttt

func main() {
	flvFileName, rtmpPushURL := parseFlag()

	var ffr httpflv.FlvFileReader
	err := ffr.Open(flvFileName)
	panicIfErr(err)

	log.Infof("open succ.")
	flvHeader, err := ffr.ReadFlvHeader()
	panicIfErr(err)
	log.Infof("read flv header succ. %v", flvHeader)

	ps := rtmp.NewPushSession(5000)
	err = ps.Push(rtmpPushURL)
	panicIfErr(err)
	log.Infof("push succ.")

	var prevTS uint32
	firstA := true
	firstV := true
	//var aPrevH *rtmp.Header
	//var vPrevH *rtmp.Header

	for i := 0; i < 1000*1000; i++ {
		tag, err := ffr.ReadTag()
		panicIfErr(err)
		//log.Infof("tag: %+v %v", tag.Header, tag.Raw[11:])
		log.Infof("tag: %+v %d", tag.Header, len(tag.Raw))

		// TODO chef: 转换代码放入lal某个包中
		var h rtmp.Header
		h.MsgLen = int(tag.Header.DataSize) //len(tag.Raw)-httpflv.TagHeaderSize
		h.Timestamp = int(tag.Header.Timestamp)
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

		// 把第一个音频和视频的时间戳改成0
		if tag.Header.T == httpflv.TagTypeAudio && firstA {
			h.Timestamp = 0
			firstA = false
		}
		if tag.Header.T == httpflv.TagTypeVideo && firstV {
			h.Timestamp = 0
			firstV = false
		}

		//var chunks []byte
		//if tag.Header.T == httpflv.TagTypeVideo {
		//	chunks = rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h, aPrevH, rtmp.LocalChunkSize)
		//	aPrevH = &h
		//}
		//if tag.Header.T == httpflv.TagTypeVideo {
		//	chunks = rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h, vPrevH, rtmp.LocalChunkSize)
		//	vPrevH = &h
		//}
		//if tag.Header.T == httpflv.TagTypeVideo {
		//	chunks = rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h, nil, rtmp.LocalChunkSize)
		//}
		chunks := rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h, nil, rtmp.LocalChunkSize)

		// 第一个包直接发送
		if prevTS == 0 {
			err = ps.TmpWrite(chunks)
			panicIfErr(err)
			prevTS = tag.Header.Timestamp
			continue
		}

		// 相等或回退了直接发送
		if tag.Header.Timestamp <= prevTS {
			err = ps.TmpWrite(chunks)
			panicIfErr(err)
			prevTS = tag.Header.Timestamp
			continue
		}

		if tag.Header.Timestamp > prevTS {
			diff := tag.Header.Timestamp - prevTS

			// 跳跃超过了30秒，直接发送
			if diff > 30000 {
				err = ps.TmpWrite(chunks)
				panicIfErr(err)
				prevTS = tag.Header.Timestamp
				continue
			}

			// 睡眠后发送，睡眠时长为时间戳间隔
			time.Sleep(time.Duration(diff) * time.Millisecond)
			err = ps.TmpWrite(chunks)
			panicIfErr(err)
			prevTS = tag.Header.Timestamp
			continue
		}

		panic("should never reach here.")
	}
	ffr.Dispose()
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}

func parseFlag() (string, string) {
	i := flag.String("i", "", "specify flv file")
	o := flag.String("o", "", "specify rtmp push url")
	flag.Parse()
	if *i == "" || *o == "" {
		flag.Usage()
		os.Exit(1)
	}
	return *i, *o
}
