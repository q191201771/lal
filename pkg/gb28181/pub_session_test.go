// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package gb28181

import (
	"encoding/hex"
	"fmt"
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestPubSession(t *testing.T) {
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
			_, _ = fp2.Write(packet.Payload)
		} else if packet.IsVideo() {
			_, _ = fp.Write(packet.Payload)
		}
	})

	go func() {
		time.Sleep(100 * time.Millisecond)

		conn, err := nazanet.NewUdpConnection(func(option *nazanet.UdpConnectionOption) {
			option.RAddr = addr
		})
		nazalog.Assert(nil, err)

		for i := 1; i < 1000; i++ {
			//filename := fmt.Sprintf("/tmp/rtp-h264-aac/%d.ps", i)
			filename := fmt.Sprintf("/tmp/rtp-ps-video/%d.ps", i)
			b, err := ioutil.ReadFile(filename)
			nazalog.Assert(nil, err)
			nazalog.Debugf("[test] %d: %s", i, hex.EncodeToString(b[12:]))

			conn.Write(b)
		}
	}()

	runErr := session.RunLoop(addr)
	nazalog.Assert(nil, runErr)
}
