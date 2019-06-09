package main

import (
	"fmt"
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/log"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/lal/pkg/util"
	"sync"
	"time"
)

type Group struct {
	config     *Config
	appName    string
	streamName string

	exitChan              chan bool
	rtmpPubSession        *rtmp.PubSession
	rtmpPullSession       *rtmp.PullSession
	httpFlvPullSession    *httpflv.PullSession
	httpFlvSubSessionList map[*httpflv.SubSession]struct{}
	rtmpSubSessionList    map[*rtmp.SubSession]struct{}
	turnToEmptyTick       int64 // trace while sub session list turn to empty
	gopCache              *httpflv.GOPCache
	mutex                 sync.Mutex

	UniqueKey string
}

func NewGroup(appName string, streamName string, config *Config) *Group {
	uk := util.GenUniqueKey("FLVGROUP")
	log.Infof("lifecycle new Group. [%s] appName=%s streamName=%s", uk, appName, streamName)

	return &Group{
		config:                config,
		appName:               appName,
		streamName:            streamName,
		exitChan:              make(chan bool),
		httpFlvSubSessionList: make(map[*httpflv.SubSession]struct{}),
		rtmpSubSessionList:    make(map[*rtmp.SubSession]struct{}),
		gopCache:              httpflv.NewGOPCache(config.GOPCacheNum),
		UniqueKey:             uk,
	}
}

func (group *Group) RunLoop() {
	t := time.NewTicker(200 * time.Millisecond)
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
			for session := range group.httpFlvSubSessionList {
				if _, isWriteTimeout := session.ConnStat.Check(now); isWriteTimeout {
					log.Warnf("sub session write timeout. [%s]", session)
					delete(group.httpFlvSubSessionList, session)
					session.Dispose(lalErr)
				}
			}
			group.mutex.Unlock()

			if group.config.Pull.StopPullWhileNoSubTimeout != 0 {
				group.mutex.Lock()
				if group.httpFlvPullSession != nil && group.turnToEmptyTick != 0 && len(group.httpFlvSubSessionList) == 0 &&
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

func (group *Group) AddHTTPFlvSubSession(session *httpflv.SubSession) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	log.Debugf("add SubSession into group. [%s]", session.UniqueKey)
	group.httpFlvSubSessionList[session] = struct{}{}
	group.turnToEmptyTick = 0

	go func() {
		if err := session.RunLoop(); err != nil {
			log.Debugf("SubSession loop done. [%s] err=%v", session.UniqueKey, err)
		}

		group.mutex.Lock()
		defer group.mutex.Unlock()
		log.Infof("del SubSession out of group. [%s]", session.UniqueKey)
		delete(group.httpFlvSubSessionList, session)
		if len(group.httpFlvSubSessionList) == 0 {
			group.turnToEmptyTick = time.Now().Unix()
		}
	}()

	// TODO chef: 在这里发送http和flv的头，还是确保有数据了再发
	session.WriteHTTPResponseHeader()
	session.WriteFlvHeader()
	if group.gopCache.WriteWholeThings(session) {
		session.HasKeyFrame = true
	}
}

func (group *Group) AddRTMPSubSession(session *rtmp.SubSession) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	log.Debugf("add SubSession into group. [%s]", session.UniqueKey)
	group.rtmpSubSessionList[session] = struct{}{}
	group.turnToEmptyTick = 0
}

func (group *Group) AddRTMPPubSession(session *rtmp.PubSession) {
	// TODO chef: 如果已经存在输入，应该拒绝掉这次推流
	group.mutex.Lock()
	defer group.mutex.Unlock()
	log.Debugf("add PubSession into group. [%s]", session.UniqueKey)
	group.rtmpPubSession = session
	session.SetAVMessageObserver(group)
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
	return group.httpFlvPullSession == nil &&
		group.rtmpPullSession == nil &&
		group.rtmpPubSession == nil &&
		len(group.httpFlvSubSessionList) == 0 &&
		len(group.rtmpSubSessionList) == 0
}

func (group *Group) ReadHTTPRespHeaderCB() {
}

func (group *Group) ReadFlvHeaderCB(flvHeader []byte) {
}

func (group *Group) ReadTagCB(tag *httpflv.Tag) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	// TODO chef: assume that write fast and would not block
	for session := range group.httpFlvSubSessionList {
		// TODO chef: 如果一个流上只有音频永远没有视频该如何处理
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

func (group *Group) ReadAVMessageCB(header rtmp.Header, timestampAbs int, message []byte) {
	//log.Info(t)
	group.mutex.Lock()
	defer group.mutex.Unlock()

	//for session := range group.rtmpSubSessionList {
	//
	//}

	flvTag := httpflv.PackHTTPFlvTag(uint8(header.MsgTypeID), timestampAbs, message)
	for session := range group.httpFlvSubSessionList {
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
