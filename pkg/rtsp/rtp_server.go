// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"github.com/q191201771/naza/pkg/nazalog"
)

type RTPServer struct {
	udpServer *UDPServer
}

func NewRTPServer(addr string) *RTPServer {
	var s RTPServer
	s.udpServer = NewUDPServer(addr, s.OnReadUDPPacket)
	return &s
}

func (r *RTPServer) OnReadUDPPacket(b []byte, addr string, err error) {
	nazalog.Debugf("< R length=%d, remote=%s, err=%v", len(b), addr, err)
	parseRTPPacket(b)
}

func (s *RTPServer) Listen() (err error) {
	return s.udpServer.Listen()
}

func (s *RTPServer) RunLoop() error {
	return s.udpServer.RunLoop()
}
