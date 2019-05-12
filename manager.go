package main

import (
	"github.com/q191201771/lal/httpflv"
	"github.com/q191201771/lal/log"
	"sync"
	"time"
)

type Manager struct {
	config *Config

	server   *httpflv.Server
	groups   map[string]*Group // TODO chef: with appName
	mutex    sync.Mutex
	exitChan chan bool
}

func NewManager(config *Config) *Manager {
	m := &Manager{
		config:   config,
		groups:   make(map[string]*Group),
		exitChan: make(chan bool),
	}
	s := httpflv.NewServer(m, config.HTTPFlv.SubListenAddr, config.SubIdleTimeout)
	m.server = s
	return m
}

func (manager *Manager) RunLoop() {
	go func() {
		if err := manager.server.RunLoop(); err != nil {
			log.Error(err)
		}
	}()

	t := time.NewTicker(1 * time.Second)
	defer t.Stop()
	// TODO chef: erase me, just for debug
	tmpT := time.NewTicker(10 * time.Second)
	defer tmpT.Stop()
	for {
		select {
		case <-manager.exitChan:
			return
		case <-t.C:
			manager.check()
		case <-tmpT.C:
			// TODO chef: lock
			log.Infof("group size:%d", len(manager.groups))
		}
	}
}

func (manager *Manager) Dispose() {
	log.Debug("Dispose manager.")
	manager.server.Dispose()
	manager.exitChan <- true
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	for _, group := range manager.groups {
		group.Dispose(lalErr)
	}
	manager.groups = nil
}

func (manager *Manager) NewHTTPFlvSubSessionCB(session *httpflv.SubSession) {
	group := manager.getOrCreateGroup(session.AppName, session.StreamName)
	group.AddSubSession(session)
	switch manager.config.Pull.Type {
	case "httpflv":
		group.PullIfNeeded(manager.config.Pull.Addr)
	default:
		log.Errorf("unknown pull type. type=%s", manager.config.Pull.Type)
	}
}

func (manager *Manager) check() {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	for k, group := range manager.groups {
		if group.IsTotalEmpty() {
			log.Infof("erase empty group. [%s]", group.UniqueKey)
			group.Dispose(lalErr)
			delete(manager.groups, k)
		}
	}
}

func (manager *Manager) getOrCreateGroup(appName string, streamName string) *Group {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	group, exist := manager.groups[streamName]
	if !exist {
		group = NewGroup(appName, streamName, manager.config)
		manager.groups[streamName] = group
	}
	go group.RunLoop()
	return group
}
