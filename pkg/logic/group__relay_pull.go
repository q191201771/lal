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

	"github.com/q191201771/lal/pkg/rtmp"
)

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

type pullProxy struct {
	isPulling   bool
	pullSession *rtmp.PullSession
}

func (group *Group) initRelayPull() {
	enable := group.config.RelayPullConfig.Enable
	addr := group.config.RelayPullConfig.Addr
	appName := group.appName
	streamName := group.streamName

	// 根据配置文件中的静态回源配置来初始化回源设置
	var pullUrl string
	if enable {
		pullUrl = fmt.Sprintf("rtmp://%s/%s/%s", addr, appName, streamName)
	}
	group.setPullUrl(enable, pullUrl)
}

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
