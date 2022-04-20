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
	"github.com/q191201771/lal/pkg/avc"
	"github.com/q191201771/lal/pkg/remux"
	"io/ioutil"
	"os"
	"time"

	"github.com/q191201771/naza/pkg/nazalog"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/naza/pkg/bininfo"
)

//lal/app/demo/customize_lalserver
//
// 演示lalserver通过插件功能扩展输入流
// 提示，插件功能是基于代码层面的，和通过与lalserver建立连接将流发送到lalserver是两种不同的方式
//
// 提示，这个demo，可以看成是业务方基于lal实现的一个定制化（功能增强版）的lalserver应用
// 换句话说，它并不强制要求在lal的github repo下
//
// demo的具体功能是，读取一个h264 es流文件，并将这个流输入到lalserver中
// 注意，lalserver其实并不关心业务方的流的来源（比如网络or文件or其他）
// 也不关心原始流的格式
// 只要业务方将流转换成lalserver所要求的格式，调用相应的接口传入数据即可
//
// 业务方的流输入到lalserver后，就可以使用lalserver所支持的协议，从lalserver拉流了
//

func showHowToCustomizePub(lals logic.ILalServer) {
	const (
		customizePubStreamName = "c110"
		h264filename           = "/tmp/test.h264"

		durationInterval = uint32(66)
	)

	go func() {
		time.Sleep(200 * time.Millisecond)
		session, err := lals.AddCustomizePubSession(customizePubStreamName)
		nazalog.Assert(nil, err)
		session.WithOption(func(option *remux.AvPacketStreamOption) {
			option.VideoFormat = remux.AvPacketStreamVideoFormatAnnexb
		})

		// demo的输入比较简单，一次性将整个es文件读入
		content, err := ioutil.ReadFile(h264filename)
		nazalog.Assert(nil, err)

		timestamp := uint32(0)
		for {
			// 借助lal中的一个帮助函数，将es流切割成一个一个的nal
			// 提示，这里切割的是整个文件，并且，函数执行结束后，并没有退出for循环，换句话说，流会无限循环输入到lalserver中
			err = avc.IterateNaluAnnexb(content, func(nal []byte) {
				// 将nal数据转换为lalserver要求的格式输入
				packet := base.AvPacket{
					PayloadType: base.AvPacketPtAvc,
					Timestamp:   timestamp,
					Payload:     append(avc.NaluStartCode4, nal...),
				}
				session.FeedAvPacket(packet)

				// 发送完后，计算时间戳，并按帧间隔时间延时发送
				t := avc.ParseNaluType(nal[0])
				if t == avc.NaluTypeSps || t == avc.NaluTypePps || t == avc.NaluTypeSei {
					// noop
				} else {
					timestamp += durationInterval
					time.Sleep(time.Duration(durationInterval) * time.Millisecond)
				}
			})
		}
	}()
}

func main() {
	defer nazalog.Sync()

	confFilename := parseFlag()
	lals := logic.NewLalServer(func(option *logic.Option) {
		option.ConfFilename = confFilename
	})

	showHowToCustomizePub(lals)

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
