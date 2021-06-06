// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import (
	"net"

	log "github.com/q191201771/naza/pkg/nazalog"
)

type ServerObserver interface {
	OnRtmpConnect(session *ServerSession, opa ObjectPairArray)
	OnNewRtmpPubSession(session *ServerSession) bool // 返回true则允许推流，返回false则强制关闭这个连接
	OnDelRtmpPubSession(session *ServerSession)
	OnNewRtmpSubSession(session *ServerSession) bool // 返回true则允许拉流，返回false则强制关闭这个连接
	OnDelRtmpSubSession(session *ServerSession)
}

type Server struct {
	observer ServerObserver
	addr     string
	ln       net.Listener
}

func NewServer(observer ServerObserver, addr string) *Server {
	return &Server{
		observer: observer,
		addr:     addr,
	}
}

func (server *Server) Listen() (err error) {
	if server.ln, err = net.Listen("tcp", server.addr); err != nil {
		return
	}
	log.Infof("start rtmp server listen. addr=%s", server.addr)
	return
}

func (server *Server) RunLoop() error {
	for {
		conn, err := server.ln.Accept()
		if err != nil {
			return err
		}
		go server.handleTcpConnect(conn)
	}
}

func (server *Server) Dispose() {
	if server.ln == nil {
		return
	}
	if err := server.ln.Close(); err != nil {
		log.Error(err)
	}
}

func (server *Server) handleTcpConnect(conn net.Conn) {
	log.Infof("accept a rtmp connection. remoteAddr=%s", conn.RemoteAddr().String())
	session := NewServerSession(server, conn)
	err := session.RunLoop()
	log.Infof("[%s] rtmp loop done. err=%v", session.uniqueKey, err)
	switch session.t {
	case ServerSessionTypeUnknown:
	// noop
	case ServerSessionTypePub:
		server.observer.OnDelRtmpPubSession(session)
	case ServerSessionTypeSub:
		server.observer.OnDelRtmpSubSession(session)
	}
}

// ServerSessionObserver
func (server *Server) OnRtmpConnect(session *ServerSession, opa ObjectPairArray) {
	server.observer.OnRtmpConnect(session, opa)
}

// ServerSessionObserver
func (server *Server) OnNewRtmpPubSession(session *ServerSession) {
	if !server.observer.OnNewRtmpPubSession(session) {
		log.Warnf("dispose PubSession since pub exist.")
		session.Dispose()
		return
	}
}

// ServerSessionObserver
func (server *Server) OnNewRtmpSubSession(session *ServerSession) {
	if !server.observer.OnNewRtmpSubSession(session) {
		session.Dispose()
		return
	}
}
