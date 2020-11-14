// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/q191201771/lal/app/demo/dispatch/datamanager"
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"
)

// 结合lalserver的HTTP Notify事件通知，以及HTTP API接口，
// 简单演示如何试验一个简单的调度服务，
// 使得多个lalserver节点可以组成一个集群，
// 集群内的所有节点功能都是相同的，
// 你可以将流推送至任意一个节点，并从任意一个节点拉流，
// 同一路流，推流和拉流可以在不同的节点。
//
// 本demo的数据存储在内存中，所以存在单点风险，
// 生产环境可以把数据存储在redis、mysql等数据库中，
// 多个调度节点从数据库中读写数据。

type Server struct {
	rtmpAddr string
	apiAddr  string
}

// config
var (
	listenAddr      = ":10101"
	serverID2Server = map[string]Server{
		"1": {
			rtmpAddr: "127.0.0.1:19350",
			apiAddr:  "127.0.0.1:8083",
		},
		"2": {
			rtmpAddr: "127.0.0.1:19550",
			apiAddr:  "127.0.0.1:8283",
		},
	}
	pullSecretParam = "lal_cluster_inner_pull=1"
)

var (
	dataManager datamanager.DataManger
)

func OnPubStartHandler(w http.ResponseWriter, r *http.Request) {
	id := unique.GenUniqueKey("ReqID")

	var info base.PubStartInfo
	if err := nazahttp.UnmarshalRequestJsonBody(r, &info); err != nil {
		nazalog.Error(err)
		return
	}
	nazalog.Infof("[%s] on_pub_start. info=%+v", id, info)

	// 演示如何踢掉session，服务于鉴权失败等场景
	//if info.URLParam == "" {
	if info.SessionID == "RTMPPUBSUB1" {
		reqServer, exist := serverID2Server[info.ServerID]
		if !exist {
			nazalog.Errorf("[%s] req server id invalid.", id)
			return
		}
		url := fmt.Sprintf("http://%s/api/ctrl/kick_out_session", reqServer.apiAddr)
		var b base.APICtrlKickOutSession
		b.StreamName = info.StreamName
		b.SessionID = info.SessionID

		nazalog.Infof("[%s] ctrl kick out session. send to %s with %+v", id, reqServer.apiAddr, b)
		if _, err := nazahttp.PostJson(url, b, nil); err != nil {
			nazalog.Errorf("[%s] post json error. err=%+v", id, err)
		}
	}

	dataManager.AddPub(info.StreamName, info.SessionID)
}

func OnPubStopHandler(w http.ResponseWriter, r *http.Request) {
	id := unique.GenUniqueKey("ReqID")

	var info base.PubStopInfo
	if err := nazahttp.UnmarshalRequestJsonBody(r, &info); err != nil {
		nazalog.Error(err)
		return
	}
	nazalog.Infof("[%s] on_pub_stop. info=%+v", id, info)

	dataManager.DelPub(info.StreamName, info.ServerID)
}

func OnSubStartHandler(w http.ResponseWriter, r *http.Request) {
	id := unique.GenUniqueKey("ReqID")

	var info base.SubStartInfo
	if err := nazahttp.UnmarshalRequestJsonBody(r, &info); err != nil {
		nazalog.Error(err)
		return
	}
	nazalog.Infof("[%s] on_sub_start. info=%+v", id, info)

	// sub拉流时，判断是否需要触发pull级联拉流
	// 1. 是内部级联拉流，不需要触发
	if strings.Contains(info.URLParam, pullSecretParam) {
		nazalog.Infof("[%s] sub is pull by other node, ignore.", id)
		return
	}
	// 2. 已经存在输入流，不需要触发
	if info.HasInSession {
		nazalog.Infof("[%s] in not empty, ignore.", id)
		return
	}

	// 3. 非法节点，本服务没有配置这个节点
	reqServer, exist := serverID2Server[info.ServerID]
	if !exist {
		nazalog.Errorf("[%s] req server id invalid.", id)
		return
	}

	pubServerID, exist := dataManager.QueryPub(info.StreamName)
	// 4. 没有查到流所在节点，不需要触发
	if !exist {
		nazalog.Infof("[%s] pub not exist, ignore.", id)
		return
	}

	// TODO chef: 5. 这里的容错是否会出现？是否可以去掉？
	pubServer, exist := serverID2Server[pubServerID]
	if !exist {
		nazalog.Errorf("[%s] pub server id invalid. serverID=%s", id, pubServerID)
		return
	}

	// 向pub所在节点，发送pull级联拉流的命令
	url := fmt.Sprintf("http://%s/api/ctrl/start_pull", reqServer.apiAddr)
	var b base.APICtrlStartPullReq
	b.Protocol = base.ProtocolRTMP
	b.Addr = pubServer.rtmpAddr
	b.AppName = info.AppName
	b.StreamName = info.StreamName
	b.URLParam = pullSecretParam

	nazalog.Infof("[%s] ctrl pull. send to %s with %+v", id, reqServer.apiAddr, b)
	if _, err := nazahttp.PostJson(url, b, nil); err != nil {
		nazalog.Errorf("[%s] post json error. err=%+v", id, err)
	}
}

func OnSubStopHandler(w http.ResponseWriter, r *http.Request) {
	id := unique.GenUniqueKey("ReqID")

	var info base.SubStopInfo
	if err := nazahttp.UnmarshalRequestJsonBody(r, &info); err != nil {
		nazalog.Error(err)
		return
	}
	nazalog.Infof("[%s] on_sub_stop. info=%+v", id, info)

	// 没什么好做的
	// 目前lalserver在sub为空时，内部会主动关闭pull
}

func OnUpdateHandler(w http.ResponseWriter, r *http.Request) {
	id := unique.GenUniqueKey("ReqID")

	var info base.UpdateInfo
	if err := nazahttp.UnmarshalRequestJsonBody(r, &info); err != nil {
		nazalog.Error(err)
		return
	}
	nazalog.Infof("[%s] on_update. info=%+v", id, info)

	// TODO chef:
	// 1. 更新pubStream2ServerID，去掉过期的，增加不存在的
	// 2. 没有pub但是有sub的，触发ctrl pull
}

func logHandler(w http.ResponseWriter, r *http.Request) {
	b, _ := ioutil.ReadAll(r.Body)
	nazalog.Infof("r=%+v, body=%s", r, b)
}

func main() {
	dataManager = datamanager.NewDataManager(datamanager.DMTMemory)

	l, err := net.Listen("tcp", listenAddr)
	nazalog.Assert(nil, err)

	m := http.NewServeMux()
	m.HandleFunc("/on_pub_start", OnPubStartHandler)
	m.HandleFunc("/on_pub_stop", OnPubStopHandler)
	m.HandleFunc("/on_sub_start", OnSubStartHandler)
	m.HandleFunc("/on_sub_stop", OnSubStopHandler)
	m.HandleFunc("/on_update", OnUpdateHandler)
	m.HandleFunc("/on_rtmp_connect", logHandler)

	srv := http.Server{
		Handler: m,
	}
	err = srv.Serve(l)
	nazalog.Assert(nil, err)
}
