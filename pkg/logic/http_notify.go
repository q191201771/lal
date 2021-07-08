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

	"github.com/q191201771/naza/pkg/bininfo"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
)

var (
	maxTaskLen       = 1024
	notifyTimeoutSec = 3
)

type PostTask struct {
	url  string
	info interface{}
}

type HttpNotify struct {
	taskQueue chan PostTask
	client    *http.Client
}

var httpNotify *HttpNotify

// 注意，这里的函数命名以On开头并不是因为是回调函数，而是notify给业务方的接口叫做on_server_start
func (h *HttpNotify) OnServerStart() {
	var info base.LalInfo
	info.BinInfo = bininfo.StringifySingleLine()
	info.LalVersion = base.LalVersion
	info.ApiVersion = base.HttpApiVersion
	info.NotifyVersion = base.HttpNotifyVersion
	info.StartTime = serverStartTime
	info.ServerId = config.ServerId
	h.asyncPost(config.HttpNotifyConfig.OnServerStart, info)
}

func (h *HttpNotify) OnUpdate(info base.UpdateInfo) {
	h.asyncPost(config.HttpNotifyConfig.OnUpdate, info)
}

func (h *HttpNotify) OnPubStart(info base.PubStartInfo) {
	h.asyncPost(config.HttpNotifyConfig.OnPubStart, info)
}

func (h *HttpNotify) OnPubStop(info base.PubStopInfo) {
	h.asyncPost(config.HttpNotifyConfig.OnPubStop, info)
}

func (h *HttpNotify) OnSubStart(info base.SubStartInfo) {
	h.asyncPost(config.HttpNotifyConfig.OnSubStart, info)
}

func (h *HttpNotify) OnSubStop(info base.SubStopInfo) {
	h.asyncPost(config.HttpNotifyConfig.OnSubStop, info)
}

func (h *HttpNotify) OnRtmpConnect(info base.RtmpConnectInfo) {
	h.asyncPost(config.HttpNotifyConfig.OnRtmpConnect, info)
}

func (h *HttpNotify) RunLoop() {
	for {
		select {
		case t := <-h.taskQueue:
			h.post(t.url, t.info)
		}
	}
}

func (h *HttpNotify) asyncPost(url string, info interface{}) {
	if !config.HttpNotifyConfig.Enable || url == "" {
		return
	}

	select {
	case h.taskQueue <- PostTask{url: url, info: info}:
		// noop
	default:
		nazalog.Error("http notify queue full.")
	}
}

func (h *HttpNotify) post(url string, info interface{}) {
	if _, err := nazahttp.PostJson(url, info, h.client); err != nil {
		nazalog.Errorf("http notify post error. err=%+v", err)
	}
}

// TODO chef: dispose

func init() {
	httpNotify = &HttpNotify{
		taskQueue: make(chan PostTask, maxTaskLen),
		client: &http.Client{
			Timeout: time.Duration(notifyTimeoutSec) * time.Second,
		},
	}
	go httpNotify.RunLoop()
}
