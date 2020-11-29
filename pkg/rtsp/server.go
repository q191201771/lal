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

type ServerObserver interface {
	// @brief  Announce阶段回调
	// @return 如果返回false，则表示上层要强制关闭这个推流请求
	OnNewRTSPPubSession(session *PubSession) bool

	OnDelRTSPPubSession(session *PubSession)

	// @return 如果返回false，则表示上层要强制关闭这个拉流请求
	// @return sdp
	OnNewRTSPSubSessionDescribe(session *SubSession) (ok bool, sdp []byte)

	// @brief Describe阶段回调
	// @return ok  如果返回false，则表示上层要强制关闭这个拉流请求
	OnNewRTSPSubSessionPlay(session *SubSession) bool

	OnDelRTSPSubSession(session *SubSession)
}

type Server struct {
	addr     string
	observer ServerObserver

	ln net.Listener

	// TODO chef:
	// 在server中管理还没有被上层接管的session
}

func NewServer(addr string, observer ServerObserver) *Server {
	return &Server{
		addr:     addr,
		observer: observer,
	}
}

func (s *Server) Listen() (err error) {
	s.ln, err = net.Listen("tcp", s.addr)
	if err != nil {
		return
	}
	nazalog.Infof("start rtsp server listen. addr=%s", s.addr)
	return
}

func (s *Server) RunLoop() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return err
		}
		go s.handleTCPConnect(conn)
	}
}

func (s *Server) Dispose() {
	if s.ln == nil {
		return
	}
	if err := s.ln.Close(); err != nil {
		nazalog.Error(err)
	}
}

// ServerBaseSessionObserver
func (s *Server) OnNewRTSPPubSession(session *PubSession) bool {
	return s.observer.OnNewRTSPPubSession(session)
}

// ServerBaseSessionObserver
func (s *Server) OnNewRTSPSubSessionDescribe(session *SubSession) (ok bool, sdp []byte) {
	return s.observer.OnNewRTSPSubSessionDescribe(session)
}

// ServerBaseSessionObserver
func (s *Server) OnNewRTSPSubSessionPlay(session *SubSession) bool {
	return s.observer.OnNewRTSPSubSessionPlay(session)
}

// ServerBaseSessionObserver
func (s *Server) OnDelRTSPPubSession(session *PubSession) {
	s.observer.OnDelRTSPPubSession(session)
}

// ServerBaseSessionObserver
func (s *Server) OnDelRTSPSubSession(session *SubSession) {
	s.observer.OnDelRTSPSubSession(session)
}

func (s *Server) handleTCPConnect(conn net.Conn) {
	session := NewServerBaseSession(s, conn)
	err := session.RunLoop()
	nazalog.Info(err)
}
