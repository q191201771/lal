// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"github.com/q191201771/lal/pkg/base"
	"path/filepath"
)

// ---------------------------------------------------------------------------------------------------------------------

type ILalServer interface {
	RunLoop() error
	Dispose()

	// AddCustomizePubSession 定制化增强功能。业务方可以将自己的流输入到 ILalServer 中
	//
	// @example 示例见 lal/app/demo/customize_lalserver
	//
	// @doc 文档见 <lalserver二次开发 - pub接入自定义流> https://pengrl.com/lal/#/customize_pub
	//
	AddCustomizePubSession(streamName string) (ICustomizePubSessionContext, error)

	// DelCustomizePubSession 将 ICustomizePubSessionContext 从 ILalServer 中删除
	//
	DelCustomizePubSession(ICustomizePubSessionContext)

	// StatLalInfo StatAllGroup StatGroup CtrlStartPull CtrlKickOutSession
	//
	// 一些获取状态、发送控制命令的API。
	// 目的是方便业务方在不修改logic包内代码的前提下，在外层实现一些特定逻辑的定制化开发。
	//
	StatLalInfo() base.LalInfo
	StatAllGroup() (sgs []base.StatGroup)
	StatGroup(streamName string) *base.StatGroup
	CtrlStartRelayPull(info base.ApiCtrlStartRelayPullReq) base.ApiCtrlStartRelayPull
	CtrlStopRelayPull(streamName string) base.ApiCtrlStopRelayPull
	CtrlKickSession(info base.ApiCtrlKickSession) base.HttpResponseBasic
}

// NewLalServer 创建一个lal server
//
// @param modOption: 定制化配置。可变参数，如果不关心，可以不填，具体字段见 Option
//
func NewLalServer(modOption ...ModOption) ILalServer {
	return NewServerManager(modOption...)
}

// ---------------------------------------------------------------------------------------------------------------------

type ICustomizePubSessionContext interface {
	// IAvPacketStream 传入音视频数据相关的接口。详细说明见 base.IAvPacketStream
	//
	base.IAvPacketStream

	UniqueKey() string
	StreamName() string
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
	OnRelayPullStart(info base.PullStartInfo)
	OnRelayPullStop(info base.PullStopInfo)
	OnRtmpConnect(info base.RtmpConnectInfo)
	OnHlsMakeTs(info base.HlsMakeTsInfo)
}

type Option struct {
	// ConfFilename 配置文件，注意，如果为空，内部会尝试从 DefaultConfFilenameList 读取默认配置文件
	//
	ConfFilename string

	// NotifyHandler
	//
	// 事件监听
	// 业务方可实现 INotifyHandler 接口并传入从而获取到对应的事件通知。
	// 如果不填写保持默认值nil，内部默认走http notify的逻辑（当然，还需要在配置文件中开启http notify功能）。
	// 注意，如果业务方实现了自己的事件监听，则lal server内部不再走http notify的逻辑（也即二选一）。
	//
	NotifyHandler INotifyHandler
}

var defaultOption = Option{
	NotifyHandler: nil, // 注意，为nil时，内部会赋值为 HttpNotify
}

type ModOption func(option *Option)

// DefaultConfFilenameList 没有指定配置文件时，按顺序作为优先级，找到第一个存在的并使用
//
var DefaultConfFilenameList = []string{
	filepath.FromSlash("lalserver.conf.json"),
	filepath.FromSlash("./conf/lalserver.conf.json"),
	filepath.FromSlash("../lalserver.conf.json"),
	filepath.FromSlash("../conf/lalserver.conf.json"),
	filepath.FromSlash("../../lalserver.conf.json"),
	filepath.FromSlash("../../conf/lalserver.conf.json"),
	filepath.FromSlash("../../../lalserver.conf.json"),
	filepath.FromSlash("../../../conf/lalserver.conf.json"),
	filepath.FromSlash("lal/conf/lalserver.conf.json"),
}
