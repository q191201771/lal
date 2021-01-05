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

type HTTPNotify struct {
	taskQueue chan PostTask
	client    *http.Client
}

var httpNotify *HTTPNotify

func (h *HTTPNotify) OnServerStart() {
	var info base.LALInfo
	info.BinInfo = bininfo.StringifySingleLine()
	info.LalVersion = base.LALVersion
	info.APIVersion = base.HTTPAPIVersion
	info.NotifyVersion = base.HTTPNotifyVersion
	info.StartTime = serverStartTime
	info.ServerID = config.ServerID
	h.asyncPost(config.HTTPNotifyConfig.OnServerStart, info)
}

func (h *HTTPNotify) OnUpdate(info base.UpdateInfo) {
	h.asyncPost(config.HTTPNotifyConfig.OnUpdate, info)
}

func (h *HTTPNotify) OnPubStart(info base.PubStartInfo) {
	h.asyncPost(config.HTTPNotifyConfig.OnPubStart, info)
}

func (h *HTTPNotify) OnPubStop(info base.PubStopInfo) {
	h.asyncPost(config.HTTPNotifyConfig.OnPubStop, info)
}

func (h *HTTPNotify) OnSubStart(info base.SubStartInfo) {
	h.asyncPost(config.HTTPNotifyConfig.OnSubStart, info)
}

func (h *HTTPNotify) OnSubStop(info base.SubStopInfo) {
	h.asyncPost(config.HTTPNotifyConfig.OnSubStop, info)
}

func (h *HTTPNotify) OnRTMPConnect(info base.RTMPConnectInfo) {
	h.asyncPost(config.HTTPNotifyConfig.OnRTMPConnect, info)
}

func (h *HTTPNotify) asyncPost(url string, info interface{}) {
	select {
	case h.taskQueue <- PostTask{url: url, info: info}:
		// noop
	default:
		nazalog.Error("http notify queue full.")
	}
}

func (h *HTTPNotify) post(url string, info interface{}) {
	if !config.HTTPNotifyConfig.Enable || url == "" {
		return
	}

	if _, err := nazahttp.PostJson(url, info, h.client); err != nil {
		nazalog.Errorf("http notify post error. err=%+v", err)
	}
}

func (h *HTTPNotify) RunLoop() {
	for {
		select {
		case t := <-h.taskQueue:
			h.post(t.url, t.info)
		}
	}
}

// TODO chef: dispose

func init() {
	httpNotify = &HTTPNotify{
		taskQueue: make(chan PostTask, maxTaskLen),
		client: &http.Client{
			Timeout: time.Duration(notifyTimeoutSec) * time.Second,
		},
	}
	go httpNotify.RunLoop()
}
