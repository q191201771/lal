// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package innertest

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/nazalog"
	"io"
	"testing"
)

// TestDump_Rtsp
//
// 重放业务方的rtsp流。
//
// 本测试函数模拟客户端，读取业务方对的dumpfile，解析为rtp，合帧，写flv文件
//
// 步骤：
//
// 1. 让业务方提供lalserver录制下来的dumpfile文件
// 2. 将dumpfile存放在下面filename变量处，或者修改下面filename变量值
// 3. 执行该测试
// go test -test.run TestDump_Rtsp
func TestDump_Rtsp(t *testing.T) {
	// TODO(chef): [test] 合帧测试，只有音频部分，没有视频部分 202211

	filename := "/tmp/outpullrtsp.laldump"
	outFlvFilename := "/tmp/outtestdumprtsp.flv"

	// 初始化输出的flv文件
	var fileWriter httpflv.FlvFileWriter
	err := fileWriter.Open(outFlvFilename)
	nazalog.Assert(nil, err)
	defer fileWriter.Dispose()
	err = fileWriter.WriteRaw(httpflv.FlvHeader)
	nazalog.Assert(nil, err)

	// 初始化remuxer
	remuxer := remux.NewAvPacket2RtmpRemuxer().WithOnRtmpMsg(func(msg base.RtmpMsg) {
		nazalog.Debugf("remuxer. %s", msg.DebugString())
		err = fileWriter.WriteTag(*remux.RtmpMsg2FlvTag(msg))
		nazalog.Assert(nil, err)
	})

	var ctx sdp.LogicContext
	var unpacker rtprtcp.IRtpUnpacker

	df := base.NewDumpFile()
	err = df.OpenToRead(filename)
	nazalog.Assert(nil, err)

	for {
		m, err := df.ReadOneMessage()
		nazalog.Debugf("< ReadOneMessage. %+v, %+v", m, err)
		if err == io.EOF {
			return
		}
		nazalog.Assert(nil, err)

		if m.Typ == base.DumpTypeInnerFileHeaderData {
			continue
		}

		if m.Typ != base.DumpTypeRtspRtpData && m.Typ != base.DumpTypeRtspSdpData {
			nazalog.Errorf("unknown type. typ=%d", m.Typ)
			return
		}

		if m.Typ == base.DumpTypeRtspSdpData {
			ctx, err = sdp.ParseSdp2LogicContext([]byte(m.Body))
			nazalog.Debugf("parse sdp, %+v, %+v", ctx, err)

			remuxer.OnSdp(ctx)
			unpacker = rtprtcp.DefaultRtpUnpackerFactory(ctx.GetAudioPayloadTypeBase(), ctx.AudioClockRate, 1024, func(pkt base.AvPacket) {
				nazalog.Debugf("unpacker. %s", pkt.DebugString())
				remuxer.OnAvPacket(pkt)
			})
			continue
		}

		pkt, err := rtprtcp.ParseRtpPacket(m.Body)
		nazalog.Debugf("< ParseRtpPacket. %+v, %+v", pkt, err)
		nazalog.Assert(nil, err)
		if ctx.IsAudioPayloadTypeOrigin(int(pkt.Header.PacketType)) {
			unpacker.Feed(pkt)
		}
	}
}
