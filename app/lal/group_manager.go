package main

import (
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/lal/pkg/util/log"
	"github.com/q191201771/lal/pkg/util/unique"
	"sync"
	"time"
)

// TODO chef: 没有sub了一定时间后，停止pull

type GroupManager struct {
	config     *Config
	appName    string
	streamName string

	exitChan     chan bool
	rtmpGroup    *rtmp.Group
	httpFlvGroup *httpflv.Group
	//rtmpPullSession    *rtmp.PullSession
	//httpFlvPullSession *httpflv.PullSession
	//turnToEmptyTick    int64 // trace while sub session list turn to empty
	//gopCache           *httpflv.GOPCache
	mutex sync.Mutex

	UniqueKey string
}

func NewGroupManager(appName string, streamName string, config *Config) *GroupManager {
	uk := unique.GenUniqueKey("FLVGROUP")
	log.Infof("lifecycle new Group. [%s] appName=%s streamName=%s", uk, appName, streamName)

	return &GroupManager{
		config:     config,
		appName:    appName,
		streamName: streamName,
		exitChan:   make(chan bool),
		//httpFlvSubSessionList: make(map[*httpflv.SubSession]struct{}),
		//gopCache:  httpflv.NewGOPCache(config.GOPCacheNum),
		UniqueKey: uk,
	}
}

func (gm *GroupManager) RunLoop() {
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()

	for {
		select {
		case <-gm.exitChan:
			return
		case <-t.C:
			//now := time.Now().Unix()

			// TODO chef: do timeout stuff. and do it fast.

			//group.mutex.Lock()
			//if group.httpFlvPullSession != nil {
			//	if isReadTimeout, _ := group.httpFlvPullSession.ConnStat.Check(now); isReadTimeout {
			//		log.Warnf("pull session read timeout. [%s]", group.httpFlvPullSession.UniqueKey)
			//		group.disposePullSession(lalErr)
			//	}
			//}
			//group.mutex.Unlock()

			//group.mutex.Lock()
			//for session := range group.httpFlvSubSessionList {
			//	if _, isWriteTimeout := session.ConnStat.Check(now); isWriteTimeout {
			//		log.Warnf("sub session write timeout. [%s]", session)
			//		delete(group.httpFlvSubSessionList, session)
			//		session.Dispose(lalErr)
			//	}
			//}
			//group.mutex.Unlock()

			//if group.config.Pull.StopPullWhileNoSubTimeout != 0 {
			//	group.mutex.Lock()
			//	if group.httpFlvPullSession != nil && group.turnToEmptyTick != 0 && len(group.httpFlvSubSessionList) == 0 &&
			//		now-group.turnToEmptyTick > group.config.Pull.StopPullWhileNoSubTimeout {
			//
			//		log.Infof("stop pull while no SubSession. [%s]", group.httpFlvPullSession.UniqueKey)
			//		group.disposePullSession(lalErr)
			//	}
			//	group.mutex.Unlock()
			//}
		}
	}
}

func (gm *GroupManager) Dispose(err error) {
	log.Infof("lifecycle dispose Group. [%s] reason=%v", gm.UniqueKey, err)
	gm.exitChan <- true
}

// 返回true则允许推流，返回false则关闭连接
func (gm *GroupManager) AddRTMPPubSession(session *rtmp.PubSession, rtmpGroup *rtmp.Group) bool {
	gm.attachRTMPGroup(rtmpGroup)

	return !gm.isInExist()
}

func (gm *GroupManager) AddRTMPSubSession(session *rtmp.SubSession, rtmpGroup *rtmp.Group) {
	gm.attachRTMPGroup(rtmpGroup)

	gm.pullIfNeeded()
}

func (gm *GroupManager) AddHTTPFlvSubSession(session *httpflv.SubSession, httpFlvGroup *httpflv.Group) {
	gm.attachHTTPFlvGroup(httpFlvGroup)

	gm.pullIfNeeded()
}

func (gm *GroupManager) IsTotalEmpty() bool {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()
	return (gm.rtmpGroup == nil || gm.rtmpGroup.IsTotalEmpty()) &&
		(gm.httpFlvGroup == nil || gm.httpFlvGroup.IsTotalEmpty())
}

// GroupObserver of httpflv.Group
func (gm *GroupManager) ReadHTTPRespHeaderCB() {
	// noop
}

// GroupObserver of httpflv.Group
func (gm *GroupManager) ReadFlvHeaderCB(flvHeader []byte) {
	// noop
}

// GroupObserver of httpflv.Group
func (gm *GroupManager) ReadFlvTagCB(tag *httpflv.Tag) {
	log.Info("ReadFlvTagCB")

	// TODO chef: broadcast to rtmp.Group
}

// GroupObserver of rtmp.Group
func (gm *GroupManager) ReadRTMPAVMsgCB(header rtmp.Header, timestampAbs int, message []byte) {
	log.Info("ReadRTMPAVMsgCB")

	// TODO chef: broadcast to httpflv.Group
}

func (gm *GroupManager) attachRTMPGroup(rtmpGroup *rtmp.Group) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()
	if gm.rtmpGroup != nil && gm.rtmpGroup != rtmpGroup {
		log.Warnf("CHEFNOTICEME %+v %+v", gm.rtmpGroup, rtmpGroup)
	}
	gm.rtmpGroup = rtmpGroup
	rtmpGroup.SetObserver(gm)
}

func (gm *GroupManager) attachHTTPFlvGroup(httpFlvGroup *httpflv.Group) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()
	if gm.httpFlvGroup != nil && gm.httpFlvGroup != httpFlvGroup {
		log.Warnf("CHEFNOTICEME %+v %+v", gm.httpFlvGroup, httpFlvGroup)
	}
	gm.httpFlvGroup = httpFlvGroup
	httpFlvGroup.SetObserver(gm)
}

func (gm *GroupManager) pullIfNeeded() {
	if !gm.isInExist() {
		switch gm.config.Pull.Type {
		case "httpflv":
			go gm.httpFlvGroup.Pull(gm.config.Pull.Addr, gm.config.Pull.ConnectTimeout, gm.config.Pull.ReadTimeout)
		case "rtmp":
			go gm.rtmpGroup.Pull(gm.config.Pull.Addr, gm.config.Pull.ConnectTimeout)
		}
	}
}

func (gm *GroupManager) isInExist() bool {
	return (gm.rtmpGroup != nil && gm.rtmpGroup.IsInExist()) ||
		(gm.httpFlvGroup != nil && gm.httpFlvGroup.IsInExist())
}
