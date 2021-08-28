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
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/q191201771/lal/pkg/sdp"

	"github.com/q191201771/lal/pkg/mpegts"

	"github.com/q191201771/lal/pkg/remux"

	"github.com/q191201771/naza/pkg/defertaskthread"

	"github.com/q191201771/lal/pkg/rtprtcp"

	"github.com/q191201771/lal/pkg/hevc"

	"github.com/q191201771/lal/pkg/httpts"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/avc"
	"github.com/q191201771/lal/pkg/rtsp"

	"github.com/q191201771/lal/pkg/hls"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
)

type Group struct {
	UniqueKey  string // const after init
	appName    string // const after init
	streamName string // const after init TODO chef: 和stat里的字段重复，可以删除掉

	exitChan chan struct{}

	mutex sync.Mutex
	//
	stat base.StatGroup
	// pub
	rtmpPubSession   *rtmp.ServerSession
	rtspPubSession   *rtsp.PubSession
	rtsp2RtmpRemuxer *remux.AvPacket2RtmpRemuxer
	rtmp2RtspRemuxer *remux.Rtmp2RtspRemuxer
	// pull
	pullEnable bool
	pullUrl    string
	pullProxy  *pullProxy
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
	// rtmp pub/pull使用
	rtmpGopCache    *GopCache
	httpflvGopCache *GopCache
	// rtmp pub使用
	dummyAudioFilter *DummyAudioFilter
	// rtmp sub使用
	rtmpBufWriter base.IBufWriter // TODO(chef): 后面可以在业务层加一个定时Flush
	// mpegts使用
	patpmt []byte
	// rtsp使用
	sdpCtx *sdp.LogicContext
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

func NewGroup(appName string, streamName string) *Group {
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
		stat: base.StatGroup{
			StreamName: streamName,
		},
		exitChan:             make(chan struct{}, 1),
		rtmpSubSessionSet:    make(map[*rtmp.ServerSession]struct{}),
		httpflvSubSessionSet: make(map[*httpflv.SubSession]struct{}),
		httptsSubSessionSet:  make(map[*httpts.SubSession]struct{}),
		rtspSubSessionSet:    make(map[*rtsp.SubSession]struct{}),
		rtmpGopCache:         NewGopCache("rtmp", uk, config.RtmpConfig.GopNum),
		httpflvGopCache:      NewGopCache("httpflv", uk, config.HttpflvConfig.GopNum),
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

	g.rtmpBufWriter = base.NewWriterFuncSize(g.write2RtmpSubSessions, config.RtmpConfig.MergeWriteSize)
	nazalog.Infof("[%s] lifecycle new group. group=%p, appName=%s, streamName=%s", uk, g, appName, streamName)

	return g
}

func (group *Group) RunLoop() {
	<-group.exitChan
}

// TODO chef: 传入时间
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

	// 定时关闭没有数据的session
	if group.tickCount%checkSessionAliveIntervalSec == 0 {
		if group.rtmpPubSession != nil {
			if readAlive, _ := group.rtmpPubSession.IsAlive(); !readAlive {
				nazalog.Warnf("[%s] session timeout. session=%s", group.UniqueKey, group.rtmpPubSession.UniqueKey())
				group.rtmpPubSession.Dispose()
				group.rtmp2RtspRemuxer = nil
			}
		}
		if group.rtspPubSession != nil {
			if readAlive, _ := group.rtspPubSession.IsAlive(); !readAlive {
				nazalog.Warnf("[%s] session timeout. session=%s", group.UniqueKey, group.rtspPubSession.UniqueKey())
				group.rtspPubSession.Dispose()
				group.rtspPubSession = nil
				group.rtsp2RtmpRemuxer = nil
			}
		}
		if group.pullProxy.pullSession != nil {
			if readAlive, _ := group.pullProxy.pullSession.IsAlive(); !readAlive {
				nazalog.Warnf("[%s] session timeout. session=%s", group.UniqueKey, group.pullProxy.pullSession.UniqueKey())
				group.pullProxy.pullSession.Dispose()
				group.delRtmpPullSession(group.pullProxy.pullSession)
			}
		}
		for session := range group.rtmpSubSessionSet {
			if _, writeAlive := session.IsAlive(); !writeAlive {
				nazalog.Warnf("[%s] session timeout. session=%s", group.UniqueKey, session.UniqueKey())
				session.Dispose()
				group.delRtmpSubSession(session)
			}
		}
		for session := range group.httpflvSubSessionSet {
			if _, writeAlive := session.IsAlive(); !writeAlive {
				nazalog.Warnf("[%s] session timeout. session=%s", group.UniqueKey, session.UniqueKey())
				session.Dispose()
				group.delHttpflvSubSession(session)
			}
		}
		for session := range group.httptsSubSessionSet {
			if _, writeAlive := session.IsAlive(); !writeAlive {
				nazalog.Warnf("[%s] session timeout. session=%s", group.UniqueKey, session.UniqueKey())
				session.Dispose()
				group.delHttptsSubSession(session)
			}
		}
		for session := range group.rtspSubSessionSet {
			if _, writeAlive := session.IsAlive(); !writeAlive {
				nazalog.Warnf("[%s] session timeout. session=%s", group.UniqueKey, session.UniqueKey())
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

// 主动释放所有资源。保证所有资源的生命周期逻辑上都在我们的控制中。降低出bug的几率，降低心智负担。
// 注意，Dispose后，不应再使用这个对象。
// 值得一提，如果是从其他协程回调回来的消息，在使用Group中的资源前，要判断资源是否存在以及可用。
//
func (group *Group) Dispose() {
	nazalog.Infof("[%s] lifecycle dispose group.", group.UniqueKey)
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

func (group *Group) AddRtmpPubSession(session *rtmp.ServerSession) bool {
	nazalog.Debugf("[%s] [%s] add PubSession into group.", group.UniqueKey, session.UniqueKey())

	group.mutex.Lock()
	defer group.mutex.Unlock()

	if group.hasInSession() {
		nazalog.Errorf("[%s] in stream already exist. wanna add=%s", group.UniqueKey, session.UniqueKey())
		return false
	}

	group.rtmpPubSession = session
	group.addIn()

	if config.RtspConfig.Enable {
		group.rtmp2RtspRemuxer = remux.NewRtmp2RtspRemuxer(
			func(sdpCtx sdp.LogicContext) {
				group.sdpCtx = &sdpCtx
			},
			group.onRtpPacket,
		)
	}

	// TODO(chef): 为rtmp pull以及rtsp也添加叠加静音音频的功能
	if config.RtmpConfig.AddDummyAudioEnable {
		// TODO(chef): 从整体控制和锁关系来说，应该让pub的数据回调到group中进锁后再让数据流入filter
		group.dummyAudioFilter = NewDummyAudioFilter(group.UniqueKey, config.RtmpConfig.AddDummyAudioWaitAudioMs, group.OnReadRtmpAvMsg)
		session.SetPubSessionObserver(group.dummyAudioFilter)
	} else {
		session.SetPubSessionObserver(group)
	}

	return true
}

// TODO chef: rtsp package中，增加回调返回值判断，如果是false，将连接关掉
func (group *Group) AddRtspPubSession(session *rtsp.PubSession) bool {
	nazalog.Debugf("[%s] [%s] add RTSP PubSession into group.", group.UniqueKey, session.UniqueKey())

	group.mutex.Lock()
	defer group.mutex.Unlock()

	if group.hasInSession() {
		nazalog.Errorf("[%s] in stream already exist. wanna add=%s", group.UniqueKey, session.UniqueKey())
		return false
	}

	group.rtspPubSession = session
	group.addIn()

	group.rtsp2RtmpRemuxer = remux.NewAvPacket2RtmpRemuxer(func(msg base.RtmpMsg) {
		group.broadcastByRtmpMsg(msg)
	})
	session.SetObserver(group)

	return true
}

func (group *Group) AddRtmpPullSession(session *rtmp.PullSession) bool {
	nazalog.Debugf("[%s] [%s] add PullSession into group.", group.UniqueKey, session.UniqueKey())

	group.mutex.Lock()
	defer group.mutex.Unlock()

	if group.hasInSession() {
		nazalog.Errorf("[%s] in stream already exist. wanna add=%s", group.UniqueKey, session.UniqueKey())
		return false
	}

	group.pullProxy.pullSession = session
	group.addIn()

	// TODO(chef): 这里也应该启动rtmp2RtspRemuxer

	return true
}

func (group *Group) DelRtmpPubSession(session *rtmp.ServerSession) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.delRtmpPubSession(session)
}

func (group *Group) DelRtspPubSession(session *rtsp.PubSession) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.delRtspPubSession(session)
}

func (group *Group) DelRtmpPullSession(session *rtmp.PullSession) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.delRtmpPullSession(session)
}

// ---------------------------------------------------------------------------------------------------------------------

func (group *Group) AddRtmpSubSession(session *rtmp.ServerSession) {
	nazalog.Debugf("[%s] [%s] add SubSession into group.", group.UniqueKey, session.UniqueKey())
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
	nazalog.Debugf("[%s] [%s] add httpflv SubSession into group.", group.UniqueKey, session.UniqueKey())
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

// TODO chef:
//   这里应该也要考虑触发hls muxer开启
//   也即HTTPTS sub需要使用hls muxer，hls muxer开启和关闭都要考虑HTTPTS sub
func (group *Group) AddHttptsSubSession(session *httpts.SubSession) {
	nazalog.Debugf("[%s] [%s] add httpts SubSession into group.", group.UniqueKey, session.UniqueKey())
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
		nazalog.Warnf("[%s] close rtsp subSession while describe but sdp not exist. [%s]",
			group.UniqueKey, session.UniqueKey())
		return false, nil
	}

	return true, group.sdpCtx.RawSdp
}

func (group *Group) HandleNewRtspSubSessionPlay(session *rtsp.SubSession) bool {
	nazalog.Debugf("[%s] [%s] add rtsp SubSession into group.", group.UniqueKey, session.UniqueKey())

	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.rtspSubSessionSet[session] = struct{}{}
	if group.stat.VideoCodec == "" {
		session.ShouldWaitVideoKeyFrame = false
	}

	// TODO(chef): rtsp sub也应该判断是否需要静态pull回源

	return true
}

func (group *Group) AddRtmpPushSession(url string, session *rtmp.PushSession) {
	nazalog.Debugf("[%s] [%s] add rtmp PushSession into group.", group.UniqueKey, session.UniqueKey())
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
	nazalog.Debugf("[%s] [%s] del rtsp SubSession from group.", group.UniqueKey, session.UniqueKey())
	group.mutex.Lock()
	defer group.mutex.Unlock()
	delete(group.rtspSubSessionSet, session)
}

func (group *Group) DelRtmpPushSession(url string, session *rtmp.PushSession) {
	nazalog.Debugf("[%s] [%s] del rtmp PushSession into group.", group.UniqueKey, session.UniqueKey())
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

func (group *Group) StringifyDebugStats() string {
	group.mutex.Lock()
	subLen := len(group.rtmpSubSessionSet) + len(group.httpflvSubSessionSet) + len(group.httptsSubSessionSet) + len(group.rtspSubSessionSet)
	group.mutex.Unlock()
	if subLen > 10 {
		return fmt.Sprintf("[%s] not log out all stats. subLen=%d", group.UniqueKey, subLen)
	}
	b, _ := json.Marshal(group.GetStat())
	return string(b)
}

func (group *Group) GetStat() base.StatGroup {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	if group.rtmpPubSession != nil {
		group.stat.StatPub = base.StatSession2Pub(group.rtmpPubSession.GetStat())
	} else if group.rtspPubSession != nil {
		group.stat.StatPub = base.StatSession2Pub(group.rtspPubSession.GetStat())
	} else {
		group.stat.StatPub = base.StatPub{}
	}

	group.stat.StatSubs = nil
	for s := range group.rtmpSubSessionSet {
		group.stat.StatSubs = append(group.stat.StatSubs, base.StatSession2Sub(s.GetStat()))
	}
	for s := range group.httpflvSubSessionSet {
		group.stat.StatSubs = append(group.stat.StatSubs, base.StatSession2Sub(s.GetStat()))
	}
	for s := range group.httptsSubSessionSet {
		group.stat.StatSubs = append(group.stat.StatSubs, base.StatSession2Sub(s.GetStat()))
	}
	for s := range group.rtspSubSessionSet {
		group.stat.StatSubs = append(group.stat.StatSubs, base.StatSession2Sub(s.GetStat()))
	}

	if group.pullProxy.pullSession != nil {
		group.stat.StatPull = base.StatSession2Pull(group.pullProxy.pullSession.GetStat())
	}

	return group.stat
}

func (group *Group) KickOutSession(sessionId string) bool {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	nazalog.Infof("[%s] kick out session. session id=%s", group.UniqueKey, sessionId)

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
		nazalog.Errorf("[%s] kick out session while session id format invalid. %s", group.UniqueKey, sessionId)
	}

	return false
}

// 外部命令主动触发pull拉流
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

func (group *Group) delRtmpPubSession(session *rtmp.ServerSession) {
	nazalog.Debugf("[%s] [%s] del rtmp PubSession from group.", group.UniqueKey, session.UniqueKey())

	if session != group.rtmpPubSession {
		nazalog.Warnf("[%s] del rtmp pub session but not match. del session=%s, group session=%p", group.UniqueKey, session.UniqueKey(), group.rtmpPubSession)
		return
	}

	group.rtmpPubSession = nil
	group.rtmp2RtspRemuxer = nil
	group.dummyAudioFilter = nil
	group.delIn()
}

func (group *Group) delRtspPubSession(session *rtsp.PubSession) {
	nazalog.Debugf("[%s] [%s] del rtsp PubSession from group.", group.UniqueKey, session.UniqueKey())

	if session != group.rtspPubSession {
		nazalog.Warnf("[%s] del rtmp pub session but not match. del session=%s, group session=%p", group.UniqueKey, session.UniqueKey(), group.rtmpPubSession)
		return
	}

	_ = group.rtspPubSession.Dispose()
	group.rtspPubSession = nil
	group.rtsp2RtmpRemuxer = nil
	group.delIn()
}

func (group *Group) delRtmpPullSession(session *rtmp.PullSession) {
	nazalog.Debugf("[%s] [%s] del rtmp PullSession from group.", group.UniqueKey, session.UniqueKey())

	group.pullProxy.pullSession = nil
	group.setPullingFlag(false)
	group.delIn()
}

func (group *Group) delRtmpSubSession(session *rtmp.ServerSession) {
	nazalog.Debugf("[%s] [%s] del rtmp SubSession from group.", group.UniqueKey, session.UniqueKey())
	delete(group.rtmpSubSessionSet, session)
}

func (group *Group) delHttpflvSubSession(session *httpflv.SubSession) {
	nazalog.Debugf("[%s] [%s] del httpflv SubSession from group.", group.UniqueKey, session.UniqueKey())
	delete(group.httpflvSubSessionSet, session)
}

func (group *Group) delHttptsSubSession(session *httpts.SubSession) {
	nazalog.Debugf("[%s] [%s] del httpts SubSession from group.", group.UniqueKey, session.UniqueKey())
	delete(group.httptsSubSessionSet, session)
}

func (group *Group) delRtspSubSession(session *rtsp.SubSession) {
	nazalog.Debugf("[%s] [%s] del rtsp SubSession from group.", group.UniqueKey, session.UniqueKey())
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
		nazalog.Infof("[%s] start relay push. url=%s", group.UniqueKey, urlWithParam)

		go func(u, u2 string) {
			pushSession := rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
				option.PushTimeoutMs = relayPushTimeoutMs
				option.WriteAvTimeoutMs = relayPushWriteAvTimeoutMs
			})
			err := pushSession.Push(u2)
			if err != nil {
				nazalog.Errorf("[%s] relay push done. err=%v", pushSession.UniqueKey(), err)
				group.DelRtmpPushSession(u, pushSession)
				return
			}
			group.AddRtmpPushSession(u, pushSession)
			err = <-pushSession.WaitChan()
			nazalog.Infof("[%s] relay push done. err=%v", pushSession.UniqueKey(), err)
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

// 有pub或pull的in session加入时，需要调用该函数
//
func (group *Group) addIn() {
	// 是否启动hls
	if config.HlsConfig.Enable {
		if group.hlsMuxer != nil {
			nazalog.Errorf("[%s] hls muxer exist while addIn. muxer=%+v", group.UniqueKey, group.hlsMuxer)
		}
		enable := config.HlsConfig.Enable || config.HlsConfig.EnableHttps
		group.hlsMuxer = hls.NewMuxer(group.streamName, enable, &config.HlsConfig.MuxerConfig, group)
		group.hlsMuxer.Start()
	}

	// 是否push转推
	if group.pushEnable {
		group.pushIfNeeded()
	}

	now := time.Now().Unix()

	// 是否录制成flv文件
	if config.RecordConfig.EnableFlv {
		filename := fmt.Sprintf("%s-%d.flv", group.streamName, now)
		filenameWithPath := filepath.Join(config.RecordConfig.FlvOutPath, filename)
		if group.recordFlv != nil {
			nazalog.Errorf("[%s] record flv but already exist. new filename=%s, old filename=%s",
				group.UniqueKey, filenameWithPath, group.recordFlv.Name())
			if err := group.recordFlv.Dispose(); err != nil {
				nazalog.Errorf("[%s] record flv dispose error. err=%+v", group.UniqueKey, err)
			}
		}
		group.recordFlv = &httpflv.FlvFileWriter{}
		if err := group.recordFlv.Open(filenameWithPath); err != nil {
			nazalog.Errorf("[%s] record flv open file failed. filename=%s, err=%+v",
				group.UniqueKey, filenameWithPath, err)
			group.recordFlv = nil
		}
		if err := group.recordFlv.WriteFlvHeader(); err != nil {
			nazalog.Errorf("[%s] record flv write flv header failed. filename=%s, err=%+v",
				group.UniqueKey, filenameWithPath, err)
			group.recordFlv = nil
		}
	}

	// 是否录制成ts文件
	if config.RecordConfig.EnableMpegts {
		filename := fmt.Sprintf("%s-%d.ts", group.streamName, now)
		filenameWithPath := filepath.Join(config.RecordConfig.MpegtsOutPath, filename)
		if group.recordMpegts != nil {
			nazalog.Errorf("[%s] record mpegts but already exist. new filename=%s, old filename=%s",
				group.UniqueKey, filenameWithPath, group.recordMpegts.Name())
			if err := group.recordMpegts.Dispose(); err != nil {
				nazalog.Errorf("[%s] record mpegts dispose error. err=%+v", group.UniqueKey, err)
			}
		}
		group.recordMpegts = &mpegts.FileWriter{}
		if err := group.recordMpegts.Create(filenameWithPath); err != nil {
			nazalog.Errorf("[%s] record mpegts open file failed. filename=%s, err=%+v",
				group.UniqueKey, filenameWithPath, err)
			group.recordFlv = nil
		}
	}
}

// 有pub或pull的in session离开时，需要调用该函数
//
func (group *Group) delIn() {
	// 停止hls
	if config.HlsConfig.Enable && group.hlsMuxer != nil {
		group.disposeHlsMuxer()
	}

	// 停止转推
	if group.pushEnable {
		for _, v := range group.url2PushProxy {
			if v.pushSession != nil {
				v.pushSession.Dispose()
			}
			v.pushSession = nil
		}
	}

	// 停止flv录制
	if config.RecordConfig.EnableFlv {
		if group.recordFlv != nil {
			if err := group.recordFlv.Dispose(); err != nil {
				nazalog.Errorf("[%s] record flv dispose error. err=%+v", group.UniqueKey, err)
			}
			group.recordFlv = nil
		}
	}

	// 停止ts录制
	if config.RecordConfig.EnableMpegts {
		if group.recordMpegts != nil {
			if err := group.recordMpegts.Dispose(); err != nil {
				nazalog.Errorf("[%s] record mpegts dispose error. err=%+v", group.UniqueKey, err)
			}
			group.recordMpegts = nil
		}
	}

	// 清理各种和in session相关的资源
	// TODO(chef) 清空rtsp pub缓存的asc sps pps等数据
	group.rtmpGopCache.Clear()
	group.httpflvGopCache.Clear()
	group.patpmt = nil
	group.sdpCtx = nil
}

func (group *Group) disposeHlsMuxer() {
	if group.hlsMuxer != nil {
		group.hlsMuxer.Dispose()

		// 添加延时任务，删除HLS文件
		if config.HlsConfig.Enable &&
			(config.HlsConfig.CleanupMode == hls.CleanupModeInTheEnd || config.HlsConfig.CleanupMode == hls.CleanupModeAsap) {
			defertaskthread.Go(
				config.HlsConfig.FragmentDurationMs*config.HlsConfig.FragmentNum*2,
				func(param ...interface{}) {
					appName := param[0].(string)
					streamName := param[1].(string)
					outPath := param[2].(string)

					if g := sm.GetGroup(appName, streamName); g != nil {
						if g.IsHlsMuxerAlive() {
							nazalog.Warnf("cancel cleanup hls file path since hls muxer still alive. streamName=%s", streamName)
							return
						}
					}

					nazalog.Infof("cleanup hls file path. streamName=%s, path=%s", streamName, outPath)
					if err := hls.RemoveAll(outPath); err != nil {
						nazalog.Warnf("cleanup hls file path error. path=%s, err=%+v", outPath, err)
					}
				},
				group.appName,
				group.streamName,
				group.hlsMuxer.OutPath(),
			)
		}

		group.hlsMuxer = nil
	}
}

// ---------------------------------------------------------------------------------------------------------------------
// 音视频数据转发、转封装的逻辑
// ---------------------------------------------------------------------------------------------------------------------

// rtmp.PubSession or rtmp.PullSession
func (group *Group) OnReadRtmpAvMsg(msg base.RtmpMsg) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	group.broadcastByRtmpMsg(msg)
}

// rtsp.PubSession
func (group *Group) OnRtpPacket(pkt rtprtcp.RtpPacket) {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	group.onRtpPacket(pkt)
}

// rtsp.PubSession
func (group *Group) OnSdp(sdpCtx sdp.LogicContext) {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	group.sdpCtx = &sdpCtx
	group.rtsp2RtmpRemuxer.OnSdp(sdpCtx)
}

// rtsp.PubSession
func (group *Group) OnAvPacket(pkt base.AvPacket) {
	group.mutex.Lock()
	defer group.mutex.Unlock()
	//nazalog.Debugf("[%s] > Group::OnAvPacket. type=%s, ts=%d, len=%d", group.UniqueKey, pkt.PayloadType.ReadableString(), pkt.Timestamp, len(pkt.Payload))

	group.rtsp2RtmpRemuxer.OnAvPacket(pkt)
}

// hls.Muxer
func (group *Group) OnPatPmt(b []byte) {
	group.patpmt = b

	if group.recordMpegts != nil {
		if err := group.recordMpegts.Write(b); err != nil {
			nazalog.Errorf("[%s] record mpegts write fragment header error. err=%+v", group.UniqueKey, err)
		}
	}
}

// hls.Muxer
func (group *Group) OnTsPackets(rawFrame []byte, boundary bool) {
	// 因为最前面Feed时已经加锁了，所以这里回调上来就不用加锁了

	for session := range group.httptsSubSessionSet {
		if session.IsFresh {
			if boundary {
				session.Write(group.patpmt)
				session.Write(rawFrame)
				session.IsFresh = false
			}
		} else {
			session.Write(rawFrame)
		}
	}

	if group.recordMpegts != nil {
		if err := group.recordMpegts.Write(rawFrame); err != nil {
			nazalog.Errorf("[%s] record mpegts write error. err=%+v", group.UniqueKey, err)
		}
	}
}

// TODO chef: 目前相当于其他类型往rtmp.AVMsg转了，考虑统一往一个通用类型转
//
// rtmp.PubSession, rtmp.PullSession, rtsp2rtmpRemuxer
//
// @param msg 调用结束后，内部不持有msg.Payload内存块
//
func (group *Group) broadcastByRtmpMsg(msg base.RtmpMsg) {
	var (
		lcd    LazyChunkDivider
		lrm2ft LazyRtmpMsg2FlvTag
	)

	//nazalog.Debugf("[%s] broadcaseRTMP. header=%+v, %s", group.UniqueKey, msg.Header, hex.Dump(nazastring.SubSliceSafety(msg.Payload, 7)))

	// # hls
	if config.HlsConfig.Enable && group.hlsMuxer != nil {
		group.hlsMuxer.FeedRtmpMessage(msg)
	}

	// # rtsp
	if config.RtspConfig.Enable && group.rtmp2RtspRemuxer != nil {
		group.rtmp2RtspRemuxer.FeedRtmpMsg(msg)
	}

	// # 设置好用于发送的 rtmp 头部信息
	currHeader := remux.MakeDefaultRtmpHeader(msg.Header)
	if currHeader.MsgLen != uint32(len(msg.Payload)) {
		nazalog.Errorf("[%s] diff. msgLen=%d, payload len=%d, %+v", group.UniqueKey, currHeader.MsgLen, len(msg.Payload), msg.Header)
	}

	// # 懒初始化rtmp chunk切片，以及httpflv转换
	lcd.Init(msg.Payload, &currHeader)
	lrm2ft.Init(msg)

	// # 广播。遍历所有 rtmp sub session，转发数据
	// ## 如果是新的 sub session，发送已缓存的信息
	for session := range group.rtmpSubSessionSet {
		if session.IsFresh {
			// TODO chef: 头信息和full gop也可以在SubSession刚加入时发送
			if group.rtmpGopCache.Metadata != nil {
				nazalog.Debugf("[%s] [%s] write metadata", group.UniqueKey, session.UniqueKey())
				_ = session.Write(group.rtmpGopCache.Metadata)
			}
			if group.rtmpGopCache.VideoSeqHeader != nil {
				nazalog.Debugf("[%s] [%s] write vsh", group.UniqueKey, session.UniqueKey())
				_ = session.Write(group.rtmpGopCache.VideoSeqHeader)
			}
			if group.rtmpGopCache.AacSeqHeader != nil {
				nazalog.Debugf("[%s] [%s] write ash", group.UniqueKey, session.UniqueKey())
				_ = session.Write(group.rtmpGopCache.AacSeqHeader)
			}
			gopCount := group.rtmpGopCache.GetGopCount()
			if gopCount > 0 {
				// GOP缓存中肯定包含了关键帧
				session.ShouldWaitVideoKeyFrame = false

				nazalog.Debugf("[%s] [%s] write gop cache. gop num=%d", group.UniqueKey, session.UniqueKey(), gopCount)
			}
			for i := 0; i < gopCount; i++ {
				for _, item := range group.rtmpGopCache.GetGopDataAt(i) {
					_ = session.Write(item)
				}
			}

			// 有新加入的sub session（本次循环的第一个新加入的sub session），把rtmp buf writer中的缓存数据全部广播发送给老的sub session
			// 从而确保新加入的sub session不会发送这部分脏的数据
			// 注意，此处可能被调用多次，但是只有第一次会实际flush缓存数据
			group.rtmpBufWriter.Flush()

			session.IsFresh = false
		}

		if session.ShouldWaitVideoKeyFrame && msg.IsVideoKeyNalu() {
			// 有sub session在等待关键帧，并且当前是关键帧
			// 把rtmp buf writer中的缓存数据全部广播发送给老的sub session
			// 并且修改这个sub session的标志
			// 让rtmp buf writer来发送这个关键帧
			group.rtmpBufWriter.Flush()
			session.ShouldWaitVideoKeyFrame = false
		}
	}
	// ## 转发本次数据
	if len(group.rtmpSubSessionSet) > 0 {
		group.rtmpBufWriter.Write(lcd.Get())
	}

	// TODO chef: rtmp sub, rtmp push, httpflv sub 的发送逻辑都差不多，可以考虑封装一下
	if group.pushEnable {
		for _, v := range group.url2PushProxy {
			if v.pushSession == nil {
				continue
			}

			if v.pushSession.IsFresh {
				if group.rtmpGopCache.Metadata != nil {
					_ = v.pushSession.Write(group.rtmpGopCache.Metadata)
				}
				if group.rtmpGopCache.VideoSeqHeader != nil {
					_ = v.pushSession.Write(group.rtmpGopCache.VideoSeqHeader)
				}
				if group.rtmpGopCache.AacSeqHeader != nil {
					_ = v.pushSession.Write(group.rtmpGopCache.AacSeqHeader)
				}
				for i := 0; i < group.rtmpGopCache.GetGopCount(); i++ {
					for _, item := range group.rtmpGopCache.GetGopDataAt(i) {
						_ = v.pushSession.Write(item)
					}
				}

				v.pushSession.IsFresh = false
			}

			_ = v.pushSession.Write(lcd.Get())
		}
	}

	// # 广播。遍历所有 httpflv sub session，转发数据
	for session := range group.httpflvSubSessionSet {
		if session.IsFresh {
			if group.httpflvGopCache.Metadata != nil {
				session.Write(group.httpflvGopCache.Metadata)
			}
			if group.httpflvGopCache.VideoSeqHeader != nil {
				session.Write(group.httpflvGopCache.VideoSeqHeader)
			}
			if group.httpflvGopCache.AacSeqHeader != nil {
				session.Write(group.httpflvGopCache.AacSeqHeader)
			}
			gopCount := group.httpflvGopCache.GetGopCount()
			if gopCount > 0 {
				// GOP缓存中肯定包含了关键帧
				session.ShouldWaitVideoKeyFrame = false
			}
			for i := 0; i < gopCount; i++ {
				for _, item := range group.httpflvGopCache.GetGopDataAt(i) {
					session.Write(item)
				}
			}

			session.IsFresh = false
		}

		// 是否在等待关键帧
		if session.ShouldWaitVideoKeyFrame {
			if msg.IsVideoKeyNalu() {
				session.Write(lrm2ft.Get())
				session.ShouldWaitVideoKeyFrame = false
			}
		} else {
			session.Write(lrm2ft.Get())
		}
	}

	// # 录制flv文件
	if group.recordFlv != nil {
		if err := group.recordFlv.WriteRaw(lrm2ft.Get()); err != nil {
			nazalog.Errorf("[%s] record flv write error. err=%+v", group.UniqueKey, err)
		}
	}

	// # 缓存关键信息，以及gop
	if config.RtmpConfig.Enable {
		group.rtmpGopCache.Feed(msg, lcd.Get)
	}
	if config.HttpflvConfig.Enable {
		group.httpflvGopCache.Feed(msg, lrm2ft.Get)
	}

	// # 记录stat
	if group.stat.AudioCodec == "" {
		if msg.IsAacSeqHeader() {
			group.stat.AudioCodec = base.AudioCodecAac
		}
	}
	if group.stat.VideoCodec == "" {
		if msg.IsAvcKeySeqHeader() {
			group.stat.VideoCodec = base.VideoCodecAvc
		}
		if msg.IsHevcKeySeqHeader() {
			group.stat.VideoCodec = base.VideoCodecHevc
		}
	}
	if group.stat.VideoHeight == 0 || group.stat.VideoWidth == 0 {
		if msg.IsAvcKeySeqHeader() {
			sps, _, err := avc.ParseSpsPpsFromSeqHeader(msg.Payload)
			if err == nil {
				var ctx avc.Context
				err = avc.ParseSps(sps, &ctx)
				if err == nil {
					group.stat.VideoHeight = int(ctx.Height)
					group.stat.VideoWidth = int(ctx.Width)
				}
			}
		}
		if msg.IsHevcKeySeqHeader() {
			_, sps, _, err := hevc.ParseVpsSpsPpsFromSeqHeader(msg.Payload)
			if err == nil {
				var ctx hevc.Context
				err = hevc.ParseSps(sps, &ctx)
				if err == nil {
					group.stat.VideoHeight = int(ctx.PicHeightInLumaSamples)
					group.stat.VideoWidth = int(ctx.PicWidthInLumaSamples)
				}
			}
		}
	}
}

// rtsp.PubSession, rtmp2RtspRemuxer
//
func (group *Group) onRtpPacket(pkt rtprtcp.RtpPacket) {
	// 音频直接发送
	if group.sdpCtx.IsAudioPayloadTypeOrigin(int(pkt.Header.PacketType)) {
		for s := range group.rtspSubSessionSet {
			s.WriteRtpPacket(pkt)
		}
		return
	}

	var (
		boundary        bool
		boundaryChecked bool // 保证只检查0次或1次，减少性能开销
	)

	for s := range group.rtspSubSessionSet {
		if !s.ShouldWaitVideoKeyFrame {
			s.WriteRtpPacket(pkt)
			continue
		}

		if !boundaryChecked {
			switch group.sdpCtx.GetVideoPayloadTypeBase() {
			case base.AvPacketPtAvc:
				boundary = rtprtcp.IsAvcBoundary(pkt)
			case base.AvPacketPtHevc:
				boundary = rtprtcp.IsHevcBoundary(pkt)
			default:
				// 注意，不是avc和hevc时，直接发送
				boundary = true
			}
			boundaryChecked = true
		}

		if boundary {
			s.WriteRtpPacket(pkt)
			s.ShouldWaitVideoKeyFrame = false
		}
	}
}

func (group *Group) write2RtmpSubSessions(b []byte) {
	for session := range group.rtmpSubSessionSet {
		if session.IsFresh || session.ShouldWaitVideoKeyFrame {
			continue
		}
		_ = session.Write(b)
	}
}

// ---------------------------------------------------------------------------------------------------------------------

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

	nazalog.Infof("[%s] start relay pull. url=%s", group.UniqueKey, group.getPullUrl())

	go func() {
		pullSession := rtmp.NewPullSession(func(option *rtmp.PullSessionOption) {
			option.PullTimeoutMs = relayPullTimeoutMs
			option.ReadAvTimeoutMs = relayPullReadAvTimeoutMs
		})
		// TODO(chef): 处理数据回调，是否应该等待Add成功之后。避免竞态条件中途加入了其他in session
		err := pullSession.Pull(group.getPullUrl(), group.OnReadRtmpAvMsg)
		if err != nil {
			nazalog.Errorf("[%s] relay pull fail. err=%v", pullSession.UniqueKey(), err)
			group.DelRtmpPullSession(pullSession)
			return
		}
		res := group.AddRtmpPullSession(pullSession)
		if res {
			err = <-pullSession.WaitChan()
			nazalog.Infof("[%s] relay pull done. err=%v", pullSession.UniqueKey(), err)
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
		nazalog.Infof("[%s] stop pull since no sub session.", group.UniqueKey)
		group.pullProxy.pullSession.Dispose()
	}
}
