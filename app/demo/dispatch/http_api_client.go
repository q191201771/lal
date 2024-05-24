// Copyright 2024, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"fmt"
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
)

func kickSession(serverId, streamName, sessionId string) {
	reqServer, exist := config.ServerId2Server[serverId]
	if !exist {
		nazalog.Errorf("[%s] req server id invalid.", serverId)
		return
	}

	url := fmt.Sprintf("http://%s/api/ctrl/kick_session", reqServer.ApiAddr)
	var b base.ApiCtrlKickSessionReq
	b.StreamName = streamName
	b.SessionId = sessionId

	nazalog.Infof("[%s] kickSession. send to %s with %+v", serverId, reqServer.ApiAddr, b)
	if _, err := nazahttp.PostJson(url, b, nil); err != nil {
		nazalog.Errorf("[%s] post json error. err=%+v", serverId, err)
	}
	return
}

func addIpBlacklist(serverId, ip string, durationSec int) {
	reqServer, exist := config.ServerId2Server[serverId]
	if !exist {
		nazalog.Errorf("[%s] req server id invalid.", serverId)
		return
	}

	url := fmt.Sprintf("http://%s/api/ctrl/add_ip_blacklist", reqServer.ApiAddr)
	var b base.ApiCtrlAddIpBlacklistReq
	b.Ip = ip
	b.DurationSec = durationSec

	nazalog.Infof("[%s] addIpBlacklist. send to %s with %+v", serverId, reqServer.ApiAddr, b)
	if _, err := nazahttp.PostJson(url, b, nil); err != nil {
		nazalog.Errorf("[%s] post json error. err=%+v", serverId, err)
	}
	return
}
