package httpflv

import (
	"fmt"
	"github.com/q191201771/lal/log"
	"github.com/q191201771/lal/util"
	"sync"
	"time"
)

type Group struct {
	Config
	appName    string
	streamName string

	exitChan        chan bool
	pullSession     *PullSession
	subSessionList  map[*SubSession]bool
	turnToEmptyTick int64 // trace while sub session list turn to empty
	gopCache        *GopCache
	mutex           sync.Mutex

	UniqueKey string
}

func NewGroup(appName string, streamName string, config Config) *Group {
	uk := util.GenUniqueKey("FLVGROUP")
	log.Infof("lifecycle new Group. [%s] appName=%s streamName=%s", uk, appName, streamName)

	return &Group{
		Config:         config,
		appName:        appName,
		streamName:     streamName,
		exitChan:       make(chan bool),
		subSessionList: make(map[*SubSession]bool),
		gopCache:       NewGopCache(config.GopCacheNum),
		UniqueKey:uk,
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
						log.Warnf("SubSession idle timeout. session:%s", sub.UniqueKey)
						delete(group.subSessionList, sub)
						sub.Dispose(fxxkErr)
					}
				}
				group.mutex.Unlock()
			}

			if group.PullReadTimeout != 0 && count%group.PullReadTimeout == 0 {
				group.mutex.Lock()
				if group.pullSession != nil {
					if now-group.pullSession.StartTick > group.PullReadTimeout {
						if _, diff := group.pullSession.GetStat(); diff.readByte == 0 {
							log.Warnf("read timeout. [%s]", group.pullSession.UniqueKey)
							group.disposePullSession(fxxkErr)
						}
					}
				}
				group.mutex.Unlock()
			}

			if group.StopPullWhileNoSubTimeout != 0 && count%group.StopPullWhileNoSubTimeout == 0 {
				group.mutex.Lock()
				if group.pullSession != nil && group.turnToEmptyTick != 0 && len(group.subSessionList) == 0 &&
					now-group.turnToEmptyTick > group.StopPullWhileNoSubTimeout {

					log.Debugf("stop pull while no SubSession. [%s]", group.pullSession.UniqueKey)
					group.disposePullSession(fxxkErr)
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

func (group *Group) AddSubSession(session *SubSession) {
	group.mutex.Lock()
	log.Debugf("add SubSession into group. [%s]", session.UniqueKey)
	group.subSessionList[session] = true
	group.turnToEmptyTick = 0

	session.WriteHttpResponseHeader()
	session.WriteFlvHeader()
	if hasKeyFrame, cache := group.gopCache.GetWholeThings(); cache != nil {
		session.HasKeyFrame = hasKeyFrame
		session.WritePacket(cache)
	}
	group.mutex.Unlock()

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
}

func (group *Group) PullIfNeeded(httpFlvPullAddr string) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	if group.pullSession != nil {
		return
	}
	pullSession := NewPullSession(group)
	group.pullSession = pullSession

	go func() {
		defer func() {
			group.mutex.Lock()
			defer group.mutex.Unlock()
			group.pullSession = nil
			log.Infof("del PullSession out of group. [%s]", pullSession.UniqueKey)
		}()

		log.Infof("<----- connect. [%s]", pullSession.UniqueKey)
		url := fmt.Sprintf("http://%s/%s/%s.flv", httpFlvPullAddr, group.appName, group.streamName)
		err := pullSession.Connect(url, time.Duration(group.PullConnectTimeout)*time.Second)
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

func (group *Group) disposePullSession(err error) {
	group.pullSession.Dispose(err)
	group.pullSession = nil
	group.gopCache.ClearAll()
}

func (group *Group) ReadHttpRespHeaderCb() {
	//log.Debugf("ReadHttpRespHeaderCb. [%s]", group.UniqueKey)
}

func (group *Group) ReadFlvHeaderCb(flvHeader []byte) {
	//log.Debugf("ReadFlvHeaderCb. [%s]", group.UniqueKey)
}

func (group *Group) ReadTagCb(tag *Tag) {
	//log.Debug(header.t, header.timestamp)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	// TODO chef: assume that write fast and would not block
	for session := range group.subSessionList {
		if session.HasKeyFrame {
			session.WritePacket(tag.Raw)
		} else {
			if tag.isMetaData() || tag.isAvcKeySeqHeader() || tag.isAacSeqHeader() || tag.isAvcKeyNalu() {
				if tag.isAvcKeyNalu() {
					session.HasKeyFrame = true
				}
				session.WritePacket(tag.Raw)
			}
		}
	}
	group.gopCache.Push(tag)
}
