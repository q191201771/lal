// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"fmt"
	"sync"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/bele"

	"github.com/q191201771/lal/pkg/aac"

	"github.com/q191201771/lal/pkg/avc"
	"github.com/q191201771/lal/pkg/rtsp"

	"github.com/q191201771/lal/pkg/hls"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"
)

// TODO chef:
//  - group可以考虑搞个协程
//  - 多长没有sub订阅拉流，关闭pull回源
//  - pull重试次数
//  - sub无数据超时时间

type Group struct {
	UniqueKey string

	appName    string
	streamName string

	exitChan chan struct{}

	mutex                sync.Mutex
	rtmpPubSession       *rtmp.ServerSession
	rtspPubSession       *rtsp.PubSession
	pullSession          *rtmp.PullSession
	isPulling            bool
	rtmpSubSessionSet    map[*rtmp.ServerSession]struct{}
	httpflvSubSessionSet map[*httpflv.SubSession]struct{}
	hlsMuxer             *hls.Muxer
	url2PushProxy        map[string]*pushProxy
	gopCache             *GOPCache
	httpflvGopCache      *GOPCache

	adts aac.ADTS
	asc  []byte
	sps  []byte
	pps  []byte
}

type pushProxy struct {
	isPushing   bool
	pushSession *rtmp.PushSession
}

func NewGroup(appName string, streamName string) *Group {
	uk := unique.GenUniqueKey("GROUP")
	nazalog.Infof("[%s] lifecycle new group. appName=%s, streamName=%s", uk, appName, streamName)

	url2PushProxy := make(map[string]*pushProxy)
	if config.RelayPushConfig.Enable {
		for _, addr := range config.RelayPushConfig.AddrList {
			url := fmt.Sprintf("rtmp://%s/%s/%s", addr, appName, streamName)
			url2PushProxy[url] = &pushProxy{
				isPushing:   false,
				pushSession: nil,
			}
		}
	}

	return &Group{
		UniqueKey:            uk,
		appName:              appName,
		streamName:           streamName,
		exitChan:             make(chan struct{}, 1),
		rtmpSubSessionSet:    make(map[*rtmp.ServerSession]struct{}),
		httpflvSubSessionSet: make(map[*httpflv.SubSession]struct{}),
		gopCache:             NewGOPCache("rtmp", uk, config.RTMPConfig.GOPNum),
		httpflvGopCache:      NewGOPCache("httpflv", uk, config.HTTPFLVConfig.GOPNum),
		url2PushProxy:        url2PushProxy,
	}
}

func (group *Group) RunLoop() {
	<-group.exitChan
}

// TODO chef: 传入时间
func (group *Group) Tick() {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	group.pullIfNeeded()
	group.pushIfNeeded()
}

// 主动释放所有资源。保证所有资源的生命周期逻辑上都在我们的控制中。降低出bug的几率，降低心智负担。
// 注意，Dispose后，不应再使用这个对象。
// 值得一提，如果是从其他协程回调回来的消息，在使用Group中的资源前，要判断资源是否存在以及可用。
//
// TODO chef:
//  后续弄个协程来替换掉目前锁的方式，来做消息同步。这样有个好处，就是不用写很多的资源有效判断。统一写一个就好了。
//  目前Dispose在IsTotalEmpty时调用，暂时没有这个问题。
func (group *Group) Dispose() {
	nazalog.Infof("[%s] lifecycle dispose group.", group.UniqueKey)
	group.exitChan <- struct{}{}

	group.mutex.Lock()
	defer group.mutex.Unlock()

	if group.rtmpPubSession != nil {
		group.rtmpPubSession.Dispose()
		group.rtmpPubSession = nil
	}

	for session := range group.rtmpSubSessionSet {
		session.Dispose()
	}
	group.rtmpSubSessionSet = nil

	for session := range group.httpflvSubSessionSet {
		session.Dispose()
	}
	group.httpflvSubSessionSet = nil

	if group.hlsMuxer != nil {
		group.hlsMuxer.Dispose()
		group.hlsMuxer = nil
	}

	if config.RelayPushConfig.Enable {
		for _, v := range group.url2PushProxy {
			if v.pushSession != nil {
				v.pushSession.Dispose()
			}
		}
		group.url2PushProxy = nil
	}
}

func (group *Group) AddRTMPPubSession(session *rtmp.ServerSession) bool {
	nazalog.Debugf("[%s] [%s] add PubSession into group.", group.UniqueKey, session.UniqueKey)

	group.mutex.Lock()
	defer group.mutex.Unlock()

	if group.rtmpPubSession != nil {
		nazalog.Errorf("[%s] PubSession already exist in group. old=%s, new=%s", group.UniqueKey, group.rtmpPubSession.UniqueKey, session.UniqueKey)
		return false
	}
	group.rtmpPubSession = session

	if config.HLSConfig.Enable {
		group.hlsMuxer = hls.NewMuxer(group.streamName, &config.HLSConfig.MuxerConfig)
		group.hlsMuxer.Start()
	}

	if config.RelayPushConfig.Enable {
		group.pushIfNeeded()
	}

	session.SetPubSessionObserver(group)

	return true
}

func (group *Group) DelRTMPPubSession(session *rtmp.ServerSession) {
	nazalog.Debugf("[%s] [%s] del PubSession from group.", group.UniqueKey, session.UniqueKey)

	group.mutex.Lock()
	defer group.mutex.Unlock()

	group.rtmpPubSession = nil

	if config.HLSConfig.Enable && group.hlsMuxer != nil {
		group.hlsMuxer.Dispose()
		group.hlsMuxer = nil
	}

	if config.RelayPushConfig.Enable {
		for _, v := range group.url2PushProxy {
			if v.pushSession != nil {
				v.pushSession.Dispose()
			}
			v.pushSession = nil
		}
	}

	group.gopCache.Clear()
	group.httpflvGopCache.Clear()
}

func (group *Group) AddRTSPPubSession(session *rtsp.PubSession) {
	nazalog.Debugf("[%s] [%s] add RTSP PubSession into group.", group.UniqueKey, session.UniqueKey)

	group.mutex.Lock()
	defer group.mutex.Unlock()

	group.rtspPubSession = session

	// TODO chef: 有in创建时，将hls，relay push等操作集中到一个小函数中
	if config.HLSConfig.Enable {
		group.hlsMuxer = hls.NewMuxer(group.streamName, &config.HLSConfig.MuxerConfig)
		group.hlsMuxer.Start()
	}

	session.SetObserver(group)
}

func (group *Group) AddRTMPPullSession(session *rtmp.PullSession) {
	nazalog.Debugf("[%s] [%s] add PullSession into group.", group.UniqueKey, session.UniqueKey())

	group.mutex.Lock()
	defer group.mutex.Unlock()

	group.pullSession = session

	if config.HLSConfig.Enable {
		group.hlsMuxer = hls.NewMuxer(group.streamName, &config.HLSConfig.MuxerConfig)
		group.hlsMuxer.Start()
	}
}

func (group *Group) DelRTMPPullSession(session *rtmp.PullSession) {
	nazalog.Debugf("[%s] [%s] del PullSession from group.", group.UniqueKey, session.UniqueKey())

	group.mutex.Lock()
	defer group.mutex.Unlock()

	group.pullSession = nil
	group.isPulling = false

	if config.HLSConfig.Enable && group.hlsMuxer != nil {
		group.hlsMuxer.Dispose()
		group.hlsMuxer = nil
	}

	group.gopCache.Clear()
	group.httpflvGopCache.Clear()
}

func (group *Group) AddRTMPSubSession(session *rtmp.ServerSession) {
	nazalog.Debugf("[%s] [%s] add SubSession into group.", group.UniqueKey, session.UniqueKey)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.rtmpSubSessionSet[session] = struct{}{}

	group.pullIfNeeded()
}

func (group *Group) DelRTMPSubSession(session *rtmp.ServerSession) {
	nazalog.Debugf("[%s] [%s] del SubSession from group.", group.UniqueKey, session.UniqueKey)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	delete(group.rtmpSubSessionSet, session)
}

func (group *Group) AddHTTPFLVSubSession(session *httpflv.SubSession) {
	nazalog.Debugf("[%s] [%s] add httpflv SubSession into group.", group.UniqueKey, session.UniqueKey)
	session.WriteHTTPResponseHeader()
	session.WriteFLVHeader()

	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.httpflvSubSessionSet[session] = struct{}{}

	group.pullIfNeeded()
}

func (group *Group) DelHTTPFLVSubSession(session *httpflv.SubSession) {
	nazalog.Debugf("[%s] [%s] del httpflv SubSession from group.", group.UniqueKey, session.UniqueKey)
	group.mutex.Lock()
	defer group.mutex.Unlock()
	delete(group.httpflvSubSessionSet, session)
}

func (group *Group) AddRTMPPushSession(url string, session *rtmp.PushSession) {
	nazalog.Debugf("[%s] [%s] add rtmp PushSession into group.", group.UniqueKey, session.UniqueKey())
	group.mutex.Lock()
	defer group.mutex.Unlock()
	if group.url2PushProxy != nil {
		group.url2PushProxy[url].pushSession = session
	}
}

func (group *Group) DelRTMPPushSession(url string, session *rtmp.PushSession) {
	nazalog.Debugf("[%s] [%s] del rtmp PushSession into group.", group.UniqueKey, session.UniqueKey())
	group.mutex.Lock()
	defer group.mutex.Unlock()
	if group.url2PushProxy != nil {
		group.url2PushProxy[url].pushSession = nil
		group.url2PushProxy[url].isPushing = false
	}
}

func (group *Group) IsTotalEmpty() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	hasPushSession := false
	for _, item := range group.url2PushProxy {
		if item.isPushing || item.pushSession != nil {
			hasPushSession = true
			break
		}
	}

	return group.rtmpPubSession == nil && len(group.rtmpSubSessionSet) == 0 &&
		len(group.httpflvSubSessionSet) == 0 &&
		group.hlsMuxer == nil &&
		!hasPushSession &&
		group.pullSession == nil
}

func (group *Group) IsInEmpty() bool {
	// TODO chef: impl me
	// 判断所有pub、pull类型是否为空
	return false
}

// rtmp.PubSession or rtmp.PullSession
func (group *Group) OnReadRTMPAVMsg(msg base.RTMPMsg) {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	//nazalog.Debugf("%+v, %02x, %02x", msg.Header, msg.Payload[0], msg.Payload[1])
	group.broadcastRTMP(msg)
}

func (group *Group) OnASC(asc []byte) {
	if err := group.adts.InitWithAACAudioSpecificConfig(asc); err != nil {
		nazalog.Errorf("init with aac asc failed. err=%+v", err)
		return
	}
	group.asc = asc
	group.broadcastMetadataAndSeqHeader()
}

func (group *Group) OnSPSPPS(sps, pps []byte) {
	group.sps = sps
	group.pps = pps
	group.broadcastMetadataAndSeqHeader()
}

func (group *Group) OnAVPacket(pkt base.AVPacket) {
	var h base.RTMPHeader
	var msg base.RTMPMsg

	switch pkt.PayloadType {
	case base.AVPacketPTAVC:
		h.TimestampAbs = pkt.Timestamp
		h.MsgStreamID = rtmp.MSID1

		h.MsgTypeID = base.RTMPTypeIDVideo
		h.CSID = rtmp.CSIDVideo
		h.MsgLen = uint32(len(pkt.Payload)) + 9

		msg.Payload = make([]byte, h.MsgLen)
		switch pkt.Payload[0] & 0x1F {
		case avc.NALUTypeIDRSlice:
			msg.Payload[0] = 0x17
			msg.Payload[1] = 0x1
			msg.Payload[2] = 0x0
			msg.Payload[3] = 0x0
			msg.Payload[4] = 0x0
			bele.BEPutUint32(msg.Payload[5:], uint32(len(pkt.Payload)))
			copy(msg.Payload[9:], pkt.Payload)
		default:
			msg.Payload[0] = 0x27
			msg.Payload[1] = 0x1
			msg.Payload[2] = 0x0
			msg.Payload[3] = 0x0
			msg.Payload[4] = 0x0
			bele.BEPutUint32(msg.Payload[5:], uint32(len(pkt.Payload)))
			copy(msg.Payload[9:], pkt.Payload)
		}
	case base.AVPacketPTAAC:
		h.TimestampAbs = pkt.Timestamp
		h.MsgStreamID = rtmp.MSID1

		h.MsgTypeID = base.RTMPTypeIDAudio
		h.CSID = rtmp.CSIDAudio
		h.MsgLen = uint32(len(pkt.Payload)) + 2

		msg.Payload = make([]byte, h.MsgLen)
		msg.Payload[0] = 0xAF
		msg.Payload[1] = 0x1
		copy(msg.Payload[2:], pkt.Payload)
	default:
		nazalog.Errorf("unknown payload type. pt=%d", pkt.PayloadType)
	}

	msg.Header = h
	group.broadcastRTMP(msg)
}

func (group *Group) StringifyStats() string {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	var pub string
	if group.rtmpPubSession == nil {
		pub = "none"
	} else {
		pub = group.rtmpPubSession.UniqueKey
	}
	var pull string
	if group.pullSession == nil {
		pull = "none"
	} else {
		pull = group.pullSession.UniqueKey()
	}
	var pushSize int
	for _, v := range group.url2PushProxy {
		if v.pushSession != nil {
			pushSize++
		}
	}

	return fmt.Sprintf("[%s] stream name=%s, rtmp pub=%s, relay rtmp pull=%s, rtmp sub size=%d, httpflv sub size=%d, relay rtmp push size=%d",
		group.UniqueKey, group.streamName, pub, pull, len(group.rtmpSubSessionSet), len(group.httpflvSubSessionSet), pushSize)
}

func (group *Group) broadcastMetadataAndSeqHeader() {
	if group.asc == nil || group.sps == nil || group.pps == nil {
		return
	}

	ctx, err := avc.ParseSPS(group.sps)
	if err != nil {
		nazalog.Errorf("parse sps failed. err=%+v", err)
		return
	}

	metadata, err := rtmp.BuildMetadata(int(ctx.Width), int(ctx.Height), int(base.RTMPSoundFormatACC), int(base.RTMPCodecIDAVC))
	if err != nil {
		nazalog.Errorf("build metadata failed. err=%+v", err)
		return
	}
	vsh, err := avc.BuildSeqHeaderFromSPSPPS(group.sps, group.pps)
	if err != nil {
		nazalog.Errorf("build avc seq header failed. err=%+v", err)
		return
	}
	ash, err := aac.BuildAACSeqHeader(group.asc)
	if err != nil {
		nazalog.Errorf("build aac seq header failed. err=%+v", err)
		return
	}

	var h base.RTMPHeader
	var msg base.RTMPMsg

	h.MsgLen = uint32(len(metadata))
	h.TimestampAbs = 0
	h.MsgTypeID = base.RTMPTypeIDMetadata
	h.MsgStreamID = rtmp.MSID1
	h.CSID = rtmp.CSIDAMF
	msg.Header = h
	msg.Payload = metadata
	group.broadcastRTMP(msg)

	h.MsgLen = uint32(len(vsh))
	h.TimestampAbs = 0
	h.MsgTypeID = base.RTMPTypeIDVideo
	h.MsgStreamID = rtmp.MSID1
	h.CSID = rtmp.CSIDVideo
	msg.Header = h
	msg.Payload = vsh
	group.broadcastRTMP(msg)

	h.MsgLen = uint32(len(ash))
	h.TimestampAbs = 0
	h.MsgTypeID = base.RTMPTypeIDAudio
	h.CSID = rtmp.CSIDAudio
	msg.Header = h
	msg.Payload = ash
	msg.Header = h
	msg.Payload = ash
	group.broadcastRTMP(msg)
}

// TODO chef: 目前相当于其他类型往rtmp.AVMsg转了，考虑统一往一个通用类型转
// @param msg 调用结束后，内部不持有msg.Payload内存块
func (group *Group) broadcastRTMP(msg base.RTMPMsg) {
	var (
		lcd    LazyChunkDivider
		lrm2ft LazyRTMPMsg2FLVTag
	)

	// # 0. hls
	if config.HLSConfig.Enable && group.hlsMuxer != nil {
		group.hlsMuxer.FeedRTMPMessage(msg)
	}

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
			for i := 0; i < group.gopCache.GetGOPCount(); i++ {
				for _, item := range group.gopCache.GetGOPDataAt(i) {
					_ = session.AsyncWrite(item)
				}
			}

			session.IsFresh = false
		}

		// ## 3.2. 转发本次数据
		_ = session.AsyncWrite(lcd.Get())
	}

	// TODO chef: rtmp sub, rtmp push, httpflv sub 的发送逻辑都差不多，可以考虑封装一下
	if config.RelayPushConfig.Enable {
		for _, v := range group.url2PushProxy {
			if v.pushSession == nil {
				continue
			}

			if v.pushSession.IsFresh {
				if group.gopCache.Metadata != nil {
					_ = v.pushSession.AsyncWrite(group.gopCache.Metadata)
				}
				if group.gopCache.VideoSeqHeader != nil {
					_ = v.pushSession.AsyncWrite(group.gopCache.VideoSeqHeader)
				}
				if group.gopCache.AACSeqHeader != nil {
					_ = v.pushSession.AsyncWrite(group.gopCache.AACSeqHeader)
				}
				for i := 0; i < group.gopCache.GetGOPCount(); i++ {
					for _, item := range group.gopCache.GetGOPDataAt(i) {
						_ = v.pushSession.AsyncWrite(item)
					}
				}

				v.pushSession.IsFresh = false
			}

			_ = v.pushSession.AsyncWrite(lcd.Get())
		}
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
			for i := 0; i < group.httpflvGopCache.GetGOPCount(); i++ {
				for _, item := range group.httpflvGopCache.GetGOPDataAt(i) {
					session.WriteRawPacket(item)
				}
			}

			session.IsFresh = false
		}

		session.WriteRawPacket(lrm2ft.Get())
	}

	// # 5. 缓存关键信息，以及gop
	if config.RTMPConfig.Enable {
		group.gopCache.Feed(msg, lcd.Get)
	}

	if config.HTTPFLVConfig.Enable {
		group.httpflvGopCache.Feed(msg, lrm2ft.Get)
	}
}

func (group *Group) pullIfNeeded() {
	// pull回源功能没开
	if !config.RelayPullConfig.Enable {
		return
	}
	// 没有sub订阅者
	if len(group.rtmpSubSessionSet) == 0 && len(group.httpflvSubSessionSet) == 0 {
		return
	}
	// 已有pub推流或pull回源
	if group.rtmpPubSession != nil || group.pullSession != nil {
		return
	}
	// 正在回源中
	if group.isPulling {
		return
	}
	group.isPulling = true

	url := fmt.Sprintf("rtmp://%s/%s/%s", config.RelayPullConfig.Addr, group.appName, group.streamName)
	nazalog.Infof("start relay pull. [%s] url=%s", group.UniqueKey, url)

	go func() {
		pullSesion := rtmp.NewPullSession()
		err := pullSesion.Pull(url, group.OnReadRTMPAVMsg)
		if err != nil {
			nazalog.Errorf("[%s] relay pull fail. err=%v", pullSesion.UniqueKey(), err)
			group.DelRTMPPullSession(pullSesion)
			return
		}
		group.AddRTMPPullSession(pullSesion)
		err = <-pullSesion.Done()
		nazalog.Infof("[%s] relay pull done. err=%v", pullSesion.UniqueKey(), err)
		group.DelRTMPPullSession(pullSesion)
	}()
}

func (group *Group) pushIfNeeded() {
	// push转推功能没开
	if !config.RelayPushConfig.Enable {
		return
	}
	// 没有pub发布者
	if group.rtmpPubSession == nil {
		return
	}
	for url, v := range group.url2PushProxy {
		// 正在转推中
		if v.isPushing {
			continue
		}
		v.isPushing = true

		nazalog.Infof("[%s] start relay push. url=%s", group.UniqueKey, url)

		go func(url string) {
			pushSession := rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
				option.ConnectTimeoutMS = relayPushConnectTimeoutMS
				option.PushTimeoutMS = relayPushTimeoutMS
				option.WriteAVTimeoutMS = relayPushWriteAVTimeoutMS
			})
			err := pushSession.Push(url)
			if err != nil {
				nazalog.Errorf("[%s] relay push done. err=%v", pushSession.UniqueKey(), err)
				group.DelRTMPPushSession(url, pushSession)
				return
			}
			group.AddRTMPPushSession(url, pushSession)
			err = <-pushSession.Done()
			nazalog.Infof("[%s] relay push done. err=%v", pushSession.UniqueKey(), err)
			group.DelRTMPPushSession(url, pushSession)
		}(url)
	}
}
