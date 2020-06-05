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

	"github.com/q191201771/lal/pkg/hls"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	log "github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"
)

type Group struct {
	UniqueKey string

	appName    string
	streamName string

	hlsConfig *hls.MuxerConfig

	exitChan chan struct{}

	mutex                sync.Mutex
	pubSession           *rtmp.ServerSession
	rtmpSubSessionSet    map[*rtmp.ServerSession]struct{}
	httpflvSubSessionSet map[*httpflv.SubSession]struct{}
	hlsMuxer             *hls.Muxer
	gopCache             *GOPCache
	// TODO chef: 如果没有开启httpflv监听，可以不做格式转换，节约CPU资源
	httpflvGopCache *GOPCache
}

var _ rtmp.PubSessionObserver = &Group{}

func NewGroup(appName string, streamName string, rtmpGOPNum int, httpflvGOPNum int, hlsConfig *hls.MuxerConfig) *Group {
	uk := unique.GenUniqueKey("GROUP")
	log.Infof("lifecycle new group. [%s] appName=%s, streamName=%s", uk, appName, streamName)
	return &Group{
		UniqueKey:            uk,
		appName:              appName,
		streamName:           streamName,
		hlsConfig:            hlsConfig,
		exitChan:             make(chan struct{}, 1),
		rtmpSubSessionSet:    make(map[*rtmp.ServerSession]struct{}),
		httpflvSubSessionSet: make(map[*httpflv.SubSession]struct{}),
		gopCache:             NewGOPCache("rtmp", uk, rtmpGOPNum),
		httpflvGopCache:      NewGOPCache("httpflv", uk, rtmpGOPNum),
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
	group.hlsMuxer = hls.NewMuxer(group.streamName, group.hlsConfig)
	group.hlsMuxer.Start()
	group.mutex.Unlock()

	session.SetPubSessionObserver(group)
	return true
}

func (group *Group) DelRTMPPubSession(session *rtmp.ServerSession) {
	log.Debugf("del PubSession from group. [%s] [%s]", group.UniqueKey, session.UniqueKey)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.pubSession = nil
	group.hlsMuxer.Stop()

	group.gopCache.Clear()
	group.httpflvGopCache.Clear()
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

	//log.Debugf("%+v, %02x, %02x", msg.Header, msg.Payload[0], msg.Payload[1])
	group.broadcastRTMP(msg)
	group.hlsMuxer.FeedRTMPMessage(msg)
}

func (group *Group) broadcastRTMP(msg rtmp.AVMsg) {
	//log.Infof("%+v", msg.Header)

	var (
		lcd    LazyChunkDivider
		lrm2ft LazyRTMPMsg2FLVTag
	)

	// # 1. 设置好用于发送的 rtmp 头部信息
	currHeader := Trans.MakeDefaultRTMPHeader(msg.Header)
	// TODO 这行代码是否放到 MakeDefaultRTMPHeader 中
	currHeader.MsgLen = uint32(len(msg.Payload))

	// # 2. 懒初始化rtmp chunk切片，以及httpflv转换
	lcd.Init(msg.Payload, &currHeader)
	lrm2ft.Init(msg)

	// # 3. 广播。遍历所有 rtmp sub session，转发数据
	for session := range group.rtmpSubSessionSet {
		// ## 3.1. 如果是新的 sub session，发送已缓存的信息
		if session.IsFresh {
			// TODO 头信息和full gop也可以在SubSession刚加入时发送
			if group.gopCache.Metadata != nil {
				_ = session.AsyncWrite(group.gopCache.Metadata)
			}
			if group.gopCache.VideoSeqHeader != nil {
				_ = session.AsyncWrite(group.gopCache.VideoSeqHeader)
			}
			if group.gopCache.AACSeqHeader != nil {
				_ = session.AsyncWrite(group.gopCache.AACSeqHeader)
			}
			for i := 0; i < group.gopCache.GetGopLen(); i++ {
				for _, item := range group.gopCache.GetGopDataAt(i) {
					_ = session.AsyncWrite(item)
				}
			}

			session.IsFresh = false
		}

		// ## 3.2. 转发本次数据
		_ = session.AsyncWrite(lcd.Get())
	}

	// # 4. 广播。遍历所有 httpflv sub session，转发数据
	for session := range group.httpflvSubSessionSet {
		if session.IsFresh {
			if group.httpflvGopCache.Metadata != nil {
				session.WriteRawPacket(group.httpflvGopCache.Metadata)
			}
			if group.httpflvGopCache.VideoSeqHeader != nil {
				session.WriteRawPacket(group.httpflvGopCache.VideoSeqHeader)
			}
			if group.httpflvGopCache.AACSeqHeader != nil {
				session.WriteRawPacket(group.httpflvGopCache.AACSeqHeader)
			}
			for i := 0; i < group.httpflvGopCache.GetGopLen(); i++ {
				for _, item := range group.httpflvGopCache.GetGopDataAt(i) {
					session.WriteRawPacket(item)
				}
			}

			session.IsFresh = false
		}

		session.WriteRawPacket(lrm2ft.Get())
	}

	// # 4. 缓存关键信息，以及gop
	group.gopCache.Feed(msg, lcd.Get)
	group.httpflvGopCache.Feed(msg, lrm2ft.Get)
}
