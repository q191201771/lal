package httpflv

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type Group struct {
	Config
	appName    string
	streamName string

	exitChan chan bool

	pullSession     *PullSession
	subSessionList  map[*SubSession]bool
	turnToEmptyTick int64 // trace while sub session list turn to empty
	mutex           sync.Mutex
}

func NewGroup(appName string, streamName string, config Config) *Group {
	return &Group{
		Config:         config,
		appName:        appName,
		streamName:     streamName,
		exitChan:       make(chan bool),
		subSessionList: make(map[*SubSession]bool),
	}
}

func (group *Group) RunLoop() {
	count := int64(0)
	t := time.NewTicker(1 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-group.exitChan:
			return
		case <-t.C:
			count++
			now := time.Now().Unix()

			if group.SubIdleTimeout != 0 && count%group.SubIdleTimeout == 0 {
				group.mutex.Lock()
				for sub := range group.subSessionList {
					if now-sub.StartTick < group.SubIdleTimeout {
						continue
					}
					if _, diff := sub.GetStat(); diff.writeByte == 0 {
						log.Println("sub idle timeout.", sub.StreamName)
						sub.ForceClose()
						delete(group.subSessionList, sub)
					}
				}
				group.mutex.Unlock()
			}

			if group.PullReadTimeout != 0 && count%group.PullReadTimeout == 0 {
				group.mutex.Lock()
				if group.pullSession != nil {
					if now-group.pullSession.StartTick > group.PullReadTimeout {
						if _, diff := group.pullSession.GetStat(); diff.readByte == 0 {
							log.Println("pull read timeout.")
							group.pullSession.ForceClose()
							group.pullSession = nil
						}
					}
				}
				group.mutex.Unlock()
			}

			if group.StopPullWhileNoSubTimeout != 0 && count%group.StopPullWhileNoSubTimeout == 0 {
				group.mutex.Lock()
				if group.pullSession != nil && group.turnToEmptyTick != 0 && len(group.subSessionList) == 0 &&
					now-group.turnToEmptyTick > group.StopPullWhileNoSubTimeout {

					log.Println("stop pull while no sub.")
					group.pullSession.ForceClose()
					group.pullSession = nil
				}
				group.mutex.Unlock()
			}
		}
	}
}

func (group *Group) Dispose() {
	group.exitChan <- true
}

func (group *Group) AddSubSession(session *SubSession) {
	group.mutex.Lock()
	log.Println("add sub session in group.")
	group.subSessionList[session] = true
	group.turnToEmptyTick = 0
	group.mutex.Unlock()

	session.writeHttpResponseHeader()
	session.writeFlvHeader()
	go func() {
		if err := session.RunLoop(); err != nil {
			log.Println(err)
		}

		group.mutex.Lock()
		defer group.mutex.Unlock()
		log.Println("erase sub session in group.")
		delete(group.subSessionList, session)
		if len(group.subSessionList) == 0 {
			group.turnToEmptyTick = time.Now().Unix()
		}
	}()
}

func (group *Group) PullIfNeeded(httpFlvPullAddr string) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	if group.pullSession != nil {
		return
	}
	log.Println("set pull session in group.")
	pullSession := NewPullSession(group)
	group.pullSession = pullSession
	go func() {
		defer func() {
			group.mutex.Lock()
			defer group.mutex.Unlock()
			group.pullSession = nil
			log.Println("erase pull session in group.")
		}()

		url := fmt.Sprintf("http://%s/%s/%s.flv", httpFlvPullAddr, group.appName, group.streamName)
		err := pullSession.Connect(url, time.Duration(group.PullConnectTimeout)*time.Second)
		log.Println("pull session connected. ", url)
		if err != nil {
			log.Println("pull session connect failed.", err)
			return
		}

		err = pullSession.RunLoop()
		if err != nil {
			return
		}
	}()

}

func (group *Group) IsTotalEmpty() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.pullSession == nil && len(group.subSessionList) == 0
}

func (group *Group) ReadHttpRespHeaderCb() {
	log.Println("ReadHttpRespHeaderCb")
}

func (group *Group) ReadFlvHeaderCb(flvHeader []byte) {
	log.Println("ReadFlvHeaderCb")
}

func (group *Group) ReadTagCb(header *TagHeader, tag []byte) {
	//log.Println(header.t, header.timestamp)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	// TODO chef: assume that write fast and would not block
	for session := range group.subSessionList {
		session.WritePacket(tag)
	}
}
