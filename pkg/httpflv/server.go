package httpflv

import (
	"github.com/q191201771/nezha/pkg/log"
	"net"
	"sync"
)

type ServerObserver interface {
	// 通知上层有新的拉流者
	// 返回值： true则允许拉流，false则关闭连接
	NewHTTPFlvSubSessionCB(session *SubSession, group *Group) bool
}

type Server struct {
	obs             ServerObserver
	addr            string
	subWriteTimeout int64

	ln net.Listener

	groupMap map[string]*Group
	mutex    sync.Mutex
}

func NewServer(obs ServerObserver, addr string, subWriteTimeout int64) *Server {
	return &Server{
		obs:             obs,
		addr:            addr,
		subWriteTimeout: subWriteTimeout,
		groupMap:        make(map[string]*Group),
	}
}

func (server *Server) RunLoop() error {
	var err error
	server.ln, err = net.Listen("tcp", server.addr)
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
	if err := server.ln.Close(); err != nil {
		log.Error(err)
	}
}

func (server *Server) GetOrCreateGroup(appName string, streamName string) *Group {
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

func (server *Server) handleConnect(conn net.Conn) {
	log.Infof("accept a http flv connection. remoteAddr=%v", conn.RemoteAddr())
	session := NewSubSession(conn, server.subWriteTimeout)
	if err := session.ReadRequest(); err != nil {
		log.Errorf("read SubSession request error. [%s]", session.UniqueKey)
		return
	}
	log.Infof("-----> http request. [%s] uri=%s", session.UniqueKey, session.URI)

	group := server.GetOrCreateGroup(session.AppName, session.StreamName)
	group.AddHTTPFlvSubSession(session)

	if !server.obs.NewHTTPFlvSubSessionCB(session, group) {
		session.Dispose(httpFlvErr)
	}
}
