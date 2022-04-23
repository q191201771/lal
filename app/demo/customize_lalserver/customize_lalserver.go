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
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/q191201771/naza/pkg/nazalog"

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

func showHowToCustomizePub(lals logic.ILalServer) {
	const (
		customizePubStreamName = "c110"
		h264filename           = "/tmp/test.h264"
		aacfilename            = "/tmp/test.aac"

		audioDurationInterval = uint32(23)
		videoDurationInterval = uint32(66)
	)

	time.Sleep(200 * time.Millisecond)
	session, err := lals.AddCustomizePubSession(customizePubStreamName)
	nazalog.Assert(nil, err)
	session.WithOption(func(option *remux.AvPacketStreamOption) {
		option.VideoFormat = remux.AvPacketStreamVideoFormatAnnexb
	})

	audioContent, err := ioutil.ReadFile(aacfilename)
	nazalog.Assert(nil, err)

	videoContent, err := ioutil.ReadFile(h264filename)
	nazalog.Assert(nil, err)

	asc, err := aac.MakeAscWithAdtsHeader(audioContent[:aac.AdtsHeaderLength])
	nazalog.Assert(nil, err)

	session.FeedAudioSpecificConfig(asc)

	var (
		m      sync.Mutex
		audios []base.AvPacket
		videos []base.AvPacket
	)

	var reorderFeedFilterFn = func(packet base.AvPacket) {
		m.Lock()
		defer m.Unlock()

		if packet.IsAudio() {
			audios = append(audios, packet)
		} else {
			videos = append(videos, packet)
		}

		for len(audios) > 0 && len(videos) > 0 {
			if audios[0].Timestamp <= videos[0].Timestamp {
				session.FeedAvPacket(audios[0])
				audios = audios[1:]
			} else {
				session.FeedAvPacket(videos[0])
				videos = videos[1:]
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		i := 0
		timestamp := uint32(0)
		for {
			ctx, err := aac.NewAdtsHeaderContext(audioContent[i : i+aac.AdtsHeaderLength])
			nazalog.Assert(nil, err)

			packet := base.AvPacket{
				PayloadType: base.AvPacketPtAac,
				Timestamp:   timestamp,
				Payload:     audioContent[i+aac.AdtsHeaderLength : i+int(ctx.AdtsLength)],
			}
			reorderFeedFilterFn(packet)

			i += int(ctx.AdtsLength)

			timestamp += audioDurationInterval
			time.Sleep(time.Duration(audioDurationInterval) * time.Millisecond)

			if i == len(audioContent) {
				break
			}
		}

		wg.Done()
	}()

	go func() {
		timestamp := uint32(0)
		for {
			// 借助lal中的一个帮助函数，将es流切割成一个一个的nal
			// 提示，这里切割的是整个文件，并且，函数执行结束后，并没有退出for循环，换句话说，流会无限循环输入到lalserver中
			err = avc.IterateNaluAnnexb(videoContent, func(nal []byte) {
				// 将nal数据转换为lalserver要求的格式输入
				packet := base.AvPacket{
					PayloadType: base.AvPacketPtAvc,
					Timestamp:   timestamp,
					Payload:     append(avc.NaluStartCode4, nal...),
				}
				reorderFeedFilterFn(packet)

				// 发送完后，计算时间戳，并按帧间隔时间延时发送
				t := avc.ParseNaluType(nal[0])
				if t == avc.NaluTypeSps || t == avc.NaluTypePps || t == avc.NaluTypeSei {
					// noop
				} else {
					timestamp += videoDurationInterval
					time.Sleep(time.Duration(videoDurationInterval) * time.Millisecond)
				}
			})
			nazalog.Assert(nil, err)

			break
		}

		wg.Done()
	}()

	wg.Wait()

	lals.DelCustomizePubSession(session)
}

func main() {
	defer nazalog.Sync()

	confFilename := parseFlag()
	lals := logic.NewLalServer(func(option *logic.Option) {
		option.ConfFilename = confFilename
	})

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
