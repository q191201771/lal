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

	"github.com/q191201771/lal/pkg/hls"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
)

type ServerManager struct {
	rtmpServer    *rtmp.Server
	httpflvServer *httpflv.Server
	hlsServer     *hls.Server
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
	if config.HTTPFLVConfig.Enable {
		m.httpflvServer = httpflv.NewServer(m, config.HTTPFLVConfig.SubListenAddr)
	}
	if config.HLSConfig.Enable {
		m.hlsServer = hls.NewServer(config.HLSConfig.SubListenAddr, config.HLSConfig.OutPath)
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
		/*
		只在第1个ServerSession产生时启动这个group协程
		注：创建的group协程暂时做结构设计预留，现在并没有实际动作，以后可以用协程执行OnReadRTMPAVMsg中数据转发
		 */
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
