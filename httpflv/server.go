package httpflv

import (
	"github.com/q191201771/lal/log"
	"net"
)

type Server struct {
	addr        string
	sessionChan chan *SubSession

	ln net.Listener
}

func NewServer(addr string) *Server {
	return &Server{
		addr:        addr,
		sessionChan: make(chan *SubSession, 8),
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

func (server *Server) Accept() (session *SubSession, ok bool) {
	session, ok = <-server.sessionChan
	return
}

func (server *Server) Dispose() {
	if err := server.ln.Close(); err != nil {
		log.Error(err)
	}
	close(server.sessionChan)
}

func (server *Server) handleSubSessionConnect(conn net.Conn) {
	log.Infof("accept a http flv connection. remoteAddr=%v", conn.RemoteAddr())
	subSession := NewSubSession(conn)
	if err := subSession.ReadRequest(); err != nil {
		log.Errorf("read SubSession request error. [%s]", subSession.UniqueKey)
		return
	}
	log.Infof("-----> http request. [%s] uri=%s", subSession.UniqueKey, subSession.Uri)

	server.sessionChan <- subSession
}
