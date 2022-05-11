// Copyright 2022, Chef.  All rights reserved.
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
	"github.com/q191201771/lal/pkg/aac"
	"github.com/q191201771/lal/pkg/avc"
	"github.com/q191201771/naza/pkg/nazalog"
	"io/ioutil"
	"os"
	"time"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/naza/pkg/bininfo"
)

// 文档见 <lalserver二次开发 - pub接入自定义流>
// https://pengrl.com/lal/#/customize_pub
//

func main() {
	defer nazalog.Sync()

	confFilename := parseFlag()
	lals := logic.NewLalServer(func(option *logic.Option) {
		option.ConfFilename = confFilename
	})

	// 比常规lalserver多加了这一行
	go showHowToCustomizePub(lals)

	err := lals.RunLoop()
	nazalog.Infof("server manager done. err=%+v", err)
}

func parseFlag() string {
	binInfoFlag := flag.Bool("v", false, "show bin info")
	cf := flag.String("c", "", "specify conf file")
	flag.Parse()

	if *binInfoFlag {
		_, _ = fmt.Fprint(os.Stderr, bininfo.StringifyMultiLine())
		_, _ = fmt.Fprintln(os.Stderr, base.LalFullInfo)
		os.Exit(0)
	}

	return *cf
}

func showHowToCustomizePub(lals logic.ILalServer) {
	const (
		h264filename = "/tmp/test.h264"
		aacfilename  = "/tmp/test.aac"

		customizePubStreamName = "c110"
	)

	time.Sleep(200 * time.Millisecond)

	// 从音频和视频各自的ES流文件中读取出所有数据
	// 然后将它们按时间戳排序，合并到一个AvPacket数组中
	audioContent, audioPackets := readAudioPacketsFromFile(aacfilename)
	_, videoPackets := readVideoPacketsFromFile(h264filename)
	packets := mergePackets(audioPackets, videoPackets)

	// 1. 向lalserver中加入自定义的pub session
	session, err := lals.AddCustomizePubSession(customizePubStreamName)
	nazalog.Assert(nil, err)
	// 2. 配置session
	session.WithOption(func(option *base.AvPacketStreamOption) {
		option.VideoFormat = base.AvPacketStreamVideoFormatAnnexb
	})

	asc, err := aac.MakeAscWithAdtsHeader(audioContent[:aac.AdtsHeaderLength])
	nazalog.Assert(nil, err)
	// 3. 填入aac的audio specific config信息
	session.FeedAudioSpecificConfig(asc)

	// 4. 按时间戳间隔匀速发送音频和视频
	startRealTime := time.Now()
	startTs := int64(0)
	for i := range packets {
		diffTs := packets[i].Timestamp - startTs
		diffReal := time.Now().Sub(startRealTime).Milliseconds()
		//nazalog.Debugf("%d: %s, %d, %d", i, packets[i].DebugString(), diffTs, diffReal)
		if diffReal < diffTs {
			time.Sleep(time.Duration(diffTs-diffReal) * time.Millisecond)
		}
		session.FeedAvPacket(packets[i])
	}

	// 5. 所有数据发送关闭后，将pub session从lal server移除
	lals.DelCustomizePubSession(session)
}

// readAudioPacketsFromFile 从aac es流文件读取所有音频包
//
func readAudioPacketsFromFile(filename string) (audioContent []byte, audioPackets []base.AvPacket) {
	var err error
	audioContent, err = ioutil.ReadFile(filename)
	nazalog.Assert(nil, err)

	pos := 0
	timestamp := float32(0)
	for {
		ctx, err := aac.NewAdtsHeaderContext(audioContent[pos : pos+aac.AdtsHeaderLength])
		nazalog.Assert(nil, err)

		packet := base.AvPacket{
			PayloadType: base.AvPacketPtAac,
			Timestamp:   int64(timestamp),
			Payload:     audioContent[pos+aac.AdtsHeaderLength : pos+int(ctx.AdtsLength)],
		}

		audioPackets = append(audioPackets, packet)

		timestamp += float32(48000*4*2) / float32(8192*2) // (frequence * bytePerSample * channel) / (packetSize * channel)

		pos += int(ctx.AdtsLength)
		if pos == len(audioContent) {
			break
		}
	}

	return
}

// readVideoPacketsFromFile 从h264 es流文件读取所有视频包
//
func readVideoPacketsFromFile(filename string) (videoContent []byte, videoPackets []base.AvPacket) {
	var err error
	videoContent, err = ioutil.ReadFile(filename)
	nazalog.Assert(nil, err)

	timestamp := float32(0)
	err = avc.IterateNaluAnnexb(videoContent, func(nal []byte) {
		// 将nal数据转换为lalserver要求的格式输入
		packet := base.AvPacket{
			PayloadType: base.AvPacketPtAvc,
			Timestamp:   int64(timestamp),
			Payload:     append(avc.NaluStartCode4, nal...),
		}

		videoPackets = append(videoPackets, packet)

		t := avc.ParseNaluType(nal[0])
		if t == avc.NaluTypeSps || t == avc.NaluTypePps || t == avc.NaluTypeSei {
			// noop
		} else {
			timestamp += float32(1000) / float32(15) // 1秒 / fps
		}
	})
	nazalog.Assert(nil, err)

	return
}

// mergePackets 将音频队列和视频队列按时间戳有序合并为一个队列
//
func mergePackets(audioPackets, videoPackets []base.AvPacket) (packets []base.AvPacket) {
	var i, j int
	for {
		// audio数组为空，将video的剩余数据取出，然后merge结束
		if i == len(audioPackets) {
			packets = append(packets, videoPackets[j:]...)
			break
		}

		//
		if j == len(videoPackets) {
			packets = append(packets, audioPackets[i:]...)
			break
		}

		// 音频和视频都有数据，取时间戳小的
		if audioPackets[i].Timestamp < videoPackets[j].Timestamp {
			packets = append(packets, audioPackets[i])
			i++
		} else {
			packets = append(packets, videoPackets[j])
			j++
		}
	}

	return
}
