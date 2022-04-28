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
)

type IServerObserver interface {
	OnRtmpConnect(session *ServerSession, opa ObjectPairArray)

	// OnNewRtmpPubSession
	//
	// 上层代码应该在这个事件回调中注册音视频数据的监听
	//
	// @return 上层如果想关闭这个session，则回调中返回不为nil的error值
	//
	OnNewRtmpPubSession(session *ServerSession) error

	// OnDelRtmpPubSession
	//
	// 注意，如果session是上层通过 OnNewRtmpPubSession 回调的返回值关闭的，则该session不再触发这个逻辑
	//
	OnDelRtmpPubSession(session *ServerSession)

	OnNewRtmpSubSession(session *ServerSession) error
	OnDelRtmpSubSession(session *ServerSession)
}

type Server struct {
	addr     string
	observer IServerObserver
	ln       net.Listener
}

func NewServer(addr string, observer IServerObserver) *Server {
	return &Server{
		addr:     addr,
		observer: observer,
	}
}

func (server *Server) Listen() (err error) {
	if server.ln, err = net.Listen("tcp", server.addr); err != nil {
		return
	}
	Log.Infof("start rtmp server listen. addr=%s", server.addr)
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
		Log.Error(err)
	}
}

func (server *Server) handleTcpConnect(conn net.Conn) {
	Log.Infof("accept a rtmp connection. remoteAddr=%s", conn.RemoteAddr().String())
	session := NewServerSession(server, conn)
	_ = session.RunLoop()

	if session.DisposeByObserverFlag {
		return
	}
	switch session.t {
	case ServerSessionTypeUnknown:
	// noop
	case ServerSessionTypePub:
		server.observer.OnDelRtmpPubSession(session)
	case ServerSessionTypeSub:
		server.observer.OnDelRtmpSubSession(session)
	}
}

// ----- IServerSessionObserver ------------------------------------------------------------------------------------

func (server *Server) OnRtmpConnect(session *ServerSession, opa ObjectPairArray) {
	server.observer.OnRtmpConnect(session, opa)
}

func (server *Server) OnNewRtmpPubSession(session *ServerSession) error {
	return server.observer.OnNewRtmpPubSession(session)
}

func (server *Server) OnNewRtmpSubSession(session *ServerSession) error {
	return server.observer.OnNewRtmpSubSession(session)
}
