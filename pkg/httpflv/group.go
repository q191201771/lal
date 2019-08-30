package httpflv

import (
	"fmt"
	"github.com/q191201771/nezha/pkg/log"
	"sync"
)

// TODO chef: set me by config
var gopCacheNum = 2

// TODO chef: 所有新增对象的UniqueKey

// TODO chef: 将Observer方式改成 func CB方式
type GroupObserver interface {
	ReadHTTPRespHeaderCB()
	ReadFlvHeaderCB(flvHeader []byte)
	ReadFlvTagCB(tag *Tag)
}

type Group struct {
	appName    string
	streamName string

	pullSession   *PullSession
	subSessionSet map[*SubSession]struct{}
	gopCache      *GOPCache
	mutex         sync.Mutex

	obs GroupObserver
}

func NewGroup(appName string, streamName string) *Group {
	return &Group{
		appName:       appName,
		streamName:    streamName,
		subSessionSet: make(map[*SubSession]struct{}),
		gopCache:      NewGOPCache(gopCacheNum),
	}
}

func (group *Group) RunLoop() {

}

func (group *Group) AddHTTPFlvSubSession(session *SubSession) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	log.Debugf("add SubSession into group. [%s]", session.UniqueKey)
	group.subSessionSet[session] = struct{}{}

	go func() {
		if err := session.RunLoop(); err != nil {
			log.Debugf("SubSession loop done. [%s] err=%v", session.UniqueKey, err)
		}

		group.mutex.Lock()
		defer group.mutex.Unlock()
		log.Infof("del SubSession out of group. [%s]", session.UniqueKey)
		delete(group.subSessionSet, session)
	}()

	// TODO chef: 在这里发送http和flv的头，还是确保有数据了再发
	session.WriteHTTPResponseHeader()
	session.WriteFlvHeader()
	if group.gopCache.WriteWholeThings(session) {
		session.HasKeyFrame = true
	}
}

func (group *Group) Pull(addr string, connectTimeout int64, readTimeout int64) {
	group.pullSession = NewPullSession(PullSessionConfig{
		ConnectTimeoutMS: int(connectTimeout),
		ReadTimeoutMS:    int(readTimeout),
	})

	defer func() {
		group.mutex.Lock()
		defer group.mutex.Unlock()
		group.pullSession = nil
		log.Infof("del httpflv PullSession out of group. [%s]", group.pullSession.UniqueKey)
	}()

	log.Infof("<----- connect. [%s]", group.pullSession.UniqueKey)
	url := fmt.Sprintf("http://%s/%s/%s.flv", addr, group.appName, group.streamName)
	// TODO chef: impl cb
	if err := group.pullSession.Pull(url, group.ReadFlvTagCB); err != nil {

	}
	//if err := group.pullSession.Pull(url, nil); err != nil {
	//log.Errorf("-----> connect error. [%s] err=%v", group.pullSession.UniqueKey, err)
	//return
	//}
	//log.Infof("-----> connect succ. [%s]", group.pullSession.UniqueKey)

	//if err := group.pullSession.RunLoop(); err != nil {
	//	log.Debugf("PullSession loop done. [%s] err=%v", group.pullSession.UniqueKey, err)
	//	return
	//}
}

func (group *Group) IsTotalEmpty() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.pullSession == nil && len(group.subSessionSet) == 0
}

func (group *Group) IsInExist() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return false
}

func (group *Group) SetObserver(obs GroupObserver) {
	// 确保如果调用SetObserver，那么调用发生在Pull之前，就不对obs加锁保护了
	group.obs = obs
}

// PullSessionObserver
func (group *Group) ReadHTTPRespHeaderCB() {
	if group.obs != nil {
		group.obs.ReadHTTPRespHeaderCB()
	}
}

// PullSessionObserver
func (group *Group) ReadFlvHeaderCB(flvHeader []byte) {
	if group.obs != nil {
		group.obs.ReadFlvHeaderCB(flvHeader)
	}
}

// PullSessionObserver
func (group *Group) ReadFlvTagCB(tag *Tag) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	// TODO chef: assume that write fast and would not block
	for session := range group.subSessionSet {
		// TODO chef: 如果一个流上只有音频永远没有视频该如何处理
		if session.HasKeyFrame {
			session.WriteRawPacket(tag.Raw)
		} else {
			if tag.IsMetadata() || tag.IsAVCKeySeqHeader() || tag.IsAACSeqHeader() || tag.IsAVCKeyNalu() {
				if tag.IsAVCKeyNalu() {
					session.HasKeyFrame = true
				}
				session.WriteRawPacket(tag.Raw)
			}
		}
	}
	group.gopCache.Push(tag)

	if group.obs != nil {
		group.obs.ReadFlvTagCB(tag)
	}
}
