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
	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
	"io/ioutil"
	"net"
	"testing"
)

// TestRe_PsPubSession
//
// 重放业务方的ps流。
//
// 本测试函数模拟客户端，读取业务方对的dumpfile，重新推送给lalserver。
//
// 步骤：
//
// 1. 让业务方提供lalserver录制下来的dumpfile文件
// 2. 将dumpfile存放在下面filename变量处，或者修改下面filename变量值
// 3. 启动lalserver
// 4. 调用HTTP API
// curl -H "Content-Type:application/json" -X POST -d '{"stream_name": "test110", "port": 10002, "timeout_ms": 10000}' http://127.0.0.1:8083/api/ctrl/start_rtp_pub
// 5. 执行该测试
// go test -test.run TestDump_PsPub
func TestDump_PsPub(t *testing.T) {
	filename := "/tmp/record.psdata"
	isTcpFlag := 1

	b, err := ioutil.ReadFile(filename)
	if len(b) == 0 || err != nil {
		return
	}

	testPushFile("127.0.0.1:10002", filename, isTcpFlag)
}

// ---------------------------------------------------------------------------------------------------------------------

// testPushFile 创建udp客户端，向 addr 地址发送 filename 文件中的包
func testPushFile(addr string, filename string, isTcpFlag int) {
	var udpConn *nazanet.UdpConnection
	var tcpConn net.Conn
	var err error

	if isTcpFlag != 0 {
		tcpConn, err = net.Dial("tcp", addr)
		nazalog.Assert(nil, err)
	} else {
		udpConn, err = nazanet.NewUdpConnection(func(option *nazanet.UdpConnectionOption) {
			option.RAddr = addr
		})
		nazalog.Assert(nil, err)
	}

	df := base.NewDumpFile()
	err = df.OpenToRead(filename)
	nazalog.Assert(nil, err)

	lb := make([]byte, 2)
	for {
		m, err := df.ReadOneMessage()
		if err != nil {
			nazalog.Errorf("%+v", err)
			break
		}
		nazalog.Debugf("%s", m.DebugString())

		if isTcpFlag != 0 {
			bele.BePutUint16(lb, uint16(m.Len))
			_, err = tcpConn.Write(lb)
			nazalog.Assert(nil, err)
			_, err = tcpConn.Write(m.Body)
			nazalog.Assert(nil, err)
		} else {
			udpConn.Write(m.Body)
		}

		//time.Sleep(10 * time.Millisecond)
	}
}
