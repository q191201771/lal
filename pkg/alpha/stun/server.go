// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package stun

import (
	"net"

	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
)

type Server struct {
	conn *nazanet.UDPConnection
}

func NewServer(addr string) (*Server, error) {
	conn, err := nazanet.NewUDPConnection(func(option *nazanet.UDPConnectionOption) {
		option.LAddr = addr
	})
	if err != nil {
		return nil, err
	}
	return &Server{
		conn: conn,
	}, nil
}

func (s *Server) RunLoop() (err error) {
	return s.conn.RunLoop(s.onReadUDPPacket)
}

func (s *Server) Dispose() error {
	return s.conn.Dispose()
}

func (s *Server) onReadUDPPacket(b []byte, raddr *net.UDPAddr, err error) bool {
	if err != nil {
		return false
	}
	h, err := UnpackHeader(b)
	if err != nil {
		nazalog.Errorf("parse header failed. err=%+v", err)
		return false
	}
	if h.Typ != typeBindingRequestBE {
		nazalog.Errorf("type invalid. type=%d", h.Typ)
		return false
	}
	resp, err := PackBindingResponse(raddr.IP, raddr.Port)
	if err != nil {
		nazalog.Errorf("pack binding response failed. err=%+v", err)
		return false
	}
	if err := s.conn.Write2Addr(resp, raddr); err != nil {
		nazalog.Errorf("write failed. err=%+v", err)
		return false
	}

	return true
}
