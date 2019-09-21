package rtmp

import (
	"github.com/q191201771/nezha/pkg/log"
	"net"
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
	server.ln, err = net.Listen("tcp", server.addr)
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
