package httpflv

import (
	log "github.com/q191201771/naza/pkg/nazalog"
	"net"
	"sync"
)

type ServerObserver interface {
	// 通知上层有新的拉流者
	// 返回值： true则允许拉流，false则关闭连接
	NewHTTPFlvSubSessionCB(session *SubSession) bool
}

type Server struct {
	obs             ServerObserver
	addr            string
	subWriteTimeout int64

	m  sync.Mutex
	ln net.Listener
}

func NewServer(obs ServerObserver, addr string, subWriteTimeout int64) *Server {
	return &Server{
		obs:             obs,
		addr:            addr,
		subWriteTimeout: subWriteTimeout,
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
	log.Infof("start httpflv listen. addr=%s", server.addr)
	for {
		conn, err := server.ln.Accept()
		if err != nil {
			return err
		}
		go server.handleConnect(conn)
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

func (server *Server) handleConnect(conn net.Conn) {
	log.Infof("accept a http flv connection. remoteAddr=%v", conn.RemoteAddr())
	session := NewSubSession(conn, server.subWriteTimeout)
	if err := session.ReadRequest(); err != nil {
		log.Errorf("read SubSession request error. [%s]", session.UniqueKey)
		return
	}
	log.Infof("-----> http request. [%s] uri=%s", session.UniqueKey, session.URI)

	if !server.obs.NewHTTPFlvSubSessionCB(session) {
		session.Dispose(httpFlvErr)
	}
}
