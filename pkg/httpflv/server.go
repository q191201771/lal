// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv

import (
	"net"

	log "github.com/q191201771/naza/pkg/nazalog"
)

type ServerObserver interface {
	// 通知上层有新的拉流者
	// 返回值： true则允许拉流，false则关闭连接
	OnNewHTTPFLVSubSession(session *SubSession) bool

	OnDelHTTPFLVSubSession(session *SubSession)
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
	log.Infof("start httpflv server listen. addr=%s", server.addr)
	return
}

func (server *Server) RunLoop() error {
	for {
		conn, err := server.ln.Accept()
		if err != nil {
			return err
		}
		go server.handleConnect(conn)
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

func (server *Server) handleConnect(conn net.Conn) {
	log.Infof("accept a httpflv connection. remoteAddr=%s", conn.RemoteAddr().String())
	session := NewSubSession(conn)
	if err := session.ReadRequest(); err != nil {
		log.Errorf("[%s] read httpflv SubSession request error. err=%v", session.UniqueKey, err)
		return
	}
	log.Debugf("[%s] < read http request. uri=%s", session.UniqueKey, session.URI)

	if !server.obs.OnNewHTTPFLVSubSession(session) {
		session.Dispose()
	}

	err := session.RunLoop()
	log.Debugf("[%s] httpflv sub session loop done. err=%v", session.UniqueKey, err)
	server.obs.OnDelHTTPFLVSubSession(session)
}
