// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"fmt"
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtsp"
	"strings"

	"github.com/q191201771/lal/pkg/rtmp"
)

// StartPull 外部命令主动触发pull拉流
//
func (group *Group) StartPull(url string) (string, error) {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	group.setPullUrl(true, url)
	return group.pullIfNeeded()
}

// ---------------------------------------------------------------------------------------------------------------------

type pullProxy struct {
	pullEnable bool // 是否开启pull
	pullUrl    string

	isPulling   bool // 是否正在pull
	rtmpSession *rtmp.PullSession
	rtspSession *rtsp.PullSession
}

// 根据配置文件中的静态回源配置来初始化回源设置
func (group *Group) initRelayPullByConfig() {
	group.pullProxy = &pullProxy{}
	enable := group.config.RelayPullConfig.Enable
	addr := group.config.RelayPullConfig.Addr
	appName := group.appName
	streamName := group.streamName

	var pullUrl string
	if enable {
		pullUrl = fmt.Sprintf("rtmp://%s/%s/%s", addr, appName, streamName)
	}
	group.setPullUrl(enable, pullUrl)
}

func (group *Group) setPullUrl(enable bool, url string) {
	group.pullProxy.pullEnable = enable
	group.pullProxy.pullUrl = url
}

func (group *Group) setPullingFlag(flag bool) {
	group.pullProxy.isPulling = flag
}

func (group *Group) setRtmpPullSession(session *rtmp.PullSession) {
	group.pullProxy.rtmpSession = session
}

func (group *Group) setRtspPullSession(session *rtsp.PullSession) {
	group.pullProxy.rtspSession = session
}

func (group *Group) resetRelayPull() {
	group.pullProxy.isPulling = false
	group.pullProxy.rtmpSession = nil
	group.pullProxy.rtspSession = nil
}

func (group *Group) getStatPull() base.StatPull {
	if group.pullProxy.rtmpSession != nil {
		return base.StatSession2Pull(group.pullProxy.rtmpSession.GetStat())
	}
	if group.pullProxy.rtspSession != nil {
		return base.StatSession2Pull(group.pullProxy.rtspSession.GetStat())
	}
	return base.StatPull{}
}

func (group *Group) disposeInactivePullSession() {
	if group.pullProxy.rtmpSession != nil {
		if readAlive, _ := group.pullProxy.rtmpSession.IsAlive(); !readAlive {
			Log.Warnf("[%s] session timeout. session=%s", group.UniqueKey, group.pullProxy.rtmpSession.UniqueKey())
			group.pullProxy.rtmpSession.Dispose()
		}
	}
	if group.pullProxy.rtspSession != nil {
		if readAlive, _ := group.pullProxy.rtspSession.IsAlive(); !readAlive {
			Log.Warnf("[%s] session timeout. session=%s", group.UniqueKey, group.pullProxy.rtspSession.UniqueKey())
			group.pullProxy.rtspSession.Dispose()
		}
	}
}

func (group *Group) updatePullSessionStat() {
	if group.pullProxy.rtmpSession != nil {
		group.pullProxy.rtmpSession.UpdateStat(calcSessionStatIntervalSec)
	}
	if group.pullProxy.rtspSession != nil {
		group.pullProxy.rtspSession.UpdateStat(calcSessionStatIntervalSec)
	}
}

func (group *Group) hasPullSession() bool {
	return group.pullProxy.rtmpSession != nil || group.pullProxy.rtspSession != nil
}

func (group *Group) pullSessionUniqueKey() string {
	if group.pullProxy.rtmpSession != nil {
		return group.pullProxy.rtmpSession.UniqueKey()
	}
	if group.pullProxy.rtspSession != nil {
		return group.pullProxy.rtspSession.UniqueKey()
	}
	return ""
}

// 判断是否需要pull从远端拉流至本地，如果需要，则触发pull
//
// 当前调用时机：
// 1. 添加新sub session
// 2. 外部命令，比如http api
// 3. 定时器，比如pull的连接断了，通过定时器可以重启触发pull
//
func (group *Group) pullIfNeeded() (string, error) {
	if !group.pullProxy.pullEnable {
		return "", nil
	}

	// 如果没有从本地拉流的，就不需要pull了
	//if !group.hasSubSession() {
	//	return
	//}

	// 如果本地已经有输入型的流，就不需要pull了
	if group.hasInSession() {
		return "", base.ErrDupInStream
	}

	// 已经在pull中，就不需要pull了
	if group.pullProxy.isPulling {
		return "", base.ErrDupInStream
	}
	group.setPullingFlag(true)

	Log.Infof("[%s] start relay pull. url=%s", group.UniqueKey, group.pullProxy.pullUrl)

	isPullByRtmp := strings.HasPrefix(group.pullProxy.pullUrl, "rtmp")

	var rtmpSession *rtmp.PullSession
	var rtspSession *rtsp.PullSession
	var uk string

	if isPullByRtmp {
		rtmpSession = rtmp.NewPullSession(func(option *rtmp.PullSessionOption) {
			option.PullTimeoutMs = relayPullTimeoutMs
		}).WithOnPullSucc(func() {
			err := group.AddRtmpPullSession(rtmpSession)
			if err != nil {
				rtmpSession.Dispose()
				return
			}
		}).WithOnReadRtmpAvMsg(group.OnReadRtmpAvMsg)

		uk = rtmpSession.UniqueKey()
	} else {
		rtspSession = rtsp.NewPullSession(group, func(option *rtsp.PullSessionOption) {
			option.PullTimeoutMs = relayPullTimeoutMs
		}).WithOnDescribeResponse(func() {
			err := group.AddRtspPullSession(rtspSession)
			if err != nil {
				rtspSession.Dispose()
				return
			}
		})

		uk = rtspSession.UniqueKey()
	}

	go func(rtPullUrl string, rtIsPullByRtmp bool, rtRtmpSession *rtmp.PullSession, rtRtspSession *rtsp.PullSession) {
		if rtIsPullByRtmp {
			// TODO(chef): 处理数据回调，是否应该等待Add成功之后。避免竞态条件中途加入了其他in session
			err := rtRtmpSession.Pull(rtPullUrl)
			if err != nil {
				Log.Errorf("[%s] relay pull fail. err=%v", rtRtmpSession.UniqueKey(), err)
				group.DelRtmpPullSession(rtRtmpSession)
				return
			}

			err = <-rtRtmpSession.WaitChan()
			Log.Infof("[%s] relay pull done. err=%v", rtRtmpSession.UniqueKey(), err)
			group.DelRtmpPullSession(rtRtmpSession)
			return
		}

		err := rtRtspSession.Pull(rtPullUrl)
		if err != nil {
			Log.Errorf("[%s] relay pull fail. err=%v", rtRtspSession.UniqueKey(), err)
			group.DelRtspPullSession(rtRtspSession)
			return
		}

		err = <-rtRtspSession.WaitChan()
		Log.Infof("[%s] relay pull done. err=%v", rtRtspSession.UniqueKey(), err)
		group.DelRtspPullSession(rtRtspSession)
		return
	}(group.pullProxy.pullUrl, isPullByRtmp, rtmpSession, rtspSession)

	return uk, nil
}

// stopPullIfNeeded
//
// 判断是否需要停止pull，也即当没有观看者时会停止pull
//
// 当前调用时机是定时器定时检查
//
func (group *Group) stopPullIfNeeded() {
	if !group.hasSubSession() {
		Log.Infof("[%s] stop pull since no sub session.", group.UniqueKey)
		if group.pullProxy.rtmpSession != nil {
			group.pullProxy.rtmpSession.Dispose()
		}
		if group.pullProxy.rtspSession != nil {
			group.pullProxy.rtspSession.Dispose()
		}
	}
}
