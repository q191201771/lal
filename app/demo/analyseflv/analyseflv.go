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
	"strconv"
	"strings"
	"time"

	"github.com/q191201771/naza/pkg/nazabytes"

	"github.com/q191201771/lal/pkg/base"

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
//     - I帧间隔时间
// - H264
//     - 打印每个tag的类型：key seq header...
//     - 打印每个tag中有多少个帧：SPS PPS SEI IDR SLICE...
//     - 打印每个SLICE的类型：I、P、B...
// 解析metadata信息，并打印

// TODO
// - 检查时间戳正向大的跳跃
// - 打印GOP中帧数量？
// - slice_num?
// - 输入源可以是httpflv，也可以是flv文件

var (
	timestampCheckFlag   = true
	printStatFlag        = true
	printEveryTagFlag    = false
	printMetaData        = true
	analysisVideoTagFlag = true
)

var (
	prevAudioTs = int64(-1)
	prevVideoTs = int64(-1)
	prevTs      = int64(-1)
	prevIdrTs   = int64(-1)
	diffIdrTs   = int64(-1)
)

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})
	defer nazalog.Sync()
	base.LogoutStartInfo()

	url := parseFlag()
	session := httpflv.NewPullSession()

	brTotal := bitrate.New(func(option *bitrate.Option) {
		option.WindowMs = 5000
	})
	brAudio := bitrate.New(func(option *bitrate.Option) {
		option.WindowMs = 5000
	})
	brVideo := bitrate.New(func(option *bitrate.Option) {
		option.WindowMs = 5000
	})

	videoCtsNotZeroCount := 0

	go func() {
		for {
			time.Sleep(5 * time.Second)
			if printStatFlag {
				nazalog.Debugf("stat. total=%dKb/s, audio=%dKb/s, video=%dKb/s, videoCtsNotZeroCount=%d, diffIdrTs=%d",
					int(brTotal.Rate()), int(brAudio.Rate()), int(brVideo.Rate()), videoCtsNotZeroCount, diffIdrTs)
			}
		}
	}()

	err := session.Pull(url, func(tag httpflv.Tag) {
		if printEveryTagFlag {
			nazalog.Debugf("header=%+v, hex=%s", tag.Header, hex.Dump(nazabytes.Prefix(tag.Payload(), 32)))
		}

		brTotal.Add(len(tag.Raw))

		switch tag.Header.Type {
		case httpflv.TagTypeMetadata:
			if printMetaData {
				nazalog.Debugf("----------\n%s", hex.Dump(tag.Payload()))

				opa, err := rtmp.ParseMetadata(tag.Payload())
				nazalog.Assert(nil, err)
				var buf bytes.Buffer
				buf.WriteString(fmt.Sprintf("-----\ncount:%d\n", len(opa)))
				for _, op := range opa {
					buf.WriteString(fmt.Sprintf("  %s: %+v\n", op.Key, op.Value))
				}
				nazalog.Debugf("%+v", buf.String())
			}
		case httpflv.TagTypeAudio:
			nazalog.Debugf("header=%+v, %+v", tag.Header, tag.IsAacSeqHeader())
			brAudio.Add(len(tag.Raw))

			if tag.IsAacSeqHeader() {
				s := session.GetStat()
				nazalog.Infof("aac seq header. readBytes=%d, %s", s.ReadBytesSum, hex.EncodeToString(tag.Payload()))
			}
			if timestampCheckFlag {
				if prevAudioTs != -1 && int64(tag.Header.Timestamp) < prevAudioTs {
					nazalog.Errorf("audio timestamp error, less than prev audio timestamp. header=%+v, prevAudioTs=%d, diff=%d", tag.Header, prevAudioTs, int64(tag.Header.Timestamp)-prevAudioTs)
				}
				if prevTs != -1 && int64(tag.Header.Timestamp) < prevTs {
					nazalog.Warnf("audio timestamp error. less than prev global timestamp. header=%+v, prevTs=%d, diff=%d", tag.Header, prevTs, int64(tag.Header.Timestamp)-prevTs)
				}
			}
			prevAudioTs = int64(tag.Header.Timestamp)
			prevTs = int64(tag.Header.Timestamp)
		case httpflv.TagTypeVideo:
			analysisVideoTag(tag)

			videoCts := bele.BeUint24(tag.Raw[13:])
			if videoCts != 0 {
				videoCtsNotZeroCount++
			}

			brVideo.Add(len(tag.Raw))

			if timestampCheckFlag {
				if prevVideoTs != -1 && int64(tag.Header.Timestamp) < prevVideoTs {
					nazalog.Errorf("video timestamp error, less than prev video timestamp. header=%+v, prevVideoTs=%d, diff=%d", tag.Header, prevVideoTs, int64(tag.Header.Timestamp)-prevVideoTs)
				}
				if prevTs != -1 && int64(tag.Header.Timestamp) < prevTs {
					nazalog.Warnf("video timestamp error, less than prev global timestamp. header=%+v, prevTs=%d, diff=%d", tag.Header, prevTs, int64(tag.Header.Timestamp)-prevTs)
				}
			}
			prevVideoTs = int64(tag.Header.Timestamp)
			prevTs = int64(tag.Header.Timestamp)
		}
	})
	nazalog.Assert(nil, err)

	// 临时测试一下主动关闭client session
	//go func() {
	//	time.Sleep(5 * time.Second)
	//	_ = session.Dispose()
	//}()

	err = <-session.WaitChan()
	nazalog.Errorf("< session.WaitChan. err=%+v", err)
}

const (
	typeUnknown uint8 = 1
	typeAvc     uint8 = 2
	typeHevc    uint8 = 3
)

var t uint8 = typeUnknown

func analysisVideoTag(tag httpflv.Tag) {
	var buf bytes.Buffer
	if tag.IsVideoKeySeqHeader() {
		if tag.IsAvcKeySeqHeader() {
			t = typeAvc
			buf.WriteString(" [AVC SeqHeader] ")
			sps, pps, err := avc.ParseSpsPpsFromSeqHeader(tag.Payload())
			if err != nil {
				buf.WriteString(" parse sps pps failed.")
			}
			nazalog.Debugf("sps:%s, pps:%s", hex.Dump(sps), hex.Dump(pps))
		} else if tag.IsHevcKeySeqHeader() {
			t = typeHevc
			buf.WriteString(" [HEVC SeqHeader] ")
			buf.WriteString(hex.Dump(tag.Payload()))
			if _, _, _, err := hevc.ParseVpsSpsPpsFromSeqHeader(tag.Payload()); err != nil {
				buf.WriteString(" parse vps sps pps failed.")
			}
		}
	} else {
		cts := bele.BeUint24(tag.Payload()[2:])
		buf.WriteString(fmt.Sprintf("%+v, cts=%d, pts=%d", tag.Header, cts, tag.Header.Timestamp+cts))

		body := tag.Payload()[5:]
		nals, err := avc.SplitNaluAvcc(body)
		nazalog.Assert(nil, err)

		for _, nal := range nals {
			switch t {
			case typeAvc:
				if avc.ParseNaluType(nal[0]) == avc.NaluTypeIdrSlice {
					nazalog.Debugf("IDR:%s", hex.Dump(nazabytes.Prefix(nal, 128)))
					if prevIdrTs != int64(-1) {
						diffIdrTs = int64(tag.Header.Timestamp) - prevIdrTs
					}
					prevIdrTs = int64(tag.Header.Timestamp)
				}
				if avc.ParseNaluType(nal[0]) == avc.NaluTypeSei {
					delay := SeiDelayMs(nal)
					if delay != -1 {
						buf.WriteString(fmt.Sprintf("delay: %dms", delay))
					}
				}
				sliceTypeReadable, _ := avc.ParseSliceTypeReadable(nal)
				buf.WriteString(fmt.Sprintf(" [%s(%s)(%d)] ", avc.ParseNaluTypeReadable(nal[0]), sliceTypeReadable, len(nal)))
			case typeHevc:
				if hevc.ParseNaluType(nal[0]) == hevc.NaluTypeSei {
					delay := SeiDelayMs(nal)
					if delay != -1 {
						buf.WriteString(fmt.Sprintf("delay: %dms", delay))
					}
				}
				buf.WriteString(fmt.Sprintf(" [%s(%d)] ", hevc.ParseNaluTypeReadable(nal[0]), nal[0]))
			}
		}
	}
	if analysisVideoTagFlag {
		nazalog.Debug(buf.String())
	}
}

// 注意，SEI的内容是自定义格式，解析的代码不具有通用性
func SeiDelayMs(seiNalu []byte) int {
	//nazalog.Debugf("sei: %s", hex.Dump(seiNalu))
	items := strings.Split(string(seiNalu), ":")
	if len(items) != 3 {
		return -1
	}

	a, err := strconv.ParseInt(items[1], 10, 64)
	if err != nil {
		return -1
	}
	t := time.Unix(a/1e3, a%1e3)
	d := time.Now().Sub(t)
	return int(d.Nanoseconds() / 1e6)
}

func parseFlag() string {
	url := flag.String("i", "", "specify http-flv url")
	flag.Parse()
	if *url == "" {
		flag.Usage()
		base.OsExitAndWaitPressIfWindows(1)
	}
	return *url
}
