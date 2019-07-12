package rtmp

import (
	"github.com/q191201771/lal/pkg/util/log"
	"net"
	"sync"
)

type ServerObserver interface {
	NewRTMPPubSessionCB(session *PubSession, group *Group) bool // 返回true则允许推流，返回false则强制关闭这个连接
	NewRTMPSubSessionCB(session *SubSession, group *Group) bool // 返回true则允许拉流，返回false则强制关闭这个连接
}

type Server struct {
	obs  ServerObserver
	addr string
	ln   net.Listener

	groupMap map[string]*Group
	mutex    sync.Mutex
}

func NewServer(obs ServerObserver, addr string) *Server {
	return &Server{
		obs:      obs,
		addr:     addr,
		groupMap: make(map[string]*Group),
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
	session := NewServerSession(server, conn)
	// TODO chef: 处理连接关闭
	session.RunLoop()
}

func (server *Server) getOrCreateGroup(appName string, streamName string) *Group {
	server.mutex.Lock()
	defer server.mutex.Unlock()

	group, exist := server.groupMap[streamName]
	if !exist {
		group = NewGroup(appName, streamName)
		server.groupMap[streamName] = group
	}
	go group.RunLoop()
	return group
}

// ServerSessionObserver
func (server *Server) NewRTMPPubSessionCB(session *PubSession) {
	group := server.getOrCreateGroup(session.AppName, session.StreamName)

	if !server.obs.NewRTMPPubSessionCB(session, group) {
		// TODO chef: 关闭这个连接
		return
	}
	group.AddPubSession(session)
}

// ServerSessionObserver
func (server *Server) NewRTMPSubSessionCB(session *SubSession) {
	group := server.getOrCreateGroup(session.AppName, session.StreamName)

	if !server.obs.NewRTMPSubSessionCB(session, group) {
		// TODO chef: 关闭这个连接
		return
	}
	group.AddSubSession(session)
}
