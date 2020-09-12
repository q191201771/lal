// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"os"
	"sync"
	"time"

	"github.com/q191201771/lal/pkg/httpts"

	"github.com/q191201771/lal/pkg/rtsp"

	"github.com/q191201771/lal/pkg/hls"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
)

type ServerManager struct {
	rtmpServer    *rtmp.Server
	httpflvServer *httpflv.Server
	hlsServer     *hls.Server
	httptsServer  *httpts.Server
	rtspServer    *rtsp.Server
	exitChan      chan struct{}

	mutex    sync.Mutex
	groupMap map[string]*Group // TODO chef: with appName
}

func NewServerManager() *ServerManager {
	m := &ServerManager{
		groupMap: make(map[string]*Group),
		exitChan: make(chan struct{}),
	}
	if config.RTMPConfig.Enable {
		m.rtmpServer = rtmp.NewServer(m, config.RTMPConfig.Addr)
	}
	if config.HTTPFLVConfig.Enable || config.HTTPFLVConfig.EnableHTTPS {
		m.httpflvServer = httpflv.NewServer(m, config.HTTPFLVConfig.ServerConfig)
	}
	if config.HLSConfig.Enable {
		m.hlsServer = hls.NewServer(config.HLSConfig.SubListenAddr, config.HLSConfig.OutPath)
	}
	if config.HTTPTSConfig.Enable {
		m.httptsServer = httpts.NewServer(m, config.HTTPTSConfig.SubListenAddr)
	}
	if config.RTSPConfig.Enable {
		m.rtspServer = rtsp.NewServer(config.RTSPConfig.Addr, m)
	}
	return m
}

func (sm *ServerManager) RunLoop() {
	if sm.rtmpServer != nil {
		if err := sm.rtmpServer.Listen(); err != nil {
			nazalog.Error(err)
			os.Exit(1)
		}
		go func() {
			if err := sm.rtmpServer.RunLoop(); err != nil {
				nazalog.Error(err)
			}
		}()
	}

	if sm.httpflvServer != nil {
		if err := sm.httpflvServer.Listen(); err != nil {
			nazalog.Error(err)
			os.Exit(1)
		}
		go func() {
			if err := sm.httpflvServer.RunLoop(); err != nil {
				nazalog.Error(err)
			}
		}()
	}

	if sm.httptsServer != nil {
		if err := sm.httptsServer.Listen(); err != nil {
			nazalog.Error(err)
			os.Exit(1)
		}
		go func() {
			if err := sm.httptsServer.RunLoop(); err != nil {
				nazalog.Error(err)
			}
		}()
	}

	if sm.hlsServer != nil {
		if err := sm.hlsServer.Listen(); err != nil {
			nazalog.Error(err)
			os.Exit(1)
		}
		go func() {
			if err := sm.hlsServer.RunLoop(); err != nil {
				nazalog.Error(err)
			}
		}()
	}

	if sm.rtspServer != nil {
		if err := sm.rtspServer.Listen(); err != nil {
			nazalog.Error(err)
			os.Exit(1)
		}
		go func() {
			if err := sm.rtspServer.RunLoop(); err != nil {
				nazalog.Error(err)
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
			sm.iterateGroup()
			count++
			if (count % 10) == 0 {
				sm.mutex.Lock()
				nazalog.Debugf("group size=%d", len(sm.groupMap))
				for _, g := range sm.groupMap {
					nazalog.Debugf("%s", g.StringifyStats())
				}
				sm.mutex.Unlock()
			}
		}
	}
}

func (sm *ServerManager) Dispose() {
	nazalog.Debug("dispose server manager.")
	if sm.rtmpServer != nil {
		sm.rtmpServer.Dispose()
	}
	if sm.httpflvServer != nil {
		sm.httpflvServer.Dispose()
	}
	if sm.httptsServer != nil {
		sm.httptsServer.Dispose()
	}
	if sm.hlsServer != nil {
		sm.hlsServer.Dispose()
	}

	sm.mutex.Lock()
	for _, group := range sm.groupMap {
		group.Dispose()
	}
	sm.mutex.Unlock()

	sm.exitChan <- struct{}{}
}

// ServerObserver of rtmp.Server
func (sm *ServerManager) OnNewRTMPPubSession(session *rtmp.ServerSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName, session.StreamName)
	return group.AddRTMPPubSession(session)
}

// ServerObserver of rtmp.Server
func (sm *ServerManager) OnDelRTMPPubSession(session *rtmp.ServerSession) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName, session.StreamName)
	if group != nil {
		group.DelRTMPPubSession(session)
	}
}

// ServerObserver of rtmp.Server
func (sm *ServerManager) OnNewRTMPSubSession(session *rtmp.ServerSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName, session.StreamName)
	group.AddRTMPSubSession(session)
	return true
}

// ServerObserver of rtmp.Server
func (sm *ServerManager) OnDelRTMPSubSession(session *rtmp.ServerSession) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName, session.StreamName)
	if group != nil {
		group.DelRTMPSubSession(session)
	}
}

// ServerObserver of httpflv.Server
func (sm *ServerManager) OnNewHTTPFLVSubSession(session *httpflv.SubSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName, session.StreamName)
	group.AddHTTPFLVSubSession(session)
	return true
}

// ServerObserver of httpflv.Server
func (sm *ServerManager) OnDelHTTPFLVSubSession(session *httpflv.SubSession) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName, session.StreamName)
	if group != nil {
		group.DelHTTPFLVSubSession(session)
	}
}

// ServerObserver of httpts.Server
func (sm *ServerManager) OnNewHTTPTSSubSession(session *httpts.SubSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName, session.StreamName)
	group.AddHTTPTSSubSession(session)
	return true
}

// ServerObserver of httpts.Server
func (sm *ServerManager) OnDelHTTPTSSubSession(session *httpts.SubSession) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName, session.StreamName)
	if group != nil {
		group.DelHTTPTSSubSession(session)
	}
}

// ServerObserver of rtsp.Server
func (sm *ServerManager) OnNewRTSPPubSession(session *rtsp.PubSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup("", session.StreamName)
	return group.AddRTSPPubSession(session)
}

// ServerObserver of rtsp.Server
func (sm *ServerManager) OnDelRTSPPubSession(session *rtsp.PubSession) {
	// TODO chef: impl me
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup("", session.StreamName)
	if group != nil {
		group.DelRTSPPubSession(session)
	}
}

func (sm *ServerManager) iterateGroup() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	for k, group := range sm.groupMap {
		if group.IsTotalEmpty() {
			nazalog.Infof("erase empty group manager. [%s]", group.UniqueKey)
			group.Dispose()
			delete(sm.groupMap, k)
			continue
		}

		group.Tick()
	}
}

func (sm *ServerManager) getOrCreateGroup(appName string, streamName string) *Group {
	group, exist := sm.groupMap[streamName]
	if !exist {
		group = NewGroup(appName, streamName)
		sm.groupMap[streamName] = group

		go group.RunLoop()
	}
	return group
}

func (sm *ServerManager) getGroup(appName string, streamName string) *Group {
	group, exist := sm.groupMap[streamName]
	if !exist {
		return nil
	}
	return group
}
