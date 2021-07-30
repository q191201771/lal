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
	// @brief 使得上层有能力管理未进化到Pub、Sub阶段的Session
	OnNewRtspSessionConnect(session *ServerCommandSession)

	// @brief 注意，对于已经进化到了Pub、Sub阶段的Session，该回调依然会被调用
	OnDelRtspSession(session *ServerCommandSession)

	///////////////////////////////////////////////////////////////////////////

	// @brief  Announce阶段回调
	// @return 如果返回false，则表示上层要强制关闭这个推流请求
	OnNewRtspPubSession(session *PubSession) bool

	OnDelRtspPubSession(session *PubSession)

	///////////////////////////////////////////////////////////////////////////

	// @return 如果返回false，则表示上层要强制关闭这个拉流请求
	// @return sdp
	OnNewRtspSubSessionDescribe(session *SubSession) (ok bool, sdp []byte)

	// @brief Describe阶段回调
	// @return ok  如果返回false，则表示上层要强制关闭这个拉流请求
	OnNewRtspSubSessionPlay(session *SubSession) bool

	OnDelRtspSubSession(session *SubSession)
}

type Server struct {
	addr     string
	observer ServerObserver

	ln net.Listener
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
		go s.handleTcpConnect(conn)
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

// ServerCommandSessionObserver
func (s *Server) OnNewRtspPubSession(session *PubSession) bool {
	return s.observer.OnNewRtspPubSession(session)
}

// ServerCommandSessionObserver
func (s *Server) OnNewRtspSubSessionDescribe(session *SubSession) (ok bool, sdp []byte) {
	return s.observer.OnNewRtspSubSessionDescribe(session)
}

// ServerCommandSessionObserver
func (s *Server) OnNewRtspSubSessionPlay(session *SubSession) bool {
	return s.observer.OnNewRtspSubSessionPlay(session)
}

// ServerCommandSessionObserver
func (s *Server) OnDelRtspPubSession(session *PubSession) {
	s.observer.OnDelRtspPubSession(session)
}

// ServerCommandSessionObserver
func (s *Server) OnDelRtspSubSession(session *SubSession) {
	s.observer.OnDelRtspSubSession(session)
}

func (s *Server) handleTcpConnect(conn net.Conn) {
	session := NewServerCommandSession(s, conn)
	s.observer.OnNewRtspSessionConnect(session)

	err := session.RunLoop()
	nazalog.Info(err)

	if session.pubSession != nil {
		s.observer.OnDelRtspPubSession(session.pubSession)
		_ = session.pubSession.Dispose()
	} else if session.subSession != nil {
		s.observer.OnDelRtspSubSession(session.subSession)
		_ = session.subSession.Dispose()
	}
	s.observer.OnDelRtspSession(session)
}
