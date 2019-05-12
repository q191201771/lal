package httpflv

import (
	"github.com/q191201771/lal/log"
	"net"
)

type ServerObserver interface {
	NewHTTPFlvSubSessionCB(session *SubSession)
}

type Server struct {
	obs             ServerObserver
	addr            string
	subWriteTimeout int64

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
	server.ln, err = net.Listen("tcp", server.addr)
	if err != nil {
		return err
	}
	log.Infof("listen. addr=%s", server.addr)
	for {
		conn, err := server.ln.Accept()
		if err != nil {
			return err
		}
		go server.handleSubSessionConnect(conn)
	}
}

func (server *Server) Dispose() {
	if err := server.ln.Close(); err != nil {
		log.Error(err)
	}
}

func (server *Server) handleSubSessionConnect(conn net.Conn) {
	log.Infof("accept a http flv connection. remoteAddr=%v", conn.RemoteAddr())
	session := NewSubSession(conn, server.subWriteTimeout)
	if err := session.ReadRequest(); err != nil {
		log.Errorf("read SubSession request error. [%s]", session.UniqueKey)
		return
	}
	log.Infof("-----> http request. [%s] uri=%s", session.UniqueKey, session.URI)

	server.obs.NewHTTPFlvSubSessionCB(session)
}
