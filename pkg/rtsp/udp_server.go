// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"net"

	"github.com/q191201771/naza/pkg/nazalog"
)

// TODO chef: 这个文件考虑弄到naza中去

type OnReadUDPPacket func(b []byte, addr string, err error)

type UDPServer struct {
	addr            string
	onReadUDPPacket OnReadUDPPacket
	conn            *net.UDPConn
}

func NewUDPServer(addr string, onReadUDPPacket OnReadUDPPacket) *UDPServer {
	return &UDPServer{
		addr:            addr,
		onReadUDPPacket: onReadUDPPacket,
	}
}

func (s *UDPServer) Listen() (err error) {
	udpAddr, err := net.ResolveUDPAddr("udp", s.addr)
	if err != nil {
		return
	}
	s.conn, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return
	}
	return
}

func (s *UDPServer) RunLoop() error {
	b := make([]byte, udpMaxPacketLength)
	for {
		l, a, e := s.conn.ReadFromUDP(b)
		if e != nil && (l < 0 || l > udpMaxPacketLength) {
			nazalog.Errorf("ReadFromUDP length invalid. length=%d", l)
		}
		s.onReadUDPPacket(b[:l], a.String(), e)
	}
}
