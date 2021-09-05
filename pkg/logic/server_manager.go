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
	"time"

	"github.com/q191201771/lal/pkg/hls"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/httpts"

	"github.com/q191201771/lal/pkg/rtsp"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
)

type ServerManager struct {
	httpServerManager *base.HttpServerManager
	httpServerHandler *HttpServerHandler
	hlsServerHandler  *hls.ServerHandler

	rtmpServer    *rtmp.Server
	rtspServer    *rtsp.Server
	httpApiServer *HttpApiServer
	exitChan      chan struct{}

	mutex        sync.Mutex
	groupManager IGroupManager
}

func NewServerManager() *ServerManager {
	m := &ServerManager{
		exitChan:     make(chan struct{}),
		groupManager: NewSimpleGroupManager(),
	}

	if config.HttpflvConfig.Enable || config.HttpflvConfig.EnableHttps ||
		config.HttptsConfig.Enable || config.HttptsConfig.EnableHttps ||
		config.HlsConfig.Enable || config.HlsConfig.EnableHttps {
		m.httpServerManager = base.NewHttpServerManager()
		m.httpServerHandler = NewHttpServerHandler(m)
		m.hlsServerHandler = hls.NewServerHandler(config.HlsConfig.OutPath)
	}

	if config.RtmpConfig.Enable {
		m.rtmpServer = rtmp.NewServer(m, config.RtmpConfig.Addr)
	}
	if config.RtspConfig.Enable {
		m.rtspServer = rtsp.NewServer(config.RtspConfig.Addr, m)
	}
	if config.HttpApiConfig.Enable {
		m.httpApiServer = NewHttpApiServer(config.HttpApiConfig.Addr, m)
	}
	return m
}

func (sm *ServerManager) RunLoop() error {
	httpNotify.OnServerStart()

	var addMux = func(config CommonHttpServerConfig, handler base.Handler, name string) error {
		if config.Enable {
			err := sm.httpServerManager.AddListen(
				base.LocalAddrCtx{Addr: config.HttpListenAddr},
				config.UrlPattern,
				handler,
			)
			if err != nil {
				nazalog.Infof("add http listen for %s failed. addr=%s, pattern=%s, err=%+v", name, config.HttpListenAddr, config.UrlPattern, err)
				return err
			}
			nazalog.Infof("add http listen for %s. addr=%s, pattern=%s", name, config.HttpListenAddr, config.UrlPattern)
		}
		if config.EnableHttps {
			err := sm.httpServerManager.AddListen(
				base.LocalAddrCtx{IsHttps: true, Addr: config.HttpsListenAddr, CertFile: config.HttpsCertFile, KeyFile: config.HttpsKeyFile},
				config.UrlPattern,
				handler,
			)
			if err != nil {
				nazalog.Infof("add https listen for %s failed. addr=%s, pattern=%s, err=%+v", name, config.HttpListenAddr, config.UrlPattern, err)
				return err
			}
			nazalog.Infof("add https listen for %s. addr=%s, pattern=%s", name, config.HttpsListenAddr, config.UrlPattern)
		}
		return nil
	}

	if err := addMux(config.HttpflvConfig.CommonHttpServerConfig, sm.httpServerHandler.ServeSubSession, "httpflv"); err != nil {
		return err
	}
	if err := addMux(config.HttptsConfig.CommonHttpServerConfig, sm.httpServerHandler.ServeSubSession, "httpts"); err != nil {
		return err
	}
	if err := addMux(config.HlsConfig.CommonHttpServerConfig, sm.hlsServerHandler.ServeHTTP, "hls"); err != nil {
		return err
	}

	if sm.httpServerManager != nil {
		go func() {
			if err := sm.httpServerManager.RunLoop(); err != nil {
				nazalog.Error(err)
			}
		}()
	}

	if sm.rtmpServer != nil {
		if err := sm.rtmpServer.Listen(); err != nil {
			return err
		}
		go func() {
			if err := sm.rtmpServer.RunLoop(); err != nil {
				nazalog.Error(err)
			}
		}()
	}

	if sm.rtspServer != nil {
		if err := sm.rtspServer.Listen(); err != nil {
			return err
		}
		go func() {
			if err := sm.rtspServer.RunLoop(); err != nil {
				nazalog.Error(err)
			}
		}()
	}

	if sm.httpApiServer != nil {
		if err := sm.httpApiServer.Listen(); err != nil {
			return err
		}
		go func() {
			if err := sm.httpApiServer.Runloop(); err != nil {
				nazalog.Error(err)
			}
		}()
	}

	uis := uint32(config.HttpNotifyConfig.UpdateIntervalSec)
	var updateInfo base.UpdateInfo
	updateInfo.ServerId = config.ServerId
	updateInfo.Groups = sm.statAllGroup()
	httpNotify.OnUpdate(updateInfo)

	t := time.NewTicker(1 * time.Second)
	defer t.Stop()
	var count uint32
	for {
		select {
		case <-sm.exitChan:
			return nil
		case <-t.C:
			count++

			sm.mutex.Lock()

			// 关闭空闲的group
			sm.groupManager.Iterate(func(appName string, streamName string, group *Group) bool {
				if group.IsTotalEmpty() {
					nazalog.Infof("erase empty group. [%s]", group.UniqueKey)
					group.Dispose()
					return false
				}

				group.Tick()
				return true
			})

			// 定时打印一些group相关的日志
			if (count % 30) == 0 {
				groupNum := sm.groupManager.Len()
				nazalog.Debugf("group size=%d", groupNum)
				if groupNum < 10 {
					sm.groupManager.Iterate(func(appName string, streamName string, group *Group) bool {
						nazalog.Debugf("%s", group.StringifyDebugStats())
						return true
					})
				}
			}

			sm.mutex.Unlock()

			// 定时通过http notify发送group相关的信息
			if uis != 0 && (count%uis) == 0 {
				updateInfo.ServerId = config.ServerId
				updateInfo.Groups = sm.statAllGroup()
				httpNotify.OnUpdate(updateInfo)
			}
		}
	}

	// never reach here
}

func (sm *ServerManager) Dispose() {
	nazalog.Debug("dispose server manager.")

	// TODO(chef) add httpServer

	if sm.rtmpServer != nil {
		sm.rtmpServer.Dispose()
	}
	//if sm.hlsServer != nil {
	//	sm.hlsServer.Dispose()
	//}

	sm.mutex.Lock()
	sm.groupManager.Iterate(func(appName string, streamName string, group *Group) bool {
		group.Dispose()
		return true
	})
	sm.mutex.Unlock()

	sm.exitChan <- struct{}{}
}

func (sm *ServerManager) GetGroup(appName string, streamName string) *Group {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	return sm.getGroup(appName, streamName)
}

// ---------------------------------------------------------------------------------------------------------------------
// rtmp.ServerObserver interface
// ---------------------------------------------------------------------------------------------------------------------

func (sm *ServerManager) OnRtmpConnect(session *rtmp.ServerSession, opa rtmp.ObjectPairArray) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	var info base.RtmpConnectInfo
	info.ServerId = config.ServerId
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	if app, err := opa.FindString("app"); err == nil {
		info.App = app
	}
	if flashVer, err := opa.FindString("flashVer"); err == nil {
		info.FlashVer = flashVer
	}
	if tcUrl, err := opa.FindString("tcUrl"); err == nil {
		info.TcUrl = tcUrl
	}
	httpNotify.OnRtmpConnect(info)
}

func (sm *ServerManager) OnNewRtmpPubSession(session *rtmp.ServerSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	res := group.AddRtmpPubSession(session)

	// TODO chef: res值为false时，可以考虑不回调
	// TODO chef: 每次赋值都逐个拼，代码冗余，考虑直接用ISession抽离一下代码
	var info base.PubStartInfo
	info.ServerId = config.ServerId
	info.Protocol = base.ProtocolRtmp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	httpNotify.OnPubStart(info)
	return res
}

func (sm *ServerManager) OnDelRtmpPubSession(session *rtmp.ServerSession) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName(), session.StreamName())
	if group == nil {
		return
	}

	group.DelRtmpPubSession(session)

	var info base.PubStopInfo
	info.ServerId = config.ServerId
	info.Protocol = base.ProtocolRtmp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	httpNotify.OnPubStop(info)
}

func (sm *ServerManager) OnNewRtmpSubSession(session *rtmp.ServerSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	group.AddRtmpSubSession(session)

	var info base.SubStartInfo
	info.ServerId = config.ServerId
	info.Protocol = base.ProtocolRtmp
	info.Protocol = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	httpNotify.OnSubStart(info)

	return true
}

func (sm *ServerManager) OnDelRtmpSubSession(session *rtmp.ServerSession) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName(), session.StreamName())
	if group == nil {
		return
	}

	group.DelRtmpSubSession(session)

	var info base.SubStopInfo
	info.ServerId = config.ServerId
	info.Protocol = base.ProtocolRtmp
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	httpNotify.OnSubStop(info)
}

// ---------------------------------------------------------------------------------------------------------------------
// httpflv.ServerObserver interface
// ---------------------------------------------------------------------------------------------------------------------

func (sm *ServerManager) OnNewHttpflvSubSession(session *httpflv.SubSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	group.AddHttpflvSubSession(session)

	var info base.SubStartInfo
	info.ServerId = config.ServerId
	info.Protocol = base.ProtocolHttpflv
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	httpNotify.OnSubStart(info)
	return true
}

func (sm *ServerManager) OnDelHttpflvSubSession(session *httpflv.SubSession) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName(), session.StreamName())
	if group == nil {
		return
	}

	group.DelHttpflvSubSession(session)

	var info base.SubStopInfo
	info.ServerId = config.ServerId
	info.Protocol = base.ProtocolHttpflv
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	httpNotify.OnSubStop(info)
}

// ---------------------------------------------------------------------------------------------------------------------
// httpts.ServerObserver interface
// ---------------------------------------------------------------------------------------------------------------------

func (sm *ServerManager) OnNewHttptsSubSession(session *httpts.SubSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	group.AddHttptsSubSession(session)

	var info base.SubStartInfo
	info.ServerId = config.ServerId
	info.Protocol = base.ProtocolHttpts
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	httpNotify.OnSubStart(info)
	return true
}

func (sm *ServerManager) OnDelHttptsSubSession(session *httpts.SubSession) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName(), session.StreamName())
	if group == nil {
		return
	}

	group.DelHttptsSubSession(session)

	var info base.SubStopInfo
	info.ServerId = config.ServerId
	info.Protocol = base.ProtocolHttpts
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	httpNotify.OnSubStop(info)
}

// ---------------------------------------------------------------------------------------------------------------------
// rtsp.ServerObserver interface
// ---------------------------------------------------------------------------------------------------------------------

func (sm *ServerManager) OnNewRtspSessionConnect(session *rtsp.ServerCommandSession) {
	// TODO chef: impl me
}

func (sm *ServerManager) OnDelRtspSession(session *rtsp.ServerCommandSession) {
	// TODO chef: impl me
}

func (sm *ServerManager) OnNewRtspPubSession(session *rtsp.PubSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup("", session.StreamName())
	res := group.AddRtspPubSession(session)

	var info base.PubStartInfo
	info.ServerId = config.ServerId
	info.Protocol = base.ProtocolRtsp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	httpNotify.OnPubStart(info)

	return res
}

func (sm *ServerManager) OnDelRtspPubSession(session *rtsp.PubSession) {
	// TODO chef: impl me
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup("", session.StreamName())
	if group == nil {
		return
	}

	group.DelRtspPubSession(session)

	var info base.PubStopInfo
	info.ServerId = config.ServerId
	info.Protocol = base.ProtocolRtsp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	httpNotify.OnPubStop(info)
}

func (sm *ServerManager) OnNewRtspSubSessionDescribe(session *rtsp.SubSession) (ok bool, sdp []byte) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup("", session.StreamName())
	return group.HandleNewRtspSubSessionDescribe(session)
}

func (sm *ServerManager) OnNewRtspSubSessionPlay(session *rtsp.SubSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup("", session.StreamName())

	res := group.HandleNewRtspSubSessionPlay(session)

	var info base.SubStartInfo
	info.ServerId = config.ServerId
	info.Protocol = base.ProtocolRtsp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	httpNotify.OnSubStart(info)

	return res
}

func (sm *ServerManager) OnDelRtspSubSession(session *rtsp.SubSession) {
	// TODO chef: impl me
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup("", session.StreamName())
	if group == nil {
		return
	}

	group.DelRtspSubSession(session)

	var info base.SubStopInfo
	info.ServerId = config.ServerId
	info.Protocol = base.ProtocolRtsp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	httpNotify.OnSubStop(info)
}

// ---------------------------------------------------------------------------------------------------------------------
// HttpApiServerObserver interface
// ---------------------------------------------------------------------------------------------------------------------

func (sm *ServerManager) OnStatAllGroup() (sgs []base.StatGroup) {
	return sm.statAllGroup()
}

func (sm *ServerManager) OnStatGroup(streamName string) *base.StatGroup {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	g := sm.getGroup("", streamName)
	if g == nil {
		return nil
	}
	// copy
	var ret base.StatGroup
	ret = g.GetStat()
	return &ret
}

func (sm *ServerManager) OnCtrlStartPull(info base.ApiCtrlStartPullReq) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	g := sm.getGroup(info.AppName, info.StreamName)
	if g == nil {
		nazalog.Warnf("group not exist, ignore start pull. streamName=%s", info.StreamName)
		return
	}
	var url string
	if info.UrlParam != "" {
		url = fmt.Sprintf("rtmp://%s/%s/%s?%s", info.Addr, info.AppName, info.StreamName, info.UrlParam)
	} else {
		url = fmt.Sprintf("rtmp://%s/%s/%s", info.Addr, info.AppName, info.StreamName)
	}
	g.StartPull(url)
}

func (sm *ServerManager) OnCtrlKickOutSession(info base.ApiCtrlKickOutSession) base.HttpResponseBasic {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	g := sm.getGroup("", info.StreamName)
	if g == nil {
		return base.HttpResponseBasic{
			ErrorCode: base.ErrorCodeGroupNotFound,
			Desp:      base.DespGroupNotFound,
		}
	}
	if !g.KickOutSession(info.SessionId) {
		return base.HttpResponseBasic{
			ErrorCode: base.ErrorCodeSessionNotFound,
			Desp:      base.DespSessionNotFound,
		}
	}
	return base.HttpResponseBasic{
		ErrorCode: base.ErrorCodeSucc,
		Desp:      base.DespSucc,
	}
}

// ---------------------------------------------------------------------------------------------------------------------

func (sm *ServerManager) statAllGroup() (sgs []base.StatGroup) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.groupManager.Iterate(func(appName string, streamName string, group *Group) bool {
		sgs = append(sgs, group.GetStat())
		return true
	})
	return
}

// 注意，函数内部不加锁，由调用方保证加锁进入
func (sm *ServerManager) getOrCreateGroup(appName string, streamName string) *Group {
	g, createFlag := sm.groupManager.GetOrCreateGroup(appName, streamName)
	if createFlag {
		go g.RunLoop()
	}
	return g
}

func (sm *ServerManager) getGroup(appName string, streamName string) *Group {
	return sm.groupManager.GetGroup(appName, streamName)
}
