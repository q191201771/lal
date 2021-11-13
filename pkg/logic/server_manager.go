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
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/q191201771/naza/pkg/bininfo"
	"github.com/q191201771/naza/pkg/defertaskthread"

	"github.com/q191201771/lal/pkg/hls"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/httpts"

	"github.com/q191201771/lal/pkg/rtsp"

	_ "net/http/pprof"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
	//"github.com/felixge/fgprof"
)

type ServerManager struct {
	option          Option
	serverStartTime string
	config          *Config

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

func NewServerManager(confFile string, modOption ...ModOption) *ServerManager {
	sm := &ServerManager{
		serverStartTime: time.Now().Format("2006-01-02 15:04:05.999"),
		exitChan:        make(chan struct{}),
	}
	sm.groupManager = NewSimpleGroupManager(sm)

	sm.config = LoadConfAndInitLog(confFile)

	// TODO(chef): refactor 启动信息可以考虑放入package base中，所有的app都打印
	dir, _ := os.Getwd()
	nazalog.Infof("wd: %s", dir)
	nazalog.Infof("args: %s", strings.Join(os.Args, " "))
	nazalog.Infof("bininfo: %s", bininfo.StringifySingleLine())
	nazalog.Infof("version: %s", base.LalFullInfo)
	nazalog.Infof("github: %s", base.LalGithubSite)
	nazalog.Infof("doc: %s", base.LalDocSite)
	nazalog.Infof("serverStartTime: %s", sm.serverStartTime)

	if sm.config.HlsConfig.Enable && sm.config.HlsConfig.UseMemoryAsDiskFlag {
		nazalog.Infof("hls use memory as disk.")
		hls.SetUseMemoryAsDiskFlag(true)
	}

	if sm.config.RecordConfig.EnableFlv {
		if err := os.MkdirAll(sm.config.RecordConfig.FlvOutPath, 0777); err != nil {
			nazalog.Errorf("record flv mkdir error. path=%s, err=%+v", sm.config.RecordConfig.FlvOutPath, err)
		}
		if err := os.MkdirAll(sm.config.RecordConfig.MpegtsOutPath, 0777); err != nil {
			nazalog.Errorf("record mpegts mkdir error. path=%s, err=%+v", sm.config.RecordConfig.MpegtsOutPath, err)
		}
	}

	sm.option = defaultOption
	for _, fn := range modOption {
		fn(&sm.option)
	}
	if sm.option.NotifyHandler == nil {
		sm.option.NotifyHandler = NewHttpNotify(sm.config.HttpNotifyConfig)
	}

	if sm.config.HttpflvConfig.Enable || sm.config.HttpflvConfig.EnableHttps ||
		sm.config.HttptsConfig.Enable || sm.config.HttptsConfig.EnableHttps ||
		sm.config.HlsConfig.Enable || sm.config.HlsConfig.EnableHttps {
		sm.httpServerManager = base.NewHttpServerManager()
		sm.httpServerHandler = NewHttpServerHandler(sm)
		sm.hlsServerHandler = hls.NewServerHandler(sm.config.HlsConfig.OutPath)
	}

	if sm.config.RtmpConfig.Enable {
		// TODO(chef): refactor 参数顺序统一。Observer都放最后好一些。比如rtmp和rtsp的NewServer
		sm.rtmpServer = rtmp.NewServer(sm, sm.config.RtmpConfig.Addr)
	}
	if sm.config.RtspConfig.Enable {
		sm.rtspServer = rtsp.NewServer(sm.config.RtspConfig.Addr, sm)
	}
	if sm.config.HttpApiConfig.Enable {
		sm.httpApiServer = NewHttpApiServer(sm.config.HttpApiConfig.Addr, sm)
	}

	return sm
}

// ----- implement ILalServer interface --------------------------------------------------------------------------------

func (sm *ServerManager) RunLoop() error {
	sm.option.NotifyHandler.OnServerStart(sm.StatLalInfo())

	if sm.config.PprofConfig.Enable {
		go runWebPprof(sm.config.PprofConfig.Addr)
	}

	go base.RunSignalHandler(func() {
		sm.Dispose()
	})

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

	if err := addMux(sm.config.HttpflvConfig.CommonHttpServerConfig, sm.httpServerHandler.ServeSubSession, "httpflv"); err != nil {
		return err
	}
	if err := addMux(sm.config.HttptsConfig.CommonHttpServerConfig, sm.httpServerHandler.ServeSubSession, "httpts"); err != nil {
		return err
	}
	if err := addMux(sm.config.HlsConfig.CommonHttpServerConfig, sm.hlsServerHandler.ServeHTTP, "hls"); err != nil {
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
			if err := sm.httpApiServer.RunLoop(); err != nil {
				nazalog.Error(err)
			}
		}()
	}

	uis := uint32(sm.config.HttpNotifyConfig.UpdateIntervalSec)
	var updateInfo base.UpdateInfo
	updateInfo.ServerId = sm.config.ServerId
	updateInfo.Groups = sm.StatAllGroup()
	sm.option.NotifyHandler.OnUpdate(updateInfo)

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
			sm.groupManager.Iterate(func(group *Group) bool {
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
					sm.groupManager.Iterate(func(group *Group) bool {
						nazalog.Debugf("%s", group.StringifyDebugStats())
						return true
					})
				}
			}

			sm.mutex.Unlock()

			// 定时通过http notify发送group相关的信息
			if uis != 0 && (count%uis) == 0 {
				updateInfo.ServerId = sm.config.ServerId
				updateInfo.Groups = sm.StatAllGroup()
				sm.option.NotifyHandler.OnUpdate(updateInfo)
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
	sm.groupManager.Iterate(func(group *Group) bool {
		group.Dispose()
		return true
	})
	sm.mutex.Unlock()

	sm.exitChan <- struct{}{}
}

func (sm *ServerManager) StatLalInfo() base.LalInfo {
	var lalInfo base.LalInfo
	lalInfo.BinInfo = bininfo.StringifySingleLine()
	lalInfo.LalVersion = base.LalVersion
	lalInfo.ApiVersion = base.HttpApiVersion
	lalInfo.NotifyVersion = base.HttpNotifyVersion
	lalInfo.StartTime = sm.serverStartTime
	lalInfo.ServerId = sm.config.ServerId
	return lalInfo
}

func (sm *ServerManager) StatAllGroup() (sgs []base.StatGroup) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.groupManager.Iterate(func(group *Group) bool {
		sgs = append(sgs, group.GetStat())
		return true
	})
	return
}

func (sm *ServerManager) StatGroup(streamName string) *base.StatGroup {
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
func (sm *ServerManager) CtrlStartPull(info base.ApiCtrlStartPullReq) {
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
func (sm *ServerManager) CtrlKickOutSession(info base.ApiCtrlKickOutSession) base.HttpResponseBasic {
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

// ----- implement rtmp.ServerObserver interface -----------------------------------------------------------------------

func (sm *ServerManager) OnRtmpConnect(session *rtmp.ServerSession, opa rtmp.ObjectPairArray) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	var info base.RtmpConnectInfo
	info.ServerId = sm.config.ServerId
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
	sm.option.NotifyHandler.OnRtmpConnect(info)
}

func (sm *ServerManager) OnNewRtmpPubSession(session *rtmp.ServerSession) bool {
	nazalog.Debugf("CHEFERASEME [%s] OnNewRtmpPubSession. %s, %s, %s, %s", session.UniqueKey(), session.Url(), session.AppName(), session.StreamName(), session.RawQuery())
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	res := group.AddRtmpPubSession(session)

	// TODO chef: res值为false时，可以考虑不回调
	// TODO chef: 每次赋值都逐个拼，代码冗余，考虑直接用ISession抽离一下代码
	var info base.PubStartInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolRtmp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	sm.option.NotifyHandler.OnPubStart(info)
	return res
}

func (sm *ServerManager) OnDelRtmpPubSession(session *rtmp.ServerSession) {
	nazalog.Debugf("CHEFERASEME [%s] OnDelRtmpPubSession. %s, %s, %s, %s", session.UniqueKey(), session.Url(), session.AppName(), session.StreamName(), session.RawQuery())
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName(), session.StreamName())
	if group == nil {
		return
	}

	group.DelRtmpPubSession(session)

	var info base.PubStopInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolRtmp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	sm.option.NotifyHandler.OnPubStop(info)
}

func (sm *ServerManager) OnNewRtmpSubSession(session *rtmp.ServerSession) bool {
	nazalog.Debugf("CHEFERASEME [%s] OnNewRtmpSubSession. %s, %s, %s, %s", session.UniqueKey(), session.Url(), session.AppName(), session.StreamName(), session.RawQuery())
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	group.AddRtmpSubSession(session)

	var info base.SubStartInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolRtmp
	info.Protocol = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	sm.option.NotifyHandler.OnSubStart(info)

	return true
}

func (sm *ServerManager) OnDelRtmpSubSession(session *rtmp.ServerSession) {
	nazalog.Debugf("CHEFERASEME [%s] OnDelRtmpSubSession. %s, %s, %s, %s", session.UniqueKey(), session.Url(), session.AppName(), session.StreamName(), session.RawQuery())
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName(), session.StreamName())
	if group == nil {
		return
	}

	group.DelRtmpSubSession(session)

	var info base.SubStopInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolRtmp
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	sm.option.NotifyHandler.OnSubStop(info)
}

// ----- implement HttpServerHandlerObserver interface -----------------------------------------------------------------

func (sm *ServerManager) OnNewHttpflvSubSession(session *httpflv.SubSession) bool {
	nazalog.Debugf("CHEFERASEME [%s] OnNewHttpflvSubSession. %s, %s, %s, %s", session.UniqueKey(), session.Url(), session.AppName(), session.StreamName(), session.RawQuery())
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	group.AddHttpflvSubSession(session)

	var info base.SubStartInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolHttpflv
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	sm.option.NotifyHandler.OnSubStart(info)
	return true
}

func (sm *ServerManager) OnDelHttpflvSubSession(session *httpflv.SubSession) {
	nazalog.Debugf("CHEFERASEME [%s] OnDelHttpflvSubSession. %s, %s, %s, %s", session.UniqueKey(), session.Url(), session.AppName(), session.StreamName(), session.RawQuery())
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName(), session.StreamName())
	if group == nil {
		return
	}

	group.DelHttpflvSubSession(session)

	var info base.SubStopInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolHttpflv
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	sm.option.NotifyHandler.OnSubStop(info)
}

func (sm *ServerManager) OnNewHttptsSubSession(session *httpts.SubSession) bool {
	nazalog.Debugf("CHEFERASEME [%s] OnNewHttptsSubSession. %s, %s, %s, %s", session.UniqueKey(), session.Url(), session.AppName(), session.StreamName(), session.RawQuery())
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	group.AddHttptsSubSession(session)

	var info base.SubStartInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolHttpts
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	sm.option.NotifyHandler.OnSubStart(info)
	return true
}

func (sm *ServerManager) OnDelHttptsSubSession(session *httpts.SubSession) {
	nazalog.Debugf("CHEFERASEME [%s] OnDelHttptsSubSession. %s, %s, %s, %s", session.UniqueKey(), session.Url(), session.AppName(), session.StreamName(), session.RawQuery())
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName(), session.StreamName())
	if group == nil {
		return
	}

	group.DelHttptsSubSession(session)

	var info base.SubStopInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolHttpts
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	sm.option.NotifyHandler.OnSubStop(info)
}

// ----- implement rtsp.ServerObserver interface -----------------------------------------------------------------------

func (sm *ServerManager) OnNewRtspSessionConnect(session *rtsp.ServerCommandSession) {
	// TODO chef: impl me
}

func (sm *ServerManager) OnDelRtspSession(session *rtsp.ServerCommandSession) {
	// TODO chef: impl me
}

func (sm *ServerManager) OnNewRtspPubSession(session *rtsp.PubSession) bool {
	nazalog.Debugf("CHEFERASEME [%s] OnNewRtspPubSession. %s, %s, %s, %s", session.UniqueKey(), session.Url(), session.AppName(), session.StreamName(), session.RawQuery())
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	res := group.AddRtspPubSession(session)

	var info base.PubStartInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolRtsp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	sm.option.NotifyHandler.OnPubStart(info)

	return res
}

func (sm *ServerManager) OnDelRtspPubSession(session *rtsp.PubSession) {
	// TODO chef: impl me
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName(), session.StreamName())
	if group == nil {
		return
	}

	group.DelRtspPubSession(session)

	var info base.PubStopInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolRtsp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	sm.option.NotifyHandler.OnPubStop(info)
}

func (sm *ServerManager) OnNewRtspSubSessionDescribe(session *rtsp.SubSession) (ok bool, sdp []byte) {
	nazalog.Debugf("CHEFERASEME [%s] OnNewRtspSubSessionDescribe. %s, %s, %s, %s", session.UniqueKey(), session.Url(), session.AppName(), session.StreamName(), session.RawQuery())
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	return group.HandleNewRtspSubSessionDescribe(session)
}

func (sm *ServerManager) OnNewRtspSubSessionPlay(session *rtsp.SubSession) bool {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())

	res := group.HandleNewRtspSubSessionPlay(session)

	var info base.SubStartInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolRtsp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	sm.option.NotifyHandler.OnSubStart(info)

	return res
}

func (sm *ServerManager) OnDelRtspSubSession(session *rtsp.SubSession) {
	// TODO chef: impl me
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup(session.AppName(), session.StreamName())
	if group == nil {
		return
	}

	group.DelRtspSubSession(session)

	var info base.SubStopInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolRtsp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()
	sm.option.NotifyHandler.OnSubStop(info)
}

// ----- implement IGroupCreator interface -----------------------------------------------------------------------------

func (sm *ServerManager) CreateGroup(appName string, streamName string) *Group {
	return NewGroup(appName, streamName, sm.config, sm)
}

// ----- implement GroupObserver interface -----------------------------------------------------------------------------

func (sm *ServerManager) CleanupHlsIfNeeded(appName string, streamName string, path string) {
	if sm.config.HlsConfig.Enable &&
		(sm.config.HlsConfig.CleanupMode == hls.CleanupModeInTheEnd || sm.config.HlsConfig.CleanupMode == hls.CleanupModeAsap) {
		defertaskthread.Go(
			sm.config.HlsConfig.FragmentDurationMs*sm.config.HlsConfig.FragmentNum*2,
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
			appName,
			streamName,
			path,
		)
	}
}

// ---------------------------------------------------------------------------------------------------------------------

func (sm *ServerManager) Config() *Config {
	return sm.config
}

func (sm *ServerManager) GetGroup(appName string, streamName string) *Group {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	return sm.getGroup(appName, streamName)
}

// ----- private method ------------------------------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------------------------------------------------

func runWebPprof(addr string) {
	nazalog.Infof("start web pprof listen. addr=%s", addr)

	//nazalog.Warn("start fgprof.")
	//http.DefaultServeMux.Handle("/debug/fgprof", fgprof.Handler())

	if err := http.ListenAndServe(addr, nil); err != nil {
		nazalog.Error(err)
		return
	}
}
