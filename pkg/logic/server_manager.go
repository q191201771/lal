package logic

import (
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/nezha/pkg/log"
	"sync"
	"time"
)

type ServerManager struct {
	config *Config

	httpFlvServer *httpflv.Server
	rtmpServer    *rtmp.Server
	groupMap      map[string]*Group // TODO chef: with appName
	mutex         sync.Mutex
	exitChan      chan struct{}
}

var _ rtmp.ServerObserver = &ServerManager{}

func NewServerManager(config *Config) *ServerManager {
	m := &ServerManager{
		config:   config,
		groupMap: make(map[string]*Group),
		exitChan: make(chan struct{}),
	}
	if len(config.HTTPFlv.SubListenAddr) != 0 {
		m.httpFlvServer = httpflv.NewServer(m, config.HTTPFlv.SubListenAddr, config.SubIdleTimeout)
	}
	if len(config.RTMP.Addr) != 0 {
		m.rtmpServer = rtmp.NewServer(m, config.RTMP.Addr)
	}
	return m
}

func (sm *ServerManager) RunLoop() {
	if sm.httpFlvServer != nil {
		go func() {
			if err := sm.httpFlvServer.RunLoop(); err != nil {
				log.Error(err)
			}
		}()
	}

	if sm.rtmpServer != nil {
		go func() {
			if err := sm.rtmpServer.RunLoop(); err != nil {
				log.Error(err)
			}
		}()
	}

	t := time.NewTicker(1 * time.Second)
	defer t.Stop()
	var count uint32
	for {
		select {
		case <-sm.exitChan:
			return
		case <-t.C:
			sm.check()
			count++
			if (count % 10) == 0 {
				sm.mutex.Lock()
				log.Infof("group size:%d", len(sm.groupMap))
				sm.mutex.Unlock()
			}
		}
	}
}

func (sm *ServerManager) Dispose() {
	log.Debug("dispose server manager.")
	if sm.httpFlvServer != nil {
		sm.httpFlvServer.Dispose()
	}
	if sm.rtmpServer != nil {
		sm.rtmpServer.Dispose()
	}

	sm.mutex.Lock()
	for _, group := range sm.groupMap {
		group.Dispose(lalErr)
	}
	sm.groupMap = nil
	sm.mutex.Unlock()

	sm.exitChan <- struct{}{}
}

// ServerObserver of rtmp.Server
func (sm *ServerManager) NewRTMPPubSessionCB(session *rtmp.ServerSession) bool {
	group := sm.getOrCreateGroup(session.AppName, session.StreamName)
	return group.AddRTMPPubSession(session)
}

// ServerObserver of rtmp.Server
func (sm *ServerManager) NewRTMPSubSessionCB(session *rtmp.ServerSession) bool {
	group := sm.getOrCreateGroup(session.AppName, session.StreamName)
	group.AddRTMPSubSession(session)
	return true
}

// ServerObserver of rtmp.Server
func (sm *ServerManager) DelRTMPPubSessionCB(session *rtmp.ServerSession) {
	group := sm.getOrCreateGroup(session.AppName, session.StreamName)
	group.DelRTMPPubSession(session)
}

// ServerObserver of rtmp.Server
func (sm *ServerManager) DelRTMPSubSessionCB(session *rtmp.ServerSession) {
	group := sm.getOrCreateGroup(session.AppName, session.StreamName)
	group.DelRTMPSubSession(session)
}

// ServerObserver of httpflv.Server
func (sm *ServerManager) NewHTTPFlvSubSessionCB(session *httpflv.SubSession, httpFlvGroup *httpflv.Group) bool {
	group := sm.getOrCreateGroup(session.AppName, session.StreamName)
	group.AddHTTPFlvSubSession(session, httpFlvGroup)
	return true
}

func (sm *ServerManager) check() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	for k, group := range sm.groupMap {
		if group.IsTotalEmpty() {
			log.Infof("erase empty group manager. [%s]", group.UniqueKey)
			group.Dispose(lalErr)
			delete(sm.groupMap, k)
		}
	}
}

func (sm *ServerManager) getOrCreateGroup(appName string, streamName string) *Group {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group, exist := sm.groupMap[streamName]
	if !exist {
		group = NewGroup(appName, streamName)
		sm.groupMap[streamName] = group
	}
	go group.RunLoop()
	return group
}
