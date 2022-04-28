// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"net/http"
	"time"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazahttp"
)

// TODO(chef): refactor 配置参数供外部传入
// TODO(chef): refactor maxTaskLen修改为能表示是阻塞任务的意思
var (
	maxTaskLen       = 1024
	notifyTimeoutSec = 3
)

type PostTask struct {
	url  string
	info interface{}
}

type HttpNotify struct {
	cfg       HttpNotifyConfig
	taskQueue chan PostTask
	client    *http.Client
}

func NewHttpNotify(cfg HttpNotifyConfig) *HttpNotify {
	httpNotify := &HttpNotify{
		cfg:       cfg,
		taskQueue: make(chan PostTask, maxTaskLen),
		client: &http.Client{
			Timeout: time.Duration(notifyTimeoutSec) * time.Second,
		},
	}
	go httpNotify.RunLoop()

	return httpNotify
}

// TODO(chef): Dispose

// ---------------------------------------------------------------------------------------------------------------------

func (h *HttpNotify) NotifyServerStart(info base.LalInfo) {
	h.asyncPost(h.cfg.OnServerStart, info)
}

func (h *HttpNotify) NotifyUpdate(info base.UpdateInfo) {
	h.asyncPost(h.cfg.OnUpdate, info)
}

func (h *HttpNotify) NotifyPubStart(info base.PubStartInfo) {
	h.asyncPost(h.cfg.OnPubStart, info)
}

func (h *HttpNotify) NotifyPubStop(info base.PubStopInfo) {
	h.asyncPost(h.cfg.OnPubStop, info)
}

func (h *HttpNotify) NotifySubStart(info base.SubStartInfo) {
	h.asyncPost(h.cfg.OnSubStart, info)
}

func (h *HttpNotify) NotifySubStop(info base.SubStopInfo) {
	h.asyncPost(h.cfg.OnSubStop, info)
}

func (h *HttpNotify) NotifyRtmpConnect(info base.RtmpConnectInfo) {
	h.asyncPost(h.cfg.OnRtmpConnect, info)
}

// ----- implement INotifyHandler interface ----------------------------------------------------------------------------

func (h *HttpNotify) OnServerStart(info base.LalInfo) {
	h.NotifyServerStart(info)
}

func (h *HttpNotify) OnUpdate(info base.UpdateInfo) {
	h.NotifyUpdate(info)
}

func (h *HttpNotify) OnPubStart(info base.PubStartInfo) {
	h.NotifyPubStart(info)
}

func (h *HttpNotify) OnPubStop(info base.PubStopInfo) {
	h.NotifyPubStop(info)
}

func (h *HttpNotify) OnSubStart(info base.SubStartInfo) {
	h.NotifySubStart(info)
}

func (h *HttpNotify) OnSubStop(info base.SubStopInfo) {
	h.NotifySubStop(info)
}

func (h *HttpNotify) OnRtmpConnect(info base.RtmpConnectInfo) {
	h.NotifyRtmpConnect(info)
}

// ---------------------------------------------------------------------------------------------------------------------

func (h *HttpNotify) RunLoop() {
	for {
		select {
		case t := <-h.taskQueue:
			h.post(t.url, t.info)
		}
	}
}

// ---------------------------------------------------------------------------------------------------------------------

func (h *HttpNotify) asyncPost(url string, info interface{}) {
	if !h.cfg.Enable || url == "" {
		return
	}

	select {
	case h.taskQueue <- PostTask{url: url, info: info}:
		// noop
	default:
		Log.Error("http notify queue full.")
	}
}

func (h *HttpNotify) post(url string, info interface{}) {
	if _, err := nazahttp.PostJson(url, info, h.client); err != nil {
		Log.Errorf("http notify post error. err=%+v", err)
	}
}
