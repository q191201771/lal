// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"sync"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	log "github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"
)

type Group struct {
	UniqueKey string

	appName    string
	streamName string

	exitChan chan struct{}

	mutex                sync.Mutex
	pubSession           *rtmp.ServerSession
	pullSession          *rtmp.PullSession
	rtmpSubSessionSet    map[*rtmp.ServerSession]struct{}
	httpflvSubSessionSet map[*httpflv.SubSession]struct{}
	// rtmp chunk格式
	metadata        []byte
	avcKeySeqHeader []byte
	aacSeqHeader    []byte
	// httpflv tag格式
	// TODO chef: 如果没有开启httpflv监听，可以不做格式转换，节约CPU资源
	metadataTag        *httpflv.Tag
	avcKeySeqHeaderTag *httpflv.Tag
	aacSeqHeaderTag    *httpflv.Tag
}

var _ rtmp.PubSessionObserver = &Group{}

func NewGroup(appName string, streamName string) *Group {
	uk := unique.GenUniqueKey("GROUP")
	log.Infof("lifecycle new group. [%s] appName=%s, streamName=%s", uk, appName, streamName)
	return &Group{
		UniqueKey:            uk,
		appName:              appName,
		streamName:           streamName,
		exitChan:             make(chan struct{}, 1),
		rtmpSubSessionSet:    make(map[*rtmp.ServerSession]struct{}),
		httpflvSubSessionSet: make(map[*httpflv.SubSession]struct{}),
	}
}

func (group *Group) RunLoop() {
	<-group.exitChan
}

func (group *Group) Dispose() {
	log.Infof("lifecycle dispose group. [%s]", group.UniqueKey)
	group.exitChan <- struct{}{}

	group.mutex.Lock()
	defer group.mutex.Unlock()
	if group.pubSession != nil {
		group.pubSession.Dispose()
	}
	for session := range group.rtmpSubSessionSet {
		session.Dispose()
	}
	for session := range group.httpflvSubSessionSet {
		session.Dispose()
	}
}

func (group *Group) AddRTMPPubSession(session *rtmp.ServerSession) bool {
	log.Debugf("add PubSession into group. [%s] [%s]", group.UniqueKey, session.UniqueKey)
	group.mutex.Lock()
	if group.pubSession != nil {
		log.Errorf("PubSession already exist in group. [%s] old=%s, new=%s", group.UniqueKey, group.pubSession.UniqueKey, session.UniqueKey)
		return false
	}

	group.pubSession = session
	group.mutex.Unlock()
	session.SetPubSessionObserver(group)
	return true
}

func (group *Group) DelRTMPPubSession(session *rtmp.ServerSession) {
	log.Debugf("del PubSession from group. [%s] [%s]", group.UniqueKey, session.UniqueKey)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.pubSession = nil
	group.metadata = nil
	group.avcKeySeqHeader = nil
	group.aacSeqHeader = nil
	group.metadataTag = nil
	group.avcKeySeqHeaderTag = nil
	group.aacSeqHeaderTag = nil
}

func (group *Group) AddRTMPSubSession(session *rtmp.ServerSession) {
	log.Debugf("add SubSession into group. [%s] [%s]", group.UniqueKey, session.UniqueKey)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.rtmpSubSessionSet[session] = struct{}{}

	// TODO chef: 多长没有拉流session存在的功能
	//group.turnToEmptyTick = 0
}

func (group *Group) DelRTMPSubSession(session *rtmp.ServerSession) {
	log.Debugf("del SubSession from group. [%s] [%s]", group.UniqueKey, session.UniqueKey)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	delete(group.rtmpSubSessionSet, session)
}

func (group *Group) AddHTTPFLVSubSession(session *httpflv.SubSession) {
	log.Debugf("add httpflv SubSession into group. [%s] [%s]", group.UniqueKey, session.UniqueKey)
	session.WriteHTTPResponseHeader()
	session.WriteFLVHeader()

	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.httpflvSubSessionSet[session] = struct{}{}
}

func (group *Group) DelHTTPFLVSubSession(session *httpflv.SubSession) {
	log.Debugf("del httpflv SubSession from group. [%s] [%s]", group.UniqueKey, session.UniqueKey)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	delete(group.httpflvSubSessionSet, session)
}

func (group *Group) IsTotalEmpty() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.pubSession == nil && len(group.rtmpSubSessionSet) == 0 && len(group.httpflvSubSessionSet) == 0
}

func (group *Group) IsInExist() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.pubSession != nil
}

// PubSession or PullSession
func (group *Group) OnReadRTMPAVMsg(msg rtmp.AVMsg) {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	group.broadcastRTMP(msg)
}

func (group *Group) broadcastRTMP(msg rtmp.AVMsg) {
	//log.Infof("%+v", header)

	var (
		currTag *httpflv.Tag
		lcd     LazyChunkDivider
	)

	// # 1. 设置好用于发送的 rtmp 头部信息
	currHeader := Trans.MakeDefaultRTMPHeader(msg.Header)
	// TODO 这行代码是否放到 MakeDefaultRTMPHeader 中
	currHeader.MsgLen = uint32(len(msg.Payload))
	lcd.Init(msg.Payload, &currHeader)

	// # 2. 广播。遍历所有 rtmp sub session，决定是否转发
	for session := range group.rtmpSubSessionSet {
		// ## 2.1. 如果是新的 sub session，发送已缓存的信息
		if session.IsFresh {
			// 发送缓存的头部信息
			if group.metadata != nil {
				_ = session.AsyncWrite(group.metadata)
			}
			if group.avcKeySeqHeader != nil {
				_ = session.AsyncWrite(group.avcKeySeqHeader)
			}
			if group.aacSeqHeader != nil {
				_ = session.AsyncWrite(group.aacSeqHeader)
			}
			session.IsFresh = false
		}

		// ## 2.2. 判断当前包的类型，以及sub session的状态，决定是否发送，并更新sub session的状态
		switch msg.Header.MsgTypeID {
		case rtmp.TypeidDataMessageAMF0:
			_ = session.AsyncWrite(lcd.Get())
		case rtmp.TypeidAudio:
			_ = session.AsyncWrite(lcd.Get())
		case rtmp.TypeidVideo:
			if session.WaitKeyNalu {
				if msg.Payload[0] == 0x17 && msg.Payload[1] == 0x0 {
					_ = session.AsyncWrite(lcd.Get())
				}
				if msg.Payload[0] == 0x17 && msg.Payload[1] == 0x1 {
					_ = session.AsyncWrite(lcd.Get())
					session.WaitKeyNalu = false
				}
			} else {
				_ = session.AsyncWrite(lcd.Get())
			}

		}
	}

	// # 3. 广播。遍历所有 httpflv sub session，决定是否转发
	for session := range group.httpflvSubSessionSet {
		// ## 3.1. 将当前 message 转换成 tag 格式
		if currTag == nil {
			currTag = Trans.RTMPMsg2FLVTag(msg)
		}

		// ## 3.2. 如果是新的sub session，发送已缓存的信息
		if session.IsFresh {
			// 发送缓存的头部信息
			if group.metadataTag != nil {
				log.Debugf("send cache metadata. [%s]", session.UniqueKey)
				session.WriteTag(group.metadataTag)
			}
			if group.avcKeySeqHeaderTag != nil {
				session.WriteTag(group.avcKeySeqHeaderTag)
			}
			if group.aacSeqHeaderTag != nil {
				session.WriteTag(group.aacSeqHeaderTag)
			}
			session.IsFresh = false
		}

		// ## 3.3. 判断当前包的类型，以及sub session的状态，决定是否发送，并更新sub session的状态
		switch msg.Header.MsgTypeID {
		case rtmp.TypeidDataMessageAMF0:
			session.WriteTag(currTag)
		case rtmp.TypeidAudio:
			session.WriteTag(currTag)
		case rtmp.TypeidVideo:
			if session.WaitKeyNalu {
				if msg.Payload[0] == 0x17 && msg.Payload[1] == 0x0 {
					session.WriteTag(currTag)
				}
				if msg.Payload[0] == 0x17 && msg.Payload[1] == 0x1 {
					session.WriteTag(currTag)
					session.WaitKeyNalu = false
				}
			} else {
				session.WriteTag(currTag)
			}

		}
	}

	// # 4. 缓存 rtmp 以及 httpflv 的 metadata 和 avc key seq header 和 aac seq header
	// 由于可能没有订阅者，所以可能需要重新打包
	switch msg.Header.MsgTypeID {
	case rtmp.TypeidDataMessageAMF0:
		if currTag == nil {
			currTag = Trans.RTMPMsg2FLVTag(msg)
		}
		group.metadata = lcd.Get()
		group.metadataTag = currTag
		log.Debugf("cache metadata. [%s] rtmp size:%d, flv size:%d", group.UniqueKey, len(group.metadata), group.metadataTag.Header.DataSize)
	case rtmp.TypeidVideo:
		// TODO chef: magic number
		if msg.Payload[0] == 0x17 && msg.Payload[1] == 0x0 {
			if currTag == nil {
				currTag = Trans.RTMPMsg2FLVTag(msg)
			}
			group.avcKeySeqHeader = lcd.Get()
			group.avcKeySeqHeaderTag = currTag
			log.Debugf("cache avc key seq header. [%s] rtmp size:%d, flv size:%d", group.UniqueKey, len(group.avcKeySeqHeader), group.avcKeySeqHeaderTag.Header.DataSize)
		}
	case rtmp.TypeidAudio:
		if (msg.Payload[0]>>4) == 0x0a && msg.Payload[1] == 0x0 {
			if currTag == nil {
				currTag = Trans.RTMPMsg2FLVTag(msg)
			}
			group.aacSeqHeader = lcd.Get()
			group.aacSeqHeaderTag = currTag
			log.Debugf("cache aac seq header. [%s] rtmp size:%d, flv size:%d", group.UniqueKey, len(group.aacSeqHeader), group.aacSeqHeaderTag.Header.DataSize)
		}
	}
}
