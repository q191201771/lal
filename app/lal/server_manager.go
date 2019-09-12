package main

import (
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/nezha/pkg/log"
	"sync"
	"time"
)

type ServerManager struct {
	config *Config

	httpFlvServer   *httpflv.Server
	rtmpServer      *rtmp.Server
	groupManagerMap map[string]*GroupManager // TODO chef: with appName
	mutex           sync.Mutex
	exitChan        chan bool
}

func NewServerManager(config *Config) *ServerManager {
	m := &ServerManager{
		config:          config,
		groupManagerMap: make(map[string]*GroupManager),
		exitChan:        make(chan bool),
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
				log.Infof("group size:%d", len(sm.groupManagerMap))
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
	for _, gm := range sm.groupManagerMap {
		gm.Dispose(lalErr)
	}
	sm.groupManagerMap = nil
	sm.mutex.Unlock()

	sm.exitChan <- true
}

// ServerObserver of rtmp.Server
func (sm *ServerManager) NewRTMPPubSessionCB(session *rtmp.ServerSession, rtmpGroup *rtmp.Group) bool {
	gm := sm.getOrCreateGroupManager(session.AppName, session.StreamName)
	return gm.AddRTMPPubSession(session, rtmpGroup)
}

// ServerObserver of rtmp.Server
func (sm *ServerManager) NewRTMPSubSessionCB(session *rtmp.ServerSession, rtmpGroup *rtmp.Group) bool {
	gm := sm.getOrCreateGroupManager(session.AppName, session.StreamName)
	gm.AddRTMPSubSession(session, rtmpGroup)
	return true
}

// ServerObserver of httpflv.Server
func (sm *ServerManager) NewHTTPFlvSubSessionCB(session *httpflv.SubSession, httpFlvGroup *httpflv.Group) bool {
	gm := sm.getOrCreateGroupManager(session.AppName, session.StreamName)
	gm.AddHTTPFlvSubSession(session, httpFlvGroup)
	return true
}

func (sm *ServerManager) check() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	for k, gm := range sm.groupManagerMap {
		if gm.IsTotalEmpty() {
			log.Infof("erase empty group manager. [%s]", gm.UniqueKey)
			gm.Dispose(lalErr)
			delete(sm.groupManagerMap, k)
		}
	}
}

func (sm *ServerManager) getOrCreateGroupManager(appName string, streamName string) *GroupManager {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	gm, exist := sm.groupManagerMap[streamName]
	if !exist {
		gm = NewGroupManager(appName, streamName, sm.config)
		sm.groupManagerMap[streamName] = gm
	}
	go gm.RunLoop()
	return gm
}
