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
	"github.com/q191201771/lal/pkg/aac"
	"github.com/q191201771/lal/pkg/avc"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/naza/pkg/nazalog"
	"io/ioutil"
	"os"
	"time"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/naza/pkg/bininfo"
)

// lal/app/demo/customize_lalserver
//
// [what]
// 演示业务方如何通过lalserver的插件功能，将自身的流输入到lalserver中
//
// [why]
// 业务方的流输入到lalserver后，就可以使用lalserver的功能了，比如录制功能，（使用lalserver所支持的协议）从lalserver拉流等等
//
// 提示，插件功能是基于代码层面的，和与lalserver建立连接将流发送到lalserver是两种不同的方式，但是流进入lalserver后，效果是一样的
//
// 提示，这个demo，可以看成是业务方基于lal实现的一个定制化（功能增强版）的lalserver应用
// 换句话说，它并不强制要求在lal的github repo下
//
// [how]
// demo的具体功能是，分别读取一个h264 es流文件和一个aac es流文件，并将音视频流输入到lalserver中
//
// 注意，其实lalserver并不关心业务方的流的来源（比如网络or文件or其他），也不关心流的原始格式
// 业务方只要将流转换成lalserver所要求的格式，调用相应的接口传入数据即可
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
	session.WithOption(func(option *remux.AvPacketStreamOption) {
		option.VideoFormat = remux.AvPacketStreamVideoFormatAnnexb
	})

	asc, err := aac.MakeAscWithAdtsHeader(audioContent[:aac.AdtsHeaderLength])
	nazalog.Assert(nil, err)
	// 3. 填入aac的audio specific config信息
	session.FeedAudioSpecificConfig(asc)

	// 4. 按时间戳间隔匀速发送音频和视频
	startRealTime := time.Now()
	startTs := uint32(0)
	for i := range packets {
		diffTs := time.Duration(packets[i].Timestamp - startTs)
		diffReal := time.Duration(time.Now().Sub(startRealTime).Milliseconds())
		//nazalog.Debugf("%d: %s, %d, %d", i, packets[i].DebugString(), diffTs, diffReal)
		if diffReal < diffTs {
			time.Sleep((diffTs - diffReal) * time.Millisecond)
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
			Timestamp:   uint32(timestamp),
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
			Timestamp:   uint32(timestamp),
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
