package main

import (
	"fmt"
	"github.com/q191201771/lal/httpflv"
	"github.com/q191201771/lal/log"
	"github.com/q191201771/lal/util"
	"sync"
	"time"
)

//
//type InSession interface {
//	SetStartTick(tick int64)
//	StartTick() int64
//}

type Group struct {
	config     *Config
	appName    string
	streamName string

	exitChan        chan bool
	pullSession     *httpflv.PullSession
	subSessionList  map[*httpflv.SubSession]bool
	turnToEmptyTick int64 // trace while sub session list turn to empty
	gopCache        *httpflv.GOPCache
	mutex           sync.Mutex

	UniqueKey string
}

func NewGroup(appName string, streamName string, config *Config) *Group {
	uk := util.GenUniqueKey("FLVGROUP")
	log.Infof("lifecycle new Group. [%s] appName=%s streamName=%s", uk, appName, streamName)

	return &Group{
		config:         config,
		appName:        appName,
		streamName:     streamName,
		exitChan:       make(chan bool),
		subSessionList: make(map[*httpflv.SubSession]bool),
		gopCache:       httpflv.NewGOPCache(config.GOPCacheNum),
		UniqueKey:      uk,
	}
}

func (group *Group) RunLoop() {
	t := time.NewTicker(300 * time.Millisecond)
	defer t.Stop()

	for {
		select {
		case <-group.exitChan:
			return
		case <-t.C:
			now := time.Now().Unix()

			// TODO chef: do timeout stuff. and do it fast.

			group.mutex.Lock()
			if group.pullSession != nil {
				if isReadTimeout, _ := group.pullSession.ConnStat.Check(now); isReadTimeout {
					log.Warnf("pull session read timeout. [%s]", group.pullSession.UniqueKey)
					group.disposePullSession(lalErr)
				}
			}
			group.mutex.Unlock()

			group.mutex.Lock()
			for session := range group.subSessionList {
				if _, isWriteTimeout := session.ConnStat.Check(now); isWriteTimeout {
					log.Warnf("sub session write timeout. [%s]", session)
					delete(group.subSessionList, session)
					session.Dispose(lalErr)
				}
			}
			group.mutex.Unlock()

			if group.config.Pull.StopPullWhileNoSubTimeout != 0 {
				group.mutex.Lock()
				if group.pullSession != nil && group.turnToEmptyTick != 0 && len(group.subSessionList) == 0 &&
					now-group.turnToEmptyTick > group.config.Pull.StopPullWhileNoSubTimeout {

					log.Infof("stop pull while no SubSession. [%s]", group.pullSession.UniqueKey)
					group.disposePullSession(lalErr)
				}
				group.mutex.Unlock()
			}
		}
	}
}

func (group *Group) Dispose(err error) {
	log.Infof("lifecycle dispose Group. [%s] reason=%v", group.UniqueKey, err)
	group.exitChan <- true
}

func (group *Group) AddSubSession(session *httpflv.SubSession) {
	group.mutex.Lock()
	log.Debugf("add SubSession into group. [%s]", session.UniqueKey)
	group.subSessionList[session] = true
	group.turnToEmptyTick = 0

	go func() {
		if err := session.RunLoop(); err != nil {
			log.Debugf("SubSession loop done. [%s] err=%v", session.UniqueKey, err)
		}

		group.mutex.Lock()
		defer group.mutex.Unlock()
		log.Infof("del SubSession out of group. [%s]", session.UniqueKey)
		delete(group.subSessionList, session)
		if len(group.subSessionList) == 0 {
			group.turnToEmptyTick = time.Now().Unix()
		}
	}()

	session.WriteHTTPResponseHeader()
	session.WriteFlvHeader()
	if group.gopCache.WriteWholeThings(session) {
		session.HasKeyFrame = true
	}
	group.mutex.Unlock()
}

func (group *Group) PullIfNeeded(httpFlvPullAddr string) {
	group.mutex.Lock()
	if group.pullSession != nil {
		return
	}
	pullSession := httpflv.NewPullSession(group, group.config.Pull.ConnectTimeout, group.config.Pull.ReadTimeout)
	group.pullSession = pullSession
	group.mutex.Unlock()

	go func() {
		defer func() {
			group.mutex.Lock()
			defer group.mutex.Unlock()
			group.pullSession = nil
			log.Infof("del PullSession out of group. [%s]", pullSession.UniqueKey)
		}()

		log.Infof("<----- connect. [%s]", pullSession.UniqueKey)
		url := fmt.Sprintf("http://%s/%s/%s.flv", httpFlvPullAddr, group.appName, group.streamName)
		err := pullSession.Connect(url)
		if err != nil {
			log.Errorf("-----> connect error. [%s] err=%v", pullSession.UniqueKey, err)
			return
		}
		log.Infof("-----> connect succ. [%s]", pullSession.UniqueKey)

		err = pullSession.RunLoop()
		if err != nil {
			log.Debugf("PullSession loop done. [%s] err=%v", pullSession.UniqueKey, err)
			return
		}
	}()

}

func (group *Group) IsTotalEmpty() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.pullSession == nil && len(group.subSessionList) == 0
}

func (group *Group) ReadHTTPRespHeaderCB() {
	//log.Debugf("ReadHTTPRespHeaderCb. [%s]", group.UniqueKey)
}

func (group *Group) ReadFlvHeaderCB(flvHeader []byte) {
	//log.Debugf("ReadFlvHeaderCb. [%s]", group.UniqueKey)
}

func (group *Group) ReadTagCB(tag *httpflv.Tag) {
	//log.Debug(header.t, header.timestamp)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	// TODO chef: assume that write fast and would not block
	for session := range group.subSessionList {
		if session.HasKeyFrame {
			session.WritePacket(tag.Raw)
		} else {
			if tag.IsMetadata() || tag.IsAVCKeySeqHeader() || tag.IsAACSeqHeader() || tag.IsAVCKeyNalu() {
				if tag.IsAVCKeyNalu() {
					session.HasKeyFrame = true
				}
				session.WritePacket(tag.Raw)
			}
		}
	}
	group.gopCache.Push(tag)
}

func (group *Group) disposePullSession(err error) {
	group.pullSession.Dispose(err)
	group.pullSession = nil
	group.gopCache.ClearAll()
}

func (group *Group) isInExist() bool {
	return group.pullSession != nil
}
