package rtmp

import (
	"fmt"
	"github.com/q191201771/nezha/pkg/log"
	"sync"
	"time"
)

type GroupObserver interface {
	AVMsgObserver
}

type Group struct {
	appName    string
	streamName string

	pubSession      *ServerSession
	pullSession     *PullSession
	subSessionSet   map[*ServerSession]struct{}
	prevAudioHeader *Header
	prevVideoHeader *Header

	// TODO chef:
	metadata        []byte
	avcKeySeqHeader []byte
	aacSeqHeader    []byte

	mutex sync.Mutex

	obs GroupObserver
}

func NewGroup(appName string, streamName string) *Group {
	return &Group{
		appName:       appName,
		streamName:    streamName,
		subSessionSet: make(map[*ServerSession]struct{}),
	}
}

func (group *Group) RunLoop() {
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			//noop
		}
	}
}

func (group *Group) Dispose() {

}

func (group *Group) AddPubSession(session *ServerSession) {
	log.Debugf("add PubSession into group. [%s]", session.UniqueKey)
	group.mutex.Lock()
	group.pubSession = session
	group.mutex.Unlock()
	session.SetPubSessionObserver(group)
}

func (group *Group) AddSubSession(session *ServerSession) {
	log.Debugf("add SubSession into group. [%s]", session.UniqueKey)
	group.mutex.Lock()
	group.subSessionSet[session] = struct{}{}
	group.mutex.Unlock()

	// TODO chef: 多长没有拉流session存在的功能
	//group.turnToEmptyTick = 0
}

func (group *Group) DelPubSession(session *ServerSession) {
	log.Debugf("del PubSession from group. [%s]", session.UniqueKey)
	group.mutex.Lock()
	group.pubSession = nil
	group.mutex.Unlock()

}

func (group *Group) DelSubSession(session *ServerSession) {
	log.Debugf("del SubSession from group. [%s]", session.UniqueKey)
	group.mutex.Lock()
	delete(group.subSessionSet, session)
	group.mutex.Unlock()
}

func (group *Group) Pull(addr string, connectTimeout int64) {
	group.pullSession = NewPullSession(group, int(connectTimeout))

	defer func() {
		group.mutex.Lock()
		defer group.mutex.Unlock()
		group.pullSession = nil
		log.Infof("del rtmp PullSession out of group.")
	}()

	url := fmt.Sprintf("rtmp://%s/%s/%s", addr, group.appName, group.streamName)
	if err := group.pullSession.Pull(url); err != nil {
		log.Error(err)
	}
	if err := group.pullSession.WaitLoop(); err != nil {
		log.Debugf("rtmp PullSession loop done. [%s] err=%v", group.pullSession.UniqueKey, err)
		return
	}
}

func (group *Group) IsTotalEmpty() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.pubSession == nil && len(group.subSessionSet) == 0
}

func (group *Group) IsInExist() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.pubSession != nil
}

func (group *Group) SetObserver(obs GroupObserver) {
	group.obs = obs
}

// PubSession or PullSession
func (group *Group) ReadRTMPAVMsgCB(header Header, timestampAbs int, message []byte) {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	group.broadcastRTMP2RTMP(header, timestampAbs, message)

	if group.obs != nil {
		group.obs.ReadRTMPAVMsgCB(header, timestampAbs, message)
	}
}

func (group *Group) broadcastRTMP2RTMP(header Header, timestampAbs int, message []byte) {
	//log.Infof("%+v", header)
	// # 1. 设置好头部信息
	var currHeader Header
	currHeader.MsgLen = len(message)
	currHeader.Timestamp = timestampAbs
	currHeader.MsgTypeID = header.MsgTypeID
	currHeader.MsgStreamID = MSID1
	switch header.MsgTypeID {
	case TypeidDataMessageAMF0:
		currHeader.CSID = CSIDAMF
		//prevHeader = nil
	case TypeidAudio:
		currHeader.CSID = CSIDAudio
		//prevHeader = group.prevAudioHeader
	case TypeidVideo:
		currHeader.CSID = CSIDVideo
		//prevHeader = group.prevVideoHeader
	}

	var absChunks []byte

	// # 2. 广播。遍历所有sub session，决定是否转发
	for session := range group.subSessionSet {
		// ## 2.1. 一个message广播给多个sub session时，只做一次chunk切割
		if absChunks == nil {
			absChunks = Message2Chunks(message, &currHeader, LocalChunkSize)
		}

		// ## 2.2. 如果是新的sub session，发送已缓存的信息
		if session.isFresh {
			// 发送缓存的头部信息
			if group.metadata != nil {
				session.AsyncWrite(group.metadata)
			}
			if group.avcKeySeqHeader != nil {
				session.AsyncWrite(group.avcKeySeqHeader)
			}
			if group.aacSeqHeader != nil {
				session.AsyncWrite(group.aacSeqHeader)
			}
			session.isFresh = false
		}

		// ## 2.3. 判断当前包的类型，以及sub session的状态，决定是否发送并更新sub session的状态
		switch header.MsgTypeID {
		case TypeidDataMessageAMF0:
			session.AsyncWrite(absChunks)
		case TypeidAudio:
			session.AsyncWrite(absChunks)
		case TypeidVideo:
			if session.waitKeyNalu {
				if message[0] == 0x17 && message[1] == 0x0 {
					session.AsyncWrite(absChunks)
				}
				if message[0] == 0x17 && message[1] == 0x1 {
					session.AsyncWrite(absChunks)
					session.waitKeyNalu = false
				}
			} else {
				session.AsyncWrite(absChunks)
			}

		}

	}

	// # 3. 缓存 metadata 和 avc key seq header 和 aac seq header
	// 由于可能没有订阅者，所以message可能还没做chunk切割，所以这里要做判断是否做chunk切割
	switch header.MsgTypeID {
	case TypeidDataMessageAMF0:
		if absChunks == nil {
			absChunks = Message2Chunks(message, &currHeader, LocalChunkSize)
		}
		log.Debug("cache metadata.")
		group.metadata = absChunks
	case TypeidVideo:
		// TODO chef: magic number
		if message[0] == 0x17 && message[1] == 0x0 {
			if absChunks == nil {
				absChunks = Message2Chunks(message, &currHeader, LocalChunkSize)
			}
			log.Debug("cache avc key seq header.")
			group.avcKeySeqHeader = absChunks
		}
	case TypeidAudio:
		if (message[0]>>4) == 0x0a && message[1] == 0x0 {
			if absChunks == nil {
				absChunks = Message2Chunks(message, &currHeader, LocalChunkSize)
			}
			log.Debug("cache aac seq header.")
			group.aacSeqHeader = absChunks
		}
	}
}
