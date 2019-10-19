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
	"sync"

	log "github.com/q191201771/naza/pkg/nazalog"
)

type ServerObserver interface {
	NewRTMPPubSessionCB(session *ServerSession) bool // 返回true则允许推流，返回false则强制关闭这个连接
	NewRTMPSubSessionCB(session *ServerSession) bool // 返回true则允许拉流，返回false则强制关闭这个连接
	DelRTMPPubSessionCB(session *ServerSession)
	DelRTMPSubSessionCB(session *ServerSession)
}

type Server struct {
	obs  ServerObserver
	addr string
	m    sync.Mutex
	ln   net.Listener
}

func NewServer(obs ServerObserver, addr string) *Server {
	return &Server{
		obs:  obs,
		addr: addr,
	}
}

func (server *Server) RunLoop() error {
	var err error
	server.m.Lock()
	server.ln, err = net.Listen("tcp", server.addr)
	server.m.Unlock()
	if err != nil {
		return err
	}
	log.Infof("start rtmp server listen. addr=%s", server.addr)
	for {
		conn, err := server.ln.Accept()
		if err != nil {
			return err
		}
		go server.handleTCPConnect(conn)
	}
}

func (server *Server) Dispose() {
	server.m.Lock()
	defer server.m.Unlock()
	if server.ln == nil {
		return
	}
	if err := server.ln.Close(); err != nil {
		log.Error(err)
	}
}

func (server *Server) handleTCPConnect(conn net.Conn) {
	log.Infof("accept a rtmp connection. remoteAddr=%v", conn.RemoteAddr())
	session := NewServerSession(server, conn)
	_ = session.RunLoop()
	switch session.t {
	case ServerSessionTypeUnknown:
	// noop
	case ServerSessionTypePub:
		server.obs.DelRTMPPubSessionCB(session)
	case ServerSessionTypeSub:
		server.obs.DelRTMPSubSessionCB(session)
	}
}

// ServerSessionObserver
func (server *Server) NewRTMPPubSessionCB(session *ServerSession) {
	if !server.obs.NewRTMPPubSessionCB(session) {
		log.Warnf("dispose PubSession since pub exist.")
		session.Dispose()
		return
	}
}

// ServerSessionObserver
func (server *Server) NewRTMPSubSessionCB(session *ServerSession) {
	if !server.obs.NewRTMPSubSessionCB(session) {
		// TODO chef: 关闭这个连接
		return
	}
}
