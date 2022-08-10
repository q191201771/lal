// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package gb28181

import (
	"fmt"
	"github.com/q191201771/lal/pkg/aac"
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestPubSession(t *testing.T) {
	// 重放业务方的流
	// 步骤：
	// 1. 业务方提供的lalserver录制下来的dump file
	// 2. 启动lalserver
	// 3. 调用HTTP API
	// 4. 执行该测试
	//testDumpFile("127.0.0.1:10002", "/tmp/test.psdata")

	// 读取一大堆.ps文件，并使用udp发送到`addr`地址（外部的，比如外部自己启动lalserver）
	// 步骤：
	// 1. 启动lalserver
	// 2. 调用HTTP API
	// 3. 执行该测试
	//helpUdpSend("127.0.0.1:10002")

	// 读取一大堆.ps文件，并使用udp发送到`addr`地址（内部启动了PubSession做接收）
	// 步骤：
	// 1. 执行该测试
	//testPubSession()
}

func testPubSession() {
	// 一个udp包一个文件，按行分隔，hex stream格式如下
	// 8060 0000 0000 0000 0beb c567 0000 01ba
	// 46ab 1ea9 4401 0139 9ffe ffff 0094 ab0d

	fp, err := os.Create("/tmp/udp2.h264")
	nazalog.Assert(nil, err)
	defer fp.Close()

	fp2, err := os.Create("/tmp/udp2.aac")
	nazalog.Assert(nil, err)
	defer fp2.Close()

	pool := nazanet.NewAvailUdpConnPool(1024, 10240)
	port, err := pool.Peek()
	nazalog.Assert(nil, err)
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	session := NewPubSession().WithOnAvPacket(func(packet *base.AvPacket) {
		nazalog.Infof("[test2] onAvPacket. packet=%s", packet.DebugString())
		if packet.IsAudio() {
			aac.NewAdtsHeaderContext(packet.Payload)

			_, _ = fp2.Write(packet.Payload)
		} else if packet.IsVideo() {
			_, _ = fp.Write(packet.Payload)
		}
	})

	go func() {
		time.Sleep(100 * time.Millisecond)

		helpUdpSend(addr)
	}()

	runErr := session.RunLoop(addr)
	nazalog.Assert(nil, runErr)
}

func helpUdpSend(addr string) {
	conn, err := nazanet.NewUdpConnection(func(option *nazanet.UdpConnectionOption) {
		option.RAddr = addr
	})
	nazalog.Assert(nil, err)
	for i := 1; i < 10000; i++ {
		filename := fmt.Sprintf("/tmp/rtp-h264-aac/%d.ps", i)
		//filename := fmt.Sprintf("/tmp/rtp-ps-video/%d.ps", i)
		b, err := ioutil.ReadFile(filename)
		nazalog.Assert(nil, err)
		//nazalog.Debugf("[test] %d: %s", i, hex.EncodeToString(b[12:]))

		conn.Write(b)
	}
}

func testDumpFile(addr string, filename string) {
	conn, err := nazanet.NewUdpConnection(func(option *nazanet.UdpConnectionOption) {
		option.RAddr = addr
	})
	nazalog.Assert(nil, err)

	df := base.NewDumpFile()
	err = df.OpenToRead(filename)
	nazalog.Assert(nil, err)

	for {
		m, err := df.ReadOneMessage()
		if err != nil {
			nazalog.Errorf("%+v", err)
			break
		}
		nazalog.Debugf("%s", m.DebugString())

		conn.Write(m.Body)
	}
}
