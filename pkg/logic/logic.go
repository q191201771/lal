// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import "github.com/q191201771/lal/pkg/base"

type ILalServer interface {
	RunLoop() error
	Dispose()

	// StatLalInfo StatXxx... CtrlXxx...
	// 一些获取状态、发送控制命令的API
	// 目的是方便业务方在不修改logic包内代码的前提下，在外层实现一些特定逻辑的定制化开发
	//
	StatLalInfo() base.LalInfo
	StatAllGroup() (sgs []base.StatGroup)
	StatGroup(streamName string) *base.StatGroup
	CtrlStartPull(info base.ApiCtrlStartPullReq)
	CtrlKickOutSession(info base.ApiCtrlKickOutSession) base.HttpResponseBasic
}

// NewLalServer 创建一个lal server
//
// @param confFile  配置文件地址
//
// @param modOption
//   可变参数，如果不关心，可以不填
//   目的是方便业务方在不修改logic包内代码的前提下，在外层实现一些特定逻辑的定制化开发
//   Option struct中可修改的参数说明：
//     - notifyHandler 事件监听
//                     业务方可实现 INotifyHandler 接口并传入从而获取到对应的事件通知
//                     如果不填写保持默认值nil，内部默认走http notify的逻辑（当然，还需要在配置文件中开启http notify功能）
//                     注意，如果业务方实现了自己的事件监听，则lal server内部不再走http notify的逻辑（也即二选一）
//
func NewLalServer(confFile string, modOption ...ModOption) ILalServer {
	return NewServerManager(confFile, modOption...)
}

// ---------------------------------------------------------------------------------------------------------------------

// INotifyHandler 事件通知接口
//
type INotifyHandler interface {
	OnServerStart(info base.LalInfo)
	OnUpdate(info base.UpdateInfo)
	OnPubStart(info base.PubStartInfo)
	OnPubStop(info base.PubStopInfo)
	OnSubStart(info base.SubStartInfo)
	OnSubStop(info base.SubStopInfo)
	OnRtmpConnect(info base.RtmpConnectInfo)
}

type Option struct {
	NotifyHandler INotifyHandler
}

var defaultOption = Option{
	NotifyHandler: nil, // 注意，为nil时，内部会赋值为 HttpNotify
}

type ModOption func(option *Option)

// ---------------------------------------------------------------------------------------------------------------------

// 一些没有放入配置文件中，包级别的配置，暂时没有对外暴露
//
var (
	relayPushTimeoutMs                = 5000
	relayPushWriteAvTimeoutMs         = 5000
	relayPullTimeoutMs                = 5000
	relayPullReadAvTimeoutMs          = 5000
	calcSessionStatIntervalSec uint32 = 5

	// checkSessionAliveIntervalSec
	//
	// - 对于输入型session，检查一定时间内，是否没有收到数据
	// - 对于输出型session，检查一定时间内，是否没有发送数据
	//   注意，这里既检查socket发送阻塞，又检查上层没有给session喂数据
	//
	checkSessionAliveIntervalSec uint32 = 10
)
