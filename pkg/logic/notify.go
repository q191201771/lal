// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"github.com/q191201771/lal/pkg/base"
)

type INotifyHandler interface {
	OnServerStart()
	OnUpdate(info base.UpdateInfo)
	OnPubStart(info base.PubStartInfo)
	OnPubStop(info base.PubStopInfo)
	OnSubStart(info base.SubStartInfo)
	OnSubStop(info base.SubStopInfo)
	OnRtmpConnect(info base.RtmpConnectInfo)
}

type DefaultNotifyHandler struct {
	httpNotify *HttpNotify
}

func NewDefaultNotifyHandler() *DefaultNotifyHandler {
	NewHttpNotify()
	return &DefaultNotifyHandler{
		httpNotify: NewHttpNotify(),
	}
}

func (d *DefaultNotifyHandler) OnServerStart() {
	d.httpNotify.OnServerStart()
}

func (d *DefaultNotifyHandler) OnUpdate(info base.UpdateInfo) {
	d.httpNotify.OnUpdate(info)
}

func (d *DefaultNotifyHandler) OnPubStart(info base.PubStartInfo) {
	d.httpNotify.OnPubStart(info)
}

func (d *DefaultNotifyHandler) OnPubStop(info base.PubStopInfo) {
	d.httpNotify.OnPubStop(info)
}

func (d *DefaultNotifyHandler) OnSubStart(info base.SubStartInfo) {
	d.httpNotify.OnSubStart(info)
}

func (d *DefaultNotifyHandler) OnSubStop(info base.SubStopInfo) {
	d.httpNotify.OnSubStop(info)
}

func (d *DefaultNotifyHandler) OnRtmpConnect(info base.RtmpConnectInfo) {
	d.httpNotify.OnRtmpConnect(info)
}
