// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"sync"
	"time"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	log "github.com/q191201771/naza/pkg/nazalog"
)

type ServerManager struct {
	config *Config

	httpflvServer *httpflv.Server
	rtmpServer    *rtmp.Server
	exitChan      chan struct{}

	mutex    sync.Mutex
	groupMap map[string]*Group // TODO chef: with appName
}

func NewServerManager(config *Config) *ServerManager {
	m := &ServerManager{
		config:   config,
		groupMap: make(map[string]*Group),
		exitChan: make(chan struct{}),
	}
	if len(config.HTTPFLV.SubListenAddr) != 0 {
		m.httpflvServer = httpflv.NewServer(m, config.HTTPFLV.SubListenAddr)
	}
	if len(config.RTMP.Addr) != 0 {
		m.rtmpServer = rtmp.NewServer(m, config.RTMP.Addr)
	}
	return m
}

func (sm *ServerManager) RunLoop() {
	if sm.httpflvServer != nil {
		go func() {
			if err := sm.httpflvServer.RunLoop(); err != nil {
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
	if sm.httpflvServer != nil {
		sm.httpflvServer.Dispose()
	}
	if sm.rtmpServer != nil {
		sm.rtmpServer.Dispose()
	}

	sm.mutex.Lock()
	for _, group := range sm.groupMap {
		group.Dispose()
	}
	sm.mutex.Unlock()

	sm.exitChan <- struct{}{}
}

// ServerObserver of rtmp.Server
func (sm *ServerManager) NewRTMPPubSessionCB(session *rtmp.ServerSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName, session.StreamName)
	return group.AddRTMPPubSession(session)
}

// ServerObserver of rtmp.Server
func (sm *ServerManager) DelRTMPPubSessionCB(session *rtmp.ServerSession) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName, session.StreamName)
	if group != nil {
		group.DelRTMPPubSession(session)
	}
}

// ServerObserver of rtmp.Server
func (sm *ServerManager) NewRTMPSubSessionCB(session *rtmp.ServerSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName, session.StreamName)
	group.AddRTMPSubSession(session)
	return true
}

// ServerObserver of rtmp.Server
func (sm *ServerManager) DelRTMPSubSessionCB(session *rtmp.ServerSession) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName, session.StreamName)
	if group != nil {
		group.DelRTMPSubSession(session)
	}
}

// ServerObserver of httpflv.Server
func (sm *ServerManager) NewHTTPFLVSubSessionCB(session *httpflv.SubSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName, session.StreamName)
	group.AddHTTPFLVSubSession(session)
	return true
}

// ServerObserver of httpflv.Server
func (sm *ServerManager) DelHTTPFLVSubSessionCB(session *httpflv.SubSession) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName, session.StreamName)
	if group != nil {
		group.DelHTTPFLVSubSession(session)
	}
}

func (sm *ServerManager) check() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	for k, group := range sm.groupMap {
		if group.IsTotalEmpty() {
			log.Infof("erase empty group manager. [%s]", group.UniqueKey)
			group.Dispose()
			delete(sm.groupMap, k)
		}
	}
}

func (sm *ServerManager) getOrCreateGroup(appName string, streamName string) *Group {
	group, exist := sm.groupMap[streamName]
	if !exist {
		group = NewGroup(appName, streamName, sm.config.RTMP.GOPNum, sm.config.HTTPFLV.GOPNum)
		sm.groupMap[streamName] = group
	}
	go group.RunLoop()
	return group
}

func (sm *ServerManager) getGroup(appName string, streamName string) *Group {
	group, exist := sm.groupMap[streamName]
	if !exist {
		return nil
	}
	return group
}
