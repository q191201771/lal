package httpflv

import (
	"github.com/q191201771/lal/log"
	"sync"
	"time"
)

type Manager struct {
	Config

	server   *Server
	groups   map[string]*Group // TODO chef: with appName
	mutex    sync.Mutex
	exitChan chan bool
}

func NewManager(config Config) *Manager {
	return &Manager{
		Config:   config,
		server:   NewServer(config.SubListenAddr),
		groups:   make(map[string]*Group),
		exitChan: make(chan bool),
	}
}

func (manager *Manager) subSessionHandler(session *SubSession) {
	group := manager.getOrCreateGroup(session.AppName, session.StreamName)
	group.AddSubSession(session)
	if manager.PullAddr != "" {
		group.PullIfNeeded(manager.PullAddr)
	}
}

func (manager *Manager) RunLoop() {
	go func() {
		for {
			if subSession, ok := manager.server.Accept(); ok {
				manager.subSessionHandler(subSession)
			} else {
				log.Error("accept sub session failed.")
				return
			}
		}
	}()

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
		group.Dispose(fxxkErr)
	}
	manager.groups = nil
}

func (manager *Manager) check() {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	for k, group := range manager.groups {
		if group.IsTotalEmpty() {
			log.Infof("erase empty group. [%s]", group.UniqueKey)
			group.Dispose(fxxkErr)
			delete(manager.groups, k)
		}
	}
}

func (manager *Manager) getOrCreateGroup(appName string, streamName string) *Group {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	group, exist := manager.groups[streamName]
	if !exist {
		group = NewGroup(appName, streamName, manager.Config)
		manager.groups[streamName] = group
	}
	go group.RunLoop()
	return group
}
