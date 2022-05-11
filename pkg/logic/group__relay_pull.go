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
	"time"

	"github.com/q191201771/lal/pkg/rtmp"
)

// StartPull 外部命令主动触发pull拉流
//
func (group *Group) StartPull(info base.ApiCtrlStartRelayPullReq) (string, error) {
	group.mutex.Lock()
	defer group.mutex.Unlock()

	group.pullProxy.apiEnable = true
	group.pullProxy.pullUrl = info.Url
	group.pullProxy.pullTimeoutMs = info.PullTimeoutMs
	group.pullProxy.pullRetryNum = info.PullRetryNum
	group.pullProxy.autoStopPullAfterNoOutMs = info.AutoStopPullAfterNoOutMs

	return group.pullIfNeeded()
}

// StopPull
//
// @return 如果PullSession存在，返回它的unique key
//
func (group *Group) StopPull() string {
	group.pullProxy.apiEnable = false
	return group.stopPull()
}

// ---------------------------------------------------------------------------------------------------------------------

type pullProxy struct {
	staticRelayPullEnable    bool // 是否开启pull TODO(chef): refactor 这两个bool可以考虑合并成一个
	apiEnable                bool
	pullUrl                  string
	pullTimeoutMs            int
	pullRetryNum             int
	autoStopPullAfterNoOutMs int // 没有观看者时，是否自动停止pull

	startCount   int
	lastHasOutTs int64

	isSessionPulling bool // 是否正在pull，注意，这是一个内部状态，表示的是session的状态，而不是整体任务应该处于的状态
	rtmpSession      *rtmp.PullSession
	rtspSession      *rtsp.PullSession
}

// 根据配置文件中的静态回源配置来初始化回源设置
func (group *Group) initRelayPullByConfig() {
	const (
		staticRelayPullTimeoutMs                = 5000 //
		staticRelayPullRetryNum                 = -1   // -1表示无限重试
		staticRelayPullAutoStopPullAfterNoOutMs = 0    // 0表示没有观众，立即关闭
	)

	enable := group.config.StaticRelayPullConfig.Enable
	addr := group.config.StaticRelayPullConfig.Addr
	appName := group.appName
	streamName := group.streamName

	group.pullProxy = &pullProxy{}

	var pullUrl string
	if enable {
		pullUrl = fmt.Sprintf("rtmp://%s/%s/%s", addr, appName, streamName)
	}

	group.pullProxy.pullUrl = pullUrl
	group.pullProxy.staticRelayPullEnable = enable
	group.pullProxy.pullTimeoutMs = staticRelayPullTimeoutMs
	group.pullProxy.pullRetryNum = staticRelayPullRetryNum
	group.pullProxy.autoStopPullAfterNoOutMs = staticRelayPullAutoStopPullAfterNoOutMs
}

func (group *Group) setRtmpPullSession(session *rtmp.PullSession) {
	group.pullProxy.rtmpSession = session
}

func (group *Group) setRtspPullSession(session *rtsp.PullSession) {
	group.pullProxy.rtspSession = session
}

func (group *Group) resetRelayPullSession() {
	group.pullProxy.isSessionPulling = false
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
	if !group.pullProxy.staticRelayPullEnable && !group.pullProxy.apiEnable {
		return "", nil
	}

	// 如果没有从本地拉流的，就不需要pull了
	if group.ShouldAutoStop() {
		return "", nil
	}

	// 如果本地已经有输入型的流，就不需要pull了
	if group.hasInSession() {
		return "", base.ErrDupInStream
	}

	// 已经在pull中，就不需要pull了
	if group.pullProxy.isSessionPulling {
		return "", base.ErrDupInStream
	}
	group.pullProxy.isSessionPulling = true

	// 检查重试次数
	if group.pullProxy.pullRetryNum >= 0 {
		if group.pullProxy.startCount > group.pullProxy.pullRetryNum {
			return "", nil
		}
	} else {
		// 负数永远都重试
	}
	group.pullProxy.startCount++

	Log.Infof("[%s] start relay pull. url=%s", group.UniqueKey, group.pullProxy.pullUrl)

	isPullByRtmp := strings.HasPrefix(group.pullProxy.pullUrl, "rtmp")

	var rtmpSession *rtmp.PullSession
	var rtspSession *rtsp.PullSession
	var uk string

	if isPullByRtmp {
		rtmpSession = rtmp.NewPullSession(func(option *rtmp.PullSessionOption) {
			option.PullTimeoutMs = group.pullProxy.pullTimeoutMs
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
			option.PullTimeoutMs = group.pullProxy.pullTimeoutMs
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

func (group *Group) stopPull() string {
	Log.Infof("[%s] stop pull since no sub session.", group.UniqueKey)

	// 关闭时，清空用于重试的计数
	group.pullProxy.startCount = 0

	if group.pullProxy.rtmpSession != nil {
		group.pullProxy.rtmpSession.Dispose()
		return group.pullProxy.rtspSession.UniqueKey()
	}
	if group.pullProxy.rtspSession != nil {
		group.pullProxy.rtspSession.Dispose()
		return group.pullProxy.rtmpSession.UniqueKey()
	}
	return ""
}

func (group *Group) tickPullModule() {
	if group.hasSubSession() {
		group.pullProxy.lastHasOutTs = time.Now().Unix()
	}

	if group.ShouldAutoStop() {
		group.stopPull()
	} else {
		group.pullIfNeeded()
	}
}

func (group *Group) ShouldAutoStop() bool {
	if group.pullProxy.autoStopPullAfterNoOutMs < 0 {
		return false
	} else if group.pullProxy.autoStopPullAfterNoOutMs == 0 {
		return !group.hasOutSession()
	} else {
		if group.hasOutSession() {
			return false
		}
		return time.Now().Unix()-group.pullProxy.lastHasOutTs >= int64(group.pullProxy.autoStopPullAfterNoOutMs)
	}
}
