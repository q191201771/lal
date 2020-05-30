// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/q191201771/lal/pkg/rtmp"

	"github.com/q191201771/lal/pkg/avc"
	"github.com/q191201771/lal/pkg/hevc"
	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/bitrate"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/naza/pkg/nazalog"
)

// 分析诊断HTTP-FLV流的时间戳。注意，这个程序还没有完成。
// 功能：
// - 时间戳回退检查
//     - 当音频时间戳出现回退时打error日志
//     - 当视频时间戳出现回退时打error日志
//     - 将音频和视频时间戳看成一个整体，出现回退时打error日志
// - 定时打印：
//     - 总体带宽
//     - 音频带宽
//     - 视频带宽
//     - 视频DTS和PTS不相等的计数
// - H264
//     - 打印每个tag的类型：key seq header...
//     - 打印每个tag中有多少个帧：SPS PPS SEI IDR SLICE...
//     - 打印每个SLICE的类型：I、P、B...

// TODO
// - 解析metadata
// - 检查时间戳正向大的跳跃

var (
	printStatFlag        = true
	printEveryTagFlag    = false
	printMetaData        = true
	analysisVideoTagFlag = false
)

func main() {
	url := parseFlag()
	session := httpflv.NewPullSession()

	brTotal := bitrate.New()
	brAudio := bitrate.New()
	brVideo := bitrate.New()

	prevAudioTS := int64(-1)
	prevVideoTS := int64(-1)
	prevTS := int64(-1)

	videoCTSNotZeroCount := 0

	go func() {
		for {
			time.Sleep(1 * time.Second)
			if printStatFlag {
				nazalog.Debugf("stat. total=%dKb/s, audio=%dKb/s, video=%dKb/s, videoCTSNotZeroCount=%d", int(brTotal.Rate()), int(brAudio.Rate()), int(brVideo.Rate()), videoCTSNotZeroCount)
			}
		}
	}()

	err := session.Pull(url, func(tag httpflv.Tag) {
		if printEveryTagFlag {
			debugLength := 32
			if len(tag.Raw) < 32 {
				debugLength = len(tag.Raw)
			}
			nazalog.Debugf("header=%+v, hex=%s", tag.Header, hex.Dump(tag.Raw[11:debugLength]))
		}

		brTotal.Add(len(tag.Raw))

		switch tag.Header.Type {
		case httpflv.TagTypeMetadata:
			//nazalog.Debugf("----------\n", hex.Dump(tag.Raw))
			if printMetaData {
				// TODO chef: 这部分可以移入到rtmp package中
				_, l, err := rtmp.AMF0.ReadString(tag.Raw[11:])
				nazalog.Assert(nil, err)
				kv, _, err := rtmp.AMF0.ReadObject(tag.Raw[11+l:])
				nazalog.Assert(nil, err)
				var buf bytes.Buffer
				buf.WriteString(fmt.Sprintf("-----\ncount:%d\n", len(kv)))
				for k, v := range kv {
					buf.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
				}
				nazalog.Debugf("%+v", buf.String())
			}
		case httpflv.TagTypeAudio:
			brAudio.Add(len(tag.Raw))

			if prevAudioTS != -1 && int64(tag.Header.Timestamp) < prevAudioTS {
				nazalog.Errorf("audio timestamp error. header=%+v, prevAudioTS=%d, diff=%d", tag.Header, prevAudioTS, int64(tag.Header.Timestamp)-prevAudioTS)
			}
			if prevTS != -1 && int64(tag.Header.Timestamp) < prevTS {
				nazalog.Errorf("audio timestamp error. header=%+v, prevTS=%d, diff=%d", tag.Header, prevTS, int64(tag.Header.Timestamp)-prevTS)
			}
			prevAudioTS = int64(tag.Header.Timestamp)
			prevTS = int64(tag.Header.Timestamp)
		case httpflv.TagTypeVideo:
			if analysisVideoTagFlag {
				analysisVideoTag(tag)
			}

			videoCTS := bele.BEUint24(tag.Raw[13:])
			if videoCTS != 0 {
				videoCTSNotZeroCount++
			}

			brVideo.Add(len(tag.Raw))

			if prevVideoTS != -1 && int64(tag.Header.Timestamp) < prevVideoTS {
				nazalog.Errorf("video timestamp error. header=%+v, prevVideoTS=%d, diff=%d", tag.Header, prevVideoTS, int64(tag.Header.Timestamp)-prevVideoTS)
			}
			if prevTS != -1 && int64(tag.Header.Timestamp) < prevTS {
				nazalog.Errorf("video timestamp error. header=%+v, prevTS=%d, diff=%d", tag.Header, prevTS, int64(tag.Header.Timestamp)-prevTS)
			}
			prevVideoTS = int64(tag.Header.Timestamp)
			prevTS = int64(tag.Header.Timestamp)
		}
	})
	nazalog.Warn(err)
}

const (
	typeUnknown uint8 = 1
	typeAVC     uint8 = 2
	typeHEVC    uint8 = 3
)

var t uint8 = typeUnknown

func analysisVideoTag(tag httpflv.Tag) {
	var buf bytes.Buffer
	if tag.IsVideoKeySeqHeader() {
		if tag.IsAVCKeySeqHeader() {
			t = typeAVC
			buf.WriteString(" [AVC SeqHeader] ")
		} else if tag.IsHEVCKeySeqHeader() {
			t = typeHEVC
			buf.WriteString(" [HEVC SeqHeader] ")
		}
	} else {
		body := tag.Raw[11:]

		for i := 5; i != int(tag.Header.DataSize); {
			naluLen := bele.BEUint32(body[i:])
			switch t {
			case typeAVC:
				buf.WriteString(fmt.Sprintf(" [%s(%s)] ", avc.CalcNaluTypeReadable(body[i+4:]), avc.CalcSliceTypeReadable(body[i+4:])))
			case typeHEVC:
				buf.WriteString(fmt.Sprintf(" [%s] ", hevc.CalcNaluTypeReadable(body[i+4:])))
			}
			i = i + 4 + int(naluLen)
		}
	}
	nazalog.Debug(buf.String())
}

func parseFlag() string {
	url := flag.String("i", "", "specify http-flv url")
	flag.Parse()
	if *url == "" {
		flag.Usage()
		os.Exit(1)
	}
	return *url
}
