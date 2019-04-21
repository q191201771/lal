package httpflv

import (
	"log"
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
	for {
		conn, err := server.ln.Accept()
		if err != nil {
			log.Println(err)
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
		log.Println(err)
	}
	close(server.sessionChan)
}

func (server *Server) handleSubSessionConnect(conn net.Conn) {
	log.Println("accept a http flv sub connection. ", conn.RemoteAddr())
	subSession := NewSubSession(conn)
	err := subSession.ReadRequest()
	if err != nil {
		return
	}
	log.Println("read sub session request. ", subSession.StreamName)

	server.sessionChan <- subSession
}
