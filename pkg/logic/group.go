// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/httpts"
	"github.com/q191201771/lal/pkg/mpegts"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/lal/pkg/rtsp"
	"github.com/q191201771/lal/pkg/sdp"
)

type GroupObserver interface {
	CleanupHlsIfNeeded(appName string, streamName string, path string)
}

type Group struct {
	UniqueKey  string // const after init
	appName    string // const after init
	streamName string // const after init TODO chef: 和stat里的字段重复，可以删除掉
	config     *Config
	observer   GroupObserver

	exitChan chan struct{}

	mutex sync.Mutex
	// pub
	rtmpPubSession   *rtmp.ServerSession
	rtspPubSession   *rtsp.PubSession
	rtsp2RtmpRemuxer *remux.AvPacket2RtmpRemuxer
	rtmp2RtspRemuxer *remux.Rtmp2RtspRemuxer
	// pull
	pullEnable bool
	pullUrl    string
	pullProxy  *pullProxy
	// rtmp pub使用
	dummyAudioFilter *remux.DummyAudioFilter
	// rtmp pub/pull使用
	rtmpGopCache    *remux.GopCache
	httpflvGopCache *remux.GopCache
	// rtsp使用
	sdpCtx *sdp.LogicContext
	// mpegts使用
	patpmt []byte
	// sub
	rtmpSubSessionSet    map[*rtmp.ServerSession]struct{}
	httpflvSubSessionSet map[*httpflv.SubSession]struct{}
	httptsSubSessionSet  map[*httpts.SubSession]struct{}
	rtspSubSessionSet    map[*rtsp.SubSession]struct{}
	// push
	pushEnable    bool
	url2PushProxy map[string]*pushProxy
	// hls
	hlsMuxer *hls.Muxer
	// record
	recordFlv    *httpflv.FlvFileWriter
	recordMpegts *mpegts.FileWriter
	// rtmp sub使用
	rtmpMergeWriter *base.MergeWriter // TODO(chef): 后面可以在业务层加一个定时Flush
	//
	stat base.StatGroup
	//
	tickCount uint32
}

type pullProxy struct {
	isPulling   bool
	pullSession *rtmp.PullSession
}

type pushProxy struct {
	isPushing   bool
	pushSession *rtmp.PushSession
}

func NewGroup(appName string, streamName string, config *Config, observer GroupObserver) *Group {
	uk := base.GenUkGroup()

	url2PushProxy := make(map[string]*pushProxy) // TODO(chef): 移入Enable里面并进行review+测试
	if config.RelayPushConfig.Enable {
		for _, addr := range config.RelayPushConfig.AddrList {
			pushUrl := fmt.Sprintf("rtmp://%s/%s/%s", addr, appName, streamName)
			url2PushProxy[pushUrl] = &pushProxy{
				isPushing:   false,
				pushSession: nil,
			}
		}
	}

	g := &Group{
		UniqueKey:  uk,
		appName:    appName,
		streamName: streamName,
		config:     config,
		observer:   observer,
		stat: base.StatGroup{
			StreamName: streamName,
		},
		exitChan:             make(chan struct{}, 1),
		rtmpSubSessionSet:    make(map[*rtmp.ServerSession]struct{}),
		httpflvSubSessionSet: make(map[*httpflv.SubSession]struct{}),
		httptsSubSessionSet:  make(map[*httpts.SubSession]struct{}),
		rtspSubSessionSet:    make(map[*rtsp.SubSession]struct{}),
		rtmpGopCache:         remux.NewGopCache("rtmp", uk, config.RtmpConfig.GopNum),
		httpflvGopCache:      remux.NewGopCache("httpflv", uk, config.HttpflvConfig.GopNum),
		pushEnable:           config.RelayPushConfig.Enable,
		url2PushProxy:        url2PushProxy,
		pullProxy:            &pullProxy{},
	}

	// 根据配置文件中的静态回源配置来初始化回源设置
	var pullUrl string
	if config.RelayPullConfig.Enable {
		pullUrl = fmt.Sprintf("rtmp://%s/%s/%s", config.RelayPullConfig.Addr, appName, streamName)
	}
	g.setPullUrl(config.RelayPullConfig.Enable, pullUrl)

	if config.RtmpConfig.MergeWriteSize > 0 {
		g.rtmpMergeWriter = base.NewMergeWriter(g.writev2RtmpSubSessions, config.RtmpConfig.MergeWriteSize)
	}

	Log.Infof("[%s] lifecycle new group. group=%p, appName=%s, streamName=%s", uk, g, appName, streamName)
	return g
}

func (group *Group) RunLoop() {
	<-group.exitChan
}

// Tick TODO chef: 传入时间
// 目前每秒触发一次
func (group *Group) Tick() {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	group.stopPullIfNeeded()
	group.pullIfNeeded()
	// 还有pub推流，没在push就触发push
	group.pushIfNeeded()

	// TODO chef:
	// 梳理和naza.Connection超时重复部分

	// TODO(chef): 所有dispose后，是否需要做打扫处理(比如设置nil以及主动调del)，还是都在后面的del回调函数中统一处理
	// 定时关闭没有数据的session
	if group.tickCount%checkSessionAliveIntervalSec == 0 {
		if group.rtmpPubSession != nil {
			if readAlive, _ := group.rtmpPubSession.IsAlive(); !readAlive {
				Log.Warnf("[%s] session timeout. session=%s", group.UniqueKey, group.rtmpPubSession.UniqueKey())
				group.rtmpPubSession.Dispose()
				group.rtmp2RtspRemuxer = nil
			}
		}
		if group.rtspPubSession != nil {
			if readAlive, _ := group.rtspPubSession.IsAlive(); !readAlive {
				Log.Warnf("[%s] session timeout. session=%s", group.UniqueKey, group.rtspPubSession.UniqueKey())
				group.rtspPubSession.Dispose()
				group.rtspPubSession = nil
				group.rtsp2RtmpRemuxer = nil
			}
		}
		if group.pullProxy.pullSession != nil {
			if readAlive, _ := group.pullProxy.pullSession.IsAlive(); !readAlive {
				Log.Warnf("[%s] session timeout. session=%s", group.UniqueKey, group.pullProxy.pullSession.UniqueKey())
				group.pullProxy.pullSession.Dispose()
				group.delRtmpPullSession(group.pullProxy.pullSession)
			}
		}
		for session := range group.rtmpSubSessionSet {
			if _, writeAlive := session.IsAlive(); !writeAlive {
				Log.Warnf("[%s] session timeout. session=%s", group.UniqueKey, session.UniqueKey())
				session.Dispose()
				group.delRtmpSubSession(session)
			}
		}
		for session := range group.httpflvSubSessionSet {
			if _, writeAlive := session.IsAlive(); !writeAlive {
				Log.Warnf("[%s] session timeout. session=%s", group.UniqueKey, session.UniqueKey())
				session.Dispose()
				group.delHttpflvSubSession(session)
			}
		}
		for session := range group.httptsSubSessionSet {
			if _, writeAlive := session.IsAlive(); !writeAlive {
				Log.Warnf("[%s] session timeout. session=%s", group.UniqueKey, session.UniqueKey())
				session.Dispose()
				group.delHttptsSubSession(session)
			}
		}
		for session := range group.rtspSubSessionSet {
			if _, writeAlive := session.IsAlive(); !writeAlive {
				Log.Warnf("[%s] session timeout. session=%s", group.UniqueKey, session.UniqueKey())
				session.Dispose()
				group.delRtspSubSession(session)
			}
		}
	}

	// 定时计算session bitrate
	if group.tickCount%calcSessionStatIntervalSec == 0 {
		if group.rtmpPubSession != nil {
			group.rtmpPubSession.UpdateStat(calcSessionStatIntervalSec)
		}
		if group.rtspPubSession != nil {
			group.rtspPubSession.UpdateStat(calcSessionStatIntervalSec)
		}
		if group.pullProxy.pullSession != nil {
			group.pullProxy.pullSession.UpdateStat(calcSessionStatIntervalSec)
		}
		for session := range group.rtmpSubSessionSet {
			session.UpdateStat(calcSessionStatIntervalSec)
		}
		for session := range group.httpflvSubSessionSet {
			session.UpdateStat(calcSessionStatIntervalSec)
		}
		for session := range group.httptsSubSessionSet {
			session.UpdateStat(calcSessionStatIntervalSec)
		}
		for session := range group.rtspSubSessionSet {
			session.UpdateStat(calcSessionStatIntervalSec)
		}
	}

	group.tickCount++
}

// Dispose 主动释放所有资源。保证所有资源的生命周期逻辑上都在我们的控制中。降低出bug的几率，降低心智负担。
// 注意，Dispose后，不应再使用这个对象。
// 值得一提，如果是从其他协程回调回来的消息，在使用Group中的资源前，要判断资源是否存在以及可用。
//
func (group *Group) Dispose() {
	Log.Infof("[%s] lifecycle dispose group.", group.UniqueKey)
	group.exitChan <- struct{}{}

	group.mutex.Lock()
	defer group.mutex.Unlock()

	if group.rtmpPubSession != nil {
		group.rtmpPubSession.Dispose()
		group.rtmpPubSession = nil
		group.rtmp2RtspRemuxer = nil
	}
	if group.rtspPubSession != nil {
		group.rtspPubSession.Dispose()
		group.rtspPubSession = nil
		group.rtsp2RtmpRemuxer = nil
	}

	for session := range group.rtmpSubSessionSet {
		session.Dispose()
	}
	group.rtmpSubSessionSet = nil

	for session := range group.httpflvSubSessionSet {
		session.Dispose()
	}
	group.httpflvSubSessionSet = nil

	for session := range group.httptsSubSessionSet {
		session.Dispose()
	}
	group.httptsSubSessionSet = nil

	group.disposeHlsMuxer()

	if group.pushEnable {
		for _, v := range group.url2PushProxy {
			if v.pushSession != nil {
				v.pushSession.Dispose()
			}
		}
		group.url2PushProxy = nil
	}
}

// ---------------------------------------------------------------------------------------------------------------------

func (group *Group) AddRtmpSubSession(session *rtmp.ServerSession) {
	Log.Debugf("[%s] [%s] add SubSession into group.", group.UniqueKey, session.UniqueKey())
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.rtmpSubSessionSet[session] = struct{}{}
	// 加入时，如果上行还没有推过视频（比如还没推流，或者是单音频流），就不需要等待关键帧了
	// 也即我们假定上行肯定是以关键帧为开始进行视频发送，假设不是，那么我们按上行的流正常发，而不过滤掉关键帧前面的不包含关键帧的非完整GOP
	// TODO(chef):
	//   1. 需要仔细考虑单音频无视频的流的情况
	//   2. 这里不修改标志，让这个session继续等关键帧也可以
	if group.stat.VideoCodec == "" {
		session.ShouldWaitVideoKeyFrame = false
	}

	group.pullIfNeeded()
}

func (group *Group) AddHttpflvSubSession(session *httpflv.SubSession) {
	Log.Debugf("[%s] [%s] add httpflv SubSession into group.", group.UniqueKey, session.UniqueKey())
	session.WriteHttpResponseHeader()
	session.WriteFlvHeader()

	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.httpflvSubSessionSet[session] = struct{}{}
	// 加入时，如果上行还没有推流过，就不需要等待关键帧了
	if group.stat.VideoCodec == "" {
		session.ShouldWaitVideoKeyFrame = false
	}

	group.pullIfNeeded()
}

// AddHttptsSubSession TODO chef:
//   这里应该也要考虑触发hls muxer开启
//   也即HTTPTS sub需要使用hls muxer，hls muxer开启和关闭都要考虑HTTPTS sub
func (group *Group) AddHttptsSubSession(session *httpts.SubSession) {
	Log.Debugf("[%s] [%s] add httpts SubSession into group.", group.UniqueKey, session.UniqueKey())
	session.WriteHttpResponseHeader()

	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.httptsSubSessionSet[session] = struct{}{}

	group.pullIfNeeded()
}

func (group *Group) HandleNewRtspSubSessionDescribe(session *rtsp.SubSession) (ok bool, sdp []byte) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	// TODO(chef): 应该有等待机制，而不是直接关闭
	if group.sdpCtx == nil {
		Log.Warnf("[%s] close rtsp subSession while describe but sdp not exist. [%s]",
			group.UniqueKey, session.UniqueKey())
		return false, nil
	}

	return true, group.sdpCtx.RawSdp
}

func (group *Group) HandleNewRtspSubSessionPlay(session *rtsp.SubSession) {
	Log.Debugf("[%s] [%s] add rtsp SubSession into group.", group.UniqueKey, session.UniqueKey())

	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.rtspSubSessionSet[session] = struct{}{}
	if group.stat.VideoCodec == "" {
		session.ShouldWaitVideoKeyFrame = false
	}

	// TODO(chef): rtsp sub也应该判断是否需要静态pull回源
}

func (group *Group) AddRtmpPushSession(url string, session *rtmp.PushSession) {
	Log.Debugf("[%s] [%s] add rtmp PushSession into group.", group.UniqueKey, session.UniqueKey())
	group.mutex.Lock()
	defer group.mutex.Unlock()
	if group.url2PushProxy != nil {
		group.url2PushProxy[url].pushSession = session
	}
}

func (group *Group) DelRtmpSubSession(session *rtmp.ServerSession) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.delRtmpSubSession(session)
}

func (group *Group) DelHttpflvSubSession(session *httpflv.SubSession) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.delHttpflvSubSession(session)
}

func (group *Group) DelHttptsSubSession(session *httpts.SubSession) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.delHttptsSubSession(session)
}

func (group *Group) DelRtspSubSession(session *rtsp.SubSession) {
	Log.Debugf("[%s] [%s] del rtsp SubSession from group.", group.UniqueKey, session.UniqueKey())
	group.mutex.Lock()
	defer group.mutex.Unlock()
	delete(group.rtspSubSessionSet, session)
}

func (group *Group) DelRtmpPushSession(url string, session *rtmp.PushSession) {
	Log.Debugf("[%s] [%s] del rtmp PushSession into group.", group.UniqueKey, session.UniqueKey())
	group.mutex.Lock()
	defer group.mutex.Unlock()
	if group.url2PushProxy != nil {
		group.url2PushProxy[url].pushSession = nil
		group.url2PushProxy[url].isPushing = false
	}
}

// ---------------------------------------------------------------------------------------------------------------------

func (group *Group) IsTotalEmpty() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.isTotalEmpty()
}

func (group *Group) HasInSession() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.hasInSession()
}

func (group *Group) HasOutSession() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.hasOutSession()
}

func (group *Group) IsHlsMuxerAlive() bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	return group.hlsMuxer != nil
}

func (group *Group) StringifyDebugStats(maxsub int) string {
	b, _ := json.Marshal(group.GetStat(maxsub))
	return string(b)
}

func (group *Group) GetStat(maxsub int) base.StatGroup {
	// TODO(chef): [refactor] param maxsub

	group.mutex.Lock()
	defer group.mutex.Unlock()

	if group.rtmpPubSession != nil {
		group.stat.StatPub = base.StatSession2Pub(group.rtmpPubSession.GetStat())
	} else if group.rtspPubSession != nil {
		group.stat.StatPub = base.StatSession2Pub(group.rtspPubSession.GetStat())
	} else {
		group.stat.StatPub = base.StatPub{}
	}

	if group.pullProxy.pullSession != nil {
		group.stat.StatPull = base.StatSession2Pull(group.pullProxy.pullSession.GetStat())
	}

	group.stat.StatSubs = nil
	var statSubCount int
	for s := range group.rtmpSubSessionSet {
		statSubCount++
		if statSubCount > maxsub {
			break
		}
		group.stat.StatSubs = append(group.stat.StatSubs, base.StatSession2Sub(s.GetStat()))
	}
	for s := range group.httpflvSubSessionSet {
		statSubCount++
		if statSubCount > maxsub {
			break
		}
		group.stat.StatSubs = append(group.stat.StatSubs, base.StatSession2Sub(s.GetStat()))
	}
	for s := range group.httptsSubSessionSet {
		statSubCount++
		if statSubCount > maxsub {
			break
		}
		group.stat.StatSubs = append(group.stat.StatSubs, base.StatSession2Sub(s.GetStat()))
	}
	for s := range group.rtspSubSessionSet {
		statSubCount++
		if statSubCount > maxsub {
			break
		}
		group.stat.StatSubs = append(group.stat.StatSubs, base.StatSession2Sub(s.GetStat()))
	}

	return group.stat
}

func (group *Group) KickOutSession(sessionId string) bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	Log.Infof("[%s] kick out session. session id=%s", group.UniqueKey, sessionId)

	if strings.HasPrefix(sessionId, base.UkPreRtmpServerSession) {
		if group.rtmpPubSession != nil {
			group.rtmpPubSession.Dispose()
			group.rtmp2RtspRemuxer = nil
			return true
		}
	} else if strings.HasPrefix(sessionId, base.UkPreRtspPubSession) {
		if group.rtspPubSession != nil {
			group.rtspPubSession.Dispose()
			group.rtsp2RtmpRemuxer = nil
			return true
		}
	} else if strings.HasPrefix(sessionId, base.UkPreFlvSubSession) {
		// TODO chef: 考虑数据结构改成sessionIdzuokey的map
		for s := range group.httpflvSubSessionSet {
			if s.UniqueKey() == sessionId {
				s.Dispose()
				return true
			}
		}
	} else if strings.HasPrefix(sessionId, base.UkPreTsSubSession) {
		for s := range group.httptsSubSessionSet {
			if s.UniqueKey() == sessionId {
				s.Dispose()
				return true
			}
		}
	} else if strings.HasPrefix(sessionId, base.UkPreRtspSubSession) {
		// TODO chef: impl me
	} else {
		Log.Errorf("[%s] kick out session while session id format invalid. %s", group.UniqueKey, sessionId)
	}

	return false
}

// StartPull 外部命令主动触发pull拉流
//
// 当前调用时机：
// 1. 比如http api
//
func (group *Group) StartPull(url string) {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	group.setPullUrl(true, url)
	group.pullIfNeeded()
}

// ---------------------------------------------------------------------------------------------------------------------

func (group *Group) delRtmpSubSession(session *rtmp.ServerSession) {
	Log.Debugf("[%s] [%s] del rtmp SubSession from group.", group.UniqueKey, session.UniqueKey())
	delete(group.rtmpSubSessionSet, session)
}

func (group *Group) delHttpflvSubSession(session *httpflv.SubSession) {
	Log.Debugf("[%s] [%s] del httpflv SubSession from group.", group.UniqueKey, session.UniqueKey())
	delete(group.httpflvSubSessionSet, session)
}

func (group *Group) delHttptsSubSession(session *httpts.SubSession) {
	Log.Debugf("[%s] [%s] del httpts SubSession from group.", group.UniqueKey, session.UniqueKey())
	delete(group.httptsSubSessionSet, session)
}

func (group *Group) delRtspSubSession(session *rtsp.SubSession) {
	Log.Debugf("[%s] [%s] del rtsp SubSession from group.", group.UniqueKey, session.UniqueKey())
	delete(group.rtspSubSessionSet, session)
}

func (group *Group) pushIfNeeded() {
	// push转推功能没开
	if !group.pushEnable {
		return
	}
	// 没有pub发布者
	if group.rtmpPubSession == nil && group.rtspPubSession == nil {
		return
	}

	// relay push时携带rtmp pub的参数
	// TODO chef: 这个逻辑放这里不太好看
	var urlParam string
	if group.rtmpPubSession != nil {
		urlParam = group.rtmpPubSession.RawQuery()
	}

	for url, v := range group.url2PushProxy {
		// 正在转推中
		if v.isPushing {
			continue
		}
		v.isPushing = true

		urlWithParam := url
		if urlParam != "" {
			urlWithParam += "?" + urlParam
		}
		Log.Infof("[%s] start relay push. url=%s", group.UniqueKey, urlWithParam)

		go func(u, u2 string) {
			pushSession := rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
				option.PushTimeoutMs = relayPushTimeoutMs
				option.WriteAvTimeoutMs = relayPushWriteAvTimeoutMs
			})
			err := pushSession.Push(u2)
			if err != nil {
				Log.Errorf("[%s] relay push done. err=%v", pushSession.UniqueKey(), err)
				group.DelRtmpPushSession(u, pushSession)
				return
			}
			group.AddRtmpPushSession(u, pushSession)
			err = <-pushSession.WaitChan()
			Log.Infof("[%s] relay push done. err=%v", pushSession.UniqueKey(), err)
			group.DelRtmpPushSession(u, pushSession)
		}(url, urlWithParam)
	}
}

func (group *Group) hasPushSession() bool {
	for _, item := range group.url2PushProxy {
		if item.isPushing || item.pushSession != nil {
			return true
		}
	}
	return false
}

func (group *Group) hasInSession() bool {
	return group.rtmpPubSession != nil ||
		group.rtspPubSession != nil ||
		group.pullProxy.pullSession != nil
}

// 是否还有out往外发送音视频数据的session，目前判断所有协议类型的sub session
//
func (group *Group) hasOutSession() bool {
	return len(group.rtmpSubSessionSet) != 0 ||
		len(group.httpflvSubSessionSet) != 0 ||
		len(group.httptsSubSessionSet) != 0 ||
		len(group.rtspSubSessionSet) != 0
}

// 当前group是否完全没有流了
//
func (group *Group) isTotalEmpty() bool {
	// TODO(chef): 是否应该只判断pub、sub、pull还是所有包括录制在内的都判断
	return !group.hasInSession() &&
		!group.hasOutSession() &&
		group.hlsMuxer == nil &&
		!group.hasPushSession()
}

func (group *Group) disposeHlsMuxer() {
	if group.hlsMuxer != nil {
		group.hlsMuxer.Dispose()

		group.observer.CleanupHlsIfNeeded(group.appName, group.streamName, group.hlsMuxer.OutPath())

		group.hlsMuxer = nil
	}
}

// ----- relay pull ----------------------------------------------------------------------------------------------------

func (group *Group) isPullEnable() bool {
	return group.pullEnable
}

func (group *Group) setPullUrl(enable bool, url string) {
	group.pullEnable = enable
	group.pullUrl = url
}

func (group *Group) getPullUrl() string {
	return group.pullUrl
}

func (group *Group) setPullingFlag(flag bool) {
	group.pullProxy.isPulling = flag
}

func (group *Group) getPullingFlag() bool {
	return group.pullProxy.isPulling
}

// 判断是否需要pull从远端拉流至本地，如果需要，则触发pull
//
// 当前调用时机：
// 1. 添加新sub session
// 2. 外部命令，比如http api
// 3. 定时器，比如pull的连接断了，通过定时器可以重启触发pull
//
func (group *Group) pullIfNeeded() {
	if !group.isPullEnable() {
		return
	}
	// 如果没有从本地拉流的，就不需要pull了
	if !group.hasOutSession() {
		return
	}
	// 如果本地已经有输入型的流，就不需要pull了
	if group.hasInSession() {
		return
	}
	// 已经在pull中，就不需要pull了
	if group.getPullingFlag() {
		return
	}
	group.setPullingFlag(true)

	Log.Infof("[%s] start relay pull. url=%s", group.UniqueKey, group.getPullUrl())

	go func() {
		pullSession := rtmp.NewPullSession(func(option *rtmp.PullSessionOption) {
			option.PullTimeoutMs = relayPullTimeoutMs
			option.ReadAvTimeoutMs = relayPullReadAvTimeoutMs
		})
		// TODO(chef): 处理数据回调，是否应该等待Add成功之后。避免竞态条件中途加入了其他in session
		err := pullSession.Pull(group.getPullUrl(), group.OnReadRtmpAvMsg)
		if err != nil {
			Log.Errorf("[%s] relay pull fail. err=%v", pullSession.UniqueKey(), err)
			group.DelRtmpPullSession(pullSession)
			return
		}
		res := group.AddRtmpPullSession(pullSession)
		if res {
			err = <-pullSession.WaitChan()
			Log.Infof("[%s] relay pull done. err=%v", pullSession.UniqueKey(), err)
			group.DelRtmpPullSession(pullSession)
		} else {
			pullSession.Dispose()
		}
	}()
}

// 判断是否需要停止pull
//
// 当前调用时机：
// 1. 定时器定时检查
//
func (group *Group) stopPullIfNeeded() {
	// 没有输出型的流了
	if group.pullProxy.pullSession != nil && !group.hasOutSession() {
		Log.Infof("[%s] stop pull since no sub session.", group.UniqueKey)
		group.pullProxy.pullSession.Dispose()
	}
}
