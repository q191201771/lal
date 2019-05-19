package main

import (
	"fmt"
	"github.com/q191201771/lal/httpflv"
	"github.com/q191201771/lal/log"
	"github.com/q191201771/lal/rtmp"
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

	exitChan           chan bool
	rtmpPullSession    *rtmp.PullSession
	httpFlvPullSession *httpflv.PullSession
	subSessionList     map[*httpflv.SubSession]bool
	turnToEmptyTick    int64 // trace while sub session list turn to empty
	gopCache           *httpflv.GOPCache
	mutex              sync.Mutex

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
			if group.httpFlvPullSession != nil {
				if isReadTimeout, _ := group.httpFlvPullSession.ConnStat.Check(now); isReadTimeout {
					log.Warnf("pull session read timeout. [%s]", group.httpFlvPullSession.UniqueKey)
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
				if group.httpFlvPullSession != nil && group.turnToEmptyTick != 0 && len(group.subSessionList) == 0 &&
					now-group.turnToEmptyTick > group.config.Pull.StopPullWhileNoSubTimeout {

					log.Infof("stop pull while no SubSession. [%s]", group.httpFlvPullSession.UniqueKey)
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

func (group *Group) PullIfNeeded() {
	group.mutex.Lock()
	if group.isInExist() {
		return
	}
	switch group.config.Pull.Type {
	case "httpflv":
		group.httpFlvPullSession = httpflv.NewPullSession(group, group.config.Pull.ConnectTimeout, group.config.Pull.ReadTimeout)
		go group.pullByHTTPFlv()
	case "rtmp":
		group.rtmpPullSession = rtmp.NewPullSession(group, group.config.Pull.ConnectTimeout)
		go group.pullByRTMP()
	default:
		log.Errorf("unknown pull type. type=%s", group.config.Pull.Type)
	}
	group.mutex.Unlock()
}

func (group *Group) IsTotalEmpty() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.httpFlvPullSession == nil && len(group.subSessionList) == 0
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

func (group *Group) ReadAvMessageCB(t int, timestampAbs int, message []byte) {
	//log.Info(t)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	flvTag := httpflv.PackHTTPFlvTag(uint8(t), timestampAbs, message)
	for session := range group.subSessionList {
		if session.HasKeyFrame {
			session.WritePacket(flvTag)
		} else {
			if httpflv.IsMetadata(flvTag) || httpflv.IsAVCKeySeqHeader(flvTag) || httpflv.IsAACSeqHeader(flvTag) || httpflv.IsAVCKeyNalu(flvTag) {
				if httpflv.IsAVCKeyNalu(flvTag) {
					session.HasKeyFrame = true
				}
				session.WritePacket(flvTag)
			}
		}
	}
}

func (group *Group) pullByHTTPFlv() {
	defer func() {
		group.mutex.Lock()
		defer group.mutex.Unlock()
		group.httpFlvPullSession = nil
		log.Infof("del httpflv PullSession out of group. [%s]", group.httpFlvPullSession.UniqueKey)
	}()

	log.Infof("<----- connect. [%s]", group.httpFlvPullSession.UniqueKey)
	url := fmt.Sprintf("http://%s/%s/%s.flv", group.config.Pull.Addr, group.appName, group.streamName)
	if err := group.httpFlvPullSession.Connect(url); err != nil {
		log.Errorf("-----> connect error. [%s] err=%v", group.httpFlvPullSession.UniqueKey, err)
		return
	}
	log.Infof("-----> connect succ. [%s]", group.httpFlvPullSession.UniqueKey)

	if err := group.httpFlvPullSession.RunLoop(); err != nil {
		log.Debugf("PullSession loop done. [%s] err=%v", group.httpFlvPullSession.UniqueKey, err)
		return
	}
}

func (group *Group) pullByRTMP() {
	defer func() {
		group.mutex.Lock()
		defer group.mutex.Unlock()
		group.rtmpPullSession = nil
		log.Infof("del rtmp PullSession out of group.")
	}()

	url := fmt.Sprintf("rtmp://%s/%s/%s", group.config.Pull.Addr, group.appName, group.streamName)
	if err := group.rtmpPullSession.Pull(url); err != nil {
		log.Error(err)
	}
	if err := group.rtmpPullSession.WaitLoop(); err != nil {
		log.Debugf("rtmp PullSession loop done. [%s] err=%v", group.rtmpPullSession.UniqueKey, err)
		return
	}
}

func (group *Group) disposePullSession(err error) {
	group.httpFlvPullSession.Dispose(err)
	group.httpFlvPullSession = nil
	group.gopCache.ClearAll()
}

func (group *Group) isInExist() bool {
	return group.httpFlvPullSession != nil || group.rtmpPullSession != nil
}
