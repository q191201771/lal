// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"flag"
	"fmt"
	"github.com/q191201771/naza/pkg/nazalog"
	"math"
	"net/http"
	"os"
	"path/filepath"
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
	pprofServer   *http.Server
	exitChan      chan struct{}

	mutex        sync.Mutex
	groupManager IGroupManager

	simpleAuthCtx *SimpleAuthCtx
}

func NewServerManager(modOption ...ModOption) *ServerManager {
	sm := &ServerManager{
		serverStartTime: base.ReadableNowTime(),
		exitChan:        make(chan struct{}, 1),
	}
	sm.groupManager = NewSimpleGroupManager(sm)

	sm.option = defaultOption
	for _, fn := range modOption {
		fn(&sm.option)
	}

	confFile := sm.option.ConfFilename
	// 运行参数中没有配置文件，尝试从几个默认位置读取
	if confFile == "" {
		nazalog.Warnf("config file did not specify in the command line, try to load it in the usual path.")
		confFile = firstExistDefaultConfFilename()

		// 所有默认位置都找不到配置文件，退出程序
		if confFile == "" {
			// TODO(chef): refactor ILalserver既然已经作为package提供了，那么内部就不应该包含flag和os exit的操作，应该返回给上层
			// TODO(chef): refactor new中逻辑是否该往后移
			flag.Usage()
			_, _ = fmt.Fprintf(os.Stderr, `
Example:
  %s -c %s

Github: %s
Doc: %s
`, os.Args[0], filepath.FromSlash("./conf/lalserver.conf.json"), base.LalGithubSite, base.LalDocSite)
			base.OsExitAndWaitPressIfWindows(1)
		}
	}
	sm.config = LoadConfAndInitLog(confFile)
	base.LogoutStartInfo()

	if sm.config.HlsConfig.Enable && sm.config.HlsConfig.UseMemoryAsDiskFlag {
		Log.Infof("hls use memory as disk.")
		hls.SetUseMemoryAsDiskFlag(true)
	}

	if sm.config.RecordConfig.EnableFlv {
		if err := os.MkdirAll(sm.config.RecordConfig.FlvOutPath, 0777); err != nil {
			Log.Errorf("record flv mkdir error. path=%s, err=%+v", sm.config.RecordConfig.FlvOutPath, err)
		}
	}

	if sm.config.RecordConfig.EnableMpegts {
		if err := os.MkdirAll(sm.config.RecordConfig.MpegtsOutPath, 0777); err != nil {
			Log.Errorf("record mpegts mkdir error. path=%s, err=%+v", sm.config.RecordConfig.MpegtsOutPath, err)
		}
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
		sm.rtmpServer = rtmp.NewServer(sm.config.RtmpConfig.Addr, sm)
	}
	if sm.config.RtspConfig.Enable {
		sm.rtspServer = rtsp.NewServer(sm.config.RtspConfig.Addr, sm)
	}
	if sm.config.HttpApiConfig.Enable {
		sm.httpApiServer = NewHttpApiServer(sm.config.HttpApiConfig.Addr, sm)
	}

	if sm.config.PprofConfig.Enable {
		sm.pprofServer = &http.Server{Addr: sm.config.PprofConfig.Addr, Handler: nil}
	}

	sm.simpleAuthCtx = NewSimpleAuthCtx(sm.config.SimpleAuthConfig)

	return sm
}

// ----- implement ILalServer interface --------------------------------------------------------------------------------

func (sm *ServerManager) RunLoop() error {
	// TODO(chef): 作为阻塞函数，外部只能获取失败或结束的信息，没法获取到启动成功的信息

	sm.option.NotifyHandler.OnServerStart(sm.StatLalInfo())

	if sm.pprofServer != nil {
		go func() {
			//Log.Warn("start fgprof.")
			//http.DefaultServeMux.Handle("/debug/fgprof", fgprof.Handler())
			Log.Infof("start web pprof listen. addr=%s", sm.config.PprofConfig.Addr)
			if err := sm.pprofServer.ListenAndServe(); err != nil {
				Log.Error(err)
			}
		}()
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
				Log.Errorf("add http listen for %s failed. addr=%s, pattern=%s, err=%+v", name, config.HttpListenAddr, config.UrlPattern, err)
				return err
			}
			Log.Infof("add http listen for %s. addr=%s, pattern=%s", name, config.HttpListenAddr, config.UrlPattern)
		}
		if config.EnableHttps {
			err := sm.httpServerManager.AddListen(
				base.LocalAddrCtx{IsHttps: true, Addr: config.HttpsListenAddr, CertFile: config.HttpsCertFile, KeyFile: config.HttpsKeyFile},
				config.UrlPattern,
				handler,
			)
			if err != nil {
				Log.Errorf("add https listen for %s failed. addr=%s, pattern=%s, err=%+v", name, config.HttpsListenAddr, config.UrlPattern, err)
			} else {
				Log.Infof("add https listen for %s. addr=%s, pattern=%s", name, config.HttpsListenAddr, config.UrlPattern)
			}
		}
		return nil
	}

	if err := addMux(sm.config.HttpflvConfig.CommonHttpServerConfig, sm.httpServerHandler.ServeSubSession, "httpflv"); err != nil {
		return err
	}
	if err := addMux(sm.config.HttptsConfig.CommonHttpServerConfig, sm.httpServerHandler.ServeSubSession, "httpts"); err != nil {
		return err
	}
	if err := addMux(sm.config.HlsConfig.CommonHttpServerConfig, sm.serveHls, "hls"); err != nil {
		return err
	}

	if sm.httpServerManager != nil {
		go func() {
			if err := sm.httpServerManager.RunLoop(); err != nil {
				Log.Error(err)
			}
		}()
	}

	if sm.rtmpServer != nil {
		if err := sm.rtmpServer.Listen(); err != nil {
			return err
		}
		go func() {
			if err := sm.rtmpServer.RunLoop(); err != nil {
				Log.Error(err)
			}
		}()
	}

	if sm.rtspServer != nil {
		if err := sm.rtspServer.Listen(); err != nil {
			return err
		}
		go func() {
			if err := sm.rtspServer.RunLoop(); err != nil {
				Log.Error(err)
			}
		}()
	}

	if sm.httpApiServer != nil {
		if err := sm.httpApiServer.Listen(); err != nil {
			return err
		}
		go func() {
			if err := sm.httpApiServer.RunLoop(); err != nil {
				Log.Error(err)
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
	var tickCount uint32
	for {
		select {
		case <-sm.exitChan:
			return nil
		case <-t.C:
			tickCount++

			sm.mutex.Lock()

			// 关闭空闲的group
			sm.groupManager.Iterate(func(group *Group) bool {
				if group.IsTotalEmpty() {
					Log.Infof("erase empty group. [%s]", group.UniqueKey)
					group.Dispose()
					return false
				}

				group.Tick(tickCount)
				return true
			})

			// 定时打印一些group相关的debug日志
			if sm.config.DebugConfig.LogGroupIntervalSec > 0 &&
				tickCount%uint32(sm.config.DebugConfig.LogGroupIntervalSec) == 0 {
				groupNum := sm.groupManager.Len()
				Log.Debugf("DEBUG_GROUP_LOG: group size=%d", groupNum)
				if sm.config.DebugConfig.LogGroupMaxGroupNum > 0 {
					var loggedGroupCount int
					sm.groupManager.Iterate(func(group *Group) bool {
						loggedGroupCount++
						if loggedGroupCount <= sm.config.DebugConfig.LogGroupMaxGroupNum {
							Log.Debugf("DEBUG_GROUP_LOG: %d %s", loggedGroupCount, group.StringifyDebugStats(sm.config.DebugConfig.LogGroupMaxSubNumPerGroup))
						}
						return true
					})
				}
			}

			sm.mutex.Unlock()

			// 定时通过http notify发送group相关的信息
			if uis != 0 && (tickCount%uis) == 0 {
				updateInfo.ServerId = sm.config.ServerId
				updateInfo.Groups = sm.StatAllGroup()
				sm.option.NotifyHandler.OnUpdate(updateInfo)
			}
		}
	}

	// never reach here
}

func (sm *ServerManager) Dispose() {
	Log.Debug("dispose server manager.")

	if sm.rtmpServer != nil {
		sm.rtmpServer.Dispose()
	}

	if sm.httpServerManager != nil {
		sm.httpServerManager.Dispose()
	}

	if sm.pprofServer != nil {
		sm.pprofServer.Close()
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
		sgs = append(sgs, group.GetStat(math.MaxInt32))
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
	ret = g.GetStat(math.MaxInt32)
	return &ret
}

func (sm *ServerManager) CtrlStartRelayPull(info base.ApiCtrlStartRelayPullReq) (string, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	streamName := info.StreamName
	if streamName == "" {
		ctx, err := base.ParseUrl(info.Url, -1)
		if err != nil {
			return "", err
		}
		streamName = ctx.LastItemOfPath
	}

	// 注意，如果group不存在，我们依然relay pull
	g := sm.getOrCreateGroup("", streamName)

	return g.StartPull(info.Url)
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

func (sm *ServerManager) AddCustomizePubSession(streamName string) (ICustomizePubSessionContext, error) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup("", streamName)
	return group.AddCustomizePubSession(streamName)
}

func (sm *ServerManager) DelCustomizePubSession(sessionCtx ICustomizePubSessionContext) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getGroup("", sessionCtx.StreamName())
	if group == nil {
		return
	}
	group.DelCustomizePubSession(sessionCtx)
}

// ----- implement rtmp.IServerObserver interface -----------------------------------------------------------------------

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

func (sm *ServerManager) OnNewRtmpPubSession(session *rtmp.ServerSession) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

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

	// 先做simple auth鉴权
	if err := sm.simpleAuthCtx.OnPubStart(info); err != nil {
		return err
	}

	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	if err := group.AddRtmpPubSession(session); err != nil {
		return err
	}

	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()

	sm.option.NotifyHandler.OnPubStart(info)
	return nil
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

func (sm *ServerManager) OnNewRtmpSubSession(session *rtmp.ServerSession) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	var info base.SubStartInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolRtmp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr

	if err := sm.simpleAuthCtx.OnSubStart(info); err != nil {
		return err
	}

	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	group.AddRtmpSubSession(session)

	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()

	sm.option.NotifyHandler.OnSubStart(info)
	return nil
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

// ----- implement IHttpServerHandlerObserver interface -----------------------------------------------------------------

func (sm *ServerManager) OnNewHttpflvSubSession(session *httpflv.SubSession) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	var info base.SubStartInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolHttpflv
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr

	if err := sm.simpleAuthCtx.OnSubStart(info); err != nil {
		return err
	}

	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	group.AddHttpflvSubSession(session)

	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()

	sm.option.NotifyHandler.OnSubStart(info)
	return nil
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

func (sm *ServerManager) OnNewHttptsSubSession(session *httpts.SubSession) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	var info base.SubStartInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolHttpts
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr
	sm.option.NotifyHandler.OnSubStart(info)

	if err := sm.simpleAuthCtx.OnSubStart(info); err != nil {
		return err
	}

	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	group.AddHttptsSubSession(session)

	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()

	sm.option.NotifyHandler.OnSubStart(info)

	return nil
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

// ----- implement rtsp.IServerObserver interface -----------------------------------------------------------------------

func (sm *ServerManager) OnNewRtspSessionConnect(session *rtsp.ServerCommandSession) {
	// TODO chef: impl me
}

func (sm *ServerManager) OnDelRtspSession(session *rtsp.ServerCommandSession) {
	// TODO chef: impl me
}

func (sm *ServerManager) OnNewRtspPubSession(session *rtsp.PubSession) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	var info base.PubStartInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolRtsp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr

	if err := sm.simpleAuthCtx.OnPubStart(info); err != nil {
		return err
	}

	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	if err := group.AddRtspPubSession(session); err != nil {
		return err
	}

	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()

	sm.option.NotifyHandler.OnPubStart(info)
	return nil
}

func (sm *ServerManager) OnDelRtspPubSession(session *rtsp.PubSession) {
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
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	return group.HandleNewRtspSubSessionDescribe(session)
}

func (sm *ServerManager) OnNewRtspSubSessionPlay(session *rtsp.SubSession) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	var info base.SubStartInfo
	info.ServerId = sm.config.ServerId
	info.Protocol = base.ProtocolRtsp
	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.UrlParam = session.RawQuery()
	info.SessionId = session.UniqueKey()
	info.RemoteAddr = session.GetStat().RemoteAddr

	if err := sm.simpleAuthCtx.OnSubStart(info); err != nil {
		return err
	}

	group := sm.getOrCreateGroup(session.AppName(), session.StreamName())
	group.HandleNewRtspSubSessionPlay(session)

	info.HasInSession = group.HasInSession()
	info.HasOutSession = group.HasOutSession()

	sm.option.NotifyHandler.OnSubStart(info)
	return nil
}

func (sm *ServerManager) OnDelRtspSubSession(session *rtsp.SubSession) {
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

// ----- implement IGroupObserver interface -----------------------------------------------------------------------------

func (sm *ServerManager) CleanupHlsIfNeeded(appName string, streamName string, path string) {
	if sm.config.HlsConfig.Enable &&
		(sm.config.HlsConfig.CleanupMode == hls.CleanupModeInTheEnd || sm.config.HlsConfig.CleanupMode == hls.CleanupModeAsap) {
		defertaskthread.Go(
			sm.config.HlsConfig.FragmentDurationMs*(sm.config.HlsConfig.FragmentNum+sm.config.HlsConfig.DeleteThreshold),
			func(param ...interface{}) {
				appName := param[0].(string)
				streamName := param[1].(string)
				outPath := param[2].(string)

				if g := sm.GetGroup(appName, streamName); g != nil {
					if g.IsHlsMuxerAlive() {
						Log.Warnf("cancel cleanup hls file path since hls muxer still alive. streamName=%s", streamName)
						return
					}
				}

				Log.Infof("cleanup hls file path. streamName=%s, path=%s", streamName, outPath)
				if err := hls.RemoveAll(outPath); err != nil {
					Log.Warnf("cleanup hls file path error. path=%s, err=%+v", outPath, err)
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

func (sm *ServerManager) serveHls(writer http.ResponseWriter, req *http.Request) {
	urlCtx, err := base.ParseUrl(base.ParseHttpRequest(req), 80)
	if err != nil {
		Log.Errorf("parse url. err=%+v", err)
		return
	}
	if urlCtx.GetFileType() == "m3u8" {
		if err = sm.simpleAuthCtx.OnHls(urlCtx.GetFilenameWithoutType(), urlCtx.RawQuery); err != nil {
			Log.Errorf("simple auth failed. err=%+v", err)
			return
		}
	}

	sm.hlsServerHandler.ServeHTTP(writer, req)
}

// ---------------------------------------------------------------------------------------------------------------------

func firstExistDefaultConfFilename() string {
	for _, dcf := range DefaultConfFilenameList {
		fi, err := os.Stat(dcf)
		if err == nil && fi.Size() > 0 && !fi.IsDir() {
			nazalog.Warnf("%s exist. using it as config file.", dcf)
			return dcf
		} else {
			nazalog.Warnf("%s not exist.", dcf)
		}
	}
	return ""
}
