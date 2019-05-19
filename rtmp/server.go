package rtmp

import (
	"github.com/q191201771/lal/log"
	"net"
)

type Server struct {
	addr string
	ln   net.Listener
}

func NewServer(addr string) *Server {
	return &Server{
		addr: addr,
	}
}

func (server *Server) RunLoop() error {
	var err error
	server.ln, err = net.Listen("tcp", server.addr)
	if err != nil {
		return err
	}
	log.Infof("start rtmp listen. addr=%s", server.addr)
	for {
		conn, err := server.ln.Accept()
		if err != nil {
			return err
		}
		go server.handleConnect(conn)
	}
}

func (server *Server) Dispose() {
	if err := server.ln.Close(); err != nil {
		log.Error(err)
	}
}

func (server *Server) handleConnect(conn net.Conn) {
	log.Infof("accept a rtmp connection. remoteAddr=%v", conn.RemoteAddr())
	session := NewServerSession(conn)
	session.RunLoop()
}
