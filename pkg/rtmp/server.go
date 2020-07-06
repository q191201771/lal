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
	OnNewRTMPPubSession(session *ServerSession) bool // 返回true则允许推流，返回false则强制关闭这个连接
	OnDelRTMPPubSession(session *ServerSession)
	OnNewRTMPSubSession(session *ServerSession) bool // 返回true则允许拉流，返回false则强制关闭这个连接
	OnDelRTMPSubSession(session *ServerSession)
}

type Server struct {
	obs  ServerObserver
	addr string
	ln   net.Listener
}

func NewServer(obs ServerObserver, addr string) *Server {
	return &Server{
		obs:  obs,
		addr: addr,
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
		go server.handleTCPConnect(conn)
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

func (server *Server) handleTCPConnect(conn net.Conn) {
	log.Infof("accept a rtmp connection. remoteAddr=%s", conn.RemoteAddr().String())
	session := NewServerSession(server, conn)
	err := session.RunLoop()
	log.Infof("[%s] rtmp loop done. err=%v", session.UniqueKey, err)
	switch session.t {
	case ServerSessionTypeUnknown:
	// noop
	case ServerSessionTypePub:
		server.obs.OnDelRTMPPubSession(session)
	case ServerSessionTypeSub:
		server.obs.OnDelRTMPSubSession(session)
	}
}

// ServerSessionObserver
func (server *Server) OnNewRTMPPubSession(session *ServerSession) {
	if !server.obs.OnNewRTMPPubSession(session) {
		log.Warnf("dispose PubSession since pub exist.")
		session.Dispose()
		return
	}
}

// ServerSessionObserver
func (server *Server) OnNewRTMPSubSession(session *ServerSession) {
	if !server.obs.OnNewRTMPSubSession(session) {
		session.Dispose()
		return
	}
}
