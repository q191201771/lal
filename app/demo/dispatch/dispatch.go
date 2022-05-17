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

//
// 结合lalserver的HTTP Notify事件通知，以及HTTP API接口，
// 简单演示如何实现一个简单的调度服务，
// 使得多个lalserver节点可以组成一个集群，
// 集群内的所有节点功能都是相同的，
// 你可以将流推送至任意一个节点，并从任意一个节点拉流，
// 同一路流，推流和拉流可以在不同的节点。
//

var config = Config{
	ListenAddr: ":10101",
	ServerId2Server: map[string]Server{
		"1": {
			RtmpAddr: "127.0.0.1:19350",
			ApiAddr:  "127.0.0.1:8083",
		},
		"2": {
			RtmpAddr: "127.0.0.1:19550",
			ApiAddr:  "127.0.0.1:8283",
		},
	},
	PullSecretParam:  "lal_cluster_inner_pull=1",
	ServerTimeoutSec: 30,
}

var dataManager datamanager.DataManger

func OnPubStartHandler(w http.ResponseWriter, r *http.Request) {
	id := unique.GenUniqueKey("ReqID")

	var info base.PubStartInfo
	if err := nazahttp.UnmarshalRequestJsonBody(r, &info); err != nil {
		nazalog.Error(err)
		return
	}
	nazalog.Infof("[%s] on_pub_start. info=%+v", id, info)

	// 演示如何踢掉session，服务于鉴权失败等场景
	//if info.UrlParam == "" {
	//if info.SessionId == "RTMPPUBSUB1" {
	//	reqServer, exist := config.ServerId2Server[info.ServerId]
	//	if !exist {
	//		nazalog.Errorf("[%s] req server id invalid.", id)
	//		return
	//	}
	//	url := fmt.Sprintf("http://%s/api/ctrl/kick_out_session", reqServer.ApiAddr)
	//	var b base.ApiCtrlKickOutSession
	//	b.StreamName = info.StreamName
	//	b.SessionId = info.SessionId
	//
	//	nazalog.Infof("[%s] ctrl kick out session. send to %s with %+v", id, reqServer.ApiAddr, b)
	//	if _, err := nazahttp.PostJson(url, b, nil); err != nil {
	//		nazalog.Errorf("[%s] post json error. err=%+v", id, err)
	//	}
	//	return
	//}

	if _, exist := config.ServerId2Server[info.ServerId]; !exist {
		nazalog.Errorf("server id has not config. serverId=%s", info.ServerId)
		return
	}

	nazalog.Infof("add pub. streamName=%s, serverId=%s", info.StreamName, info.ServerId)
	dataManager.AddPub(info.StreamName, info.ServerId)
}

func OnPubStopHandler(w http.ResponseWriter, r *http.Request) {
	id := unique.GenUniqueKey("ReqID")

	var info base.PubStopInfo
	if err := nazahttp.UnmarshalRequestJsonBody(r, &info); err != nil {
		nazalog.Error(err)
		return
	}
	nazalog.Infof("[%s] on_pub_stop. info=%+v", id, info)

	if _, exist := config.ServerId2Server[info.ServerId]; !exist {
		nazalog.Errorf("server id has not config. serverId=%s", info.ServerId)
		return
	}

	nazalog.Infof("del pub. streamName=%s, serverId=%s", info.StreamName, info.ServerId)
	dataManager.DelPub(info.StreamName, info.ServerId)
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
	if strings.Contains(info.UrlParam, config.PullSecretParam) {
		nazalog.Infof("[%s] sub is pull by other node, ignore.", id)
		return
	}
	// 2. 汇报的节点已经存在输入流，不需要触发
	if info.HasInSession {
		nazalog.Infof("[%s] in not empty, ignore.", id)
		return
	}

	// 3. 非法节点，本服务没有配置汇报的节点
	reqServer, exist := config.ServerId2Server[info.ServerId]
	if !exist {
		nazalog.Errorf("[%s] req server id invalid.", id)
		return
	}

	pubServerId, exist := dataManager.QueryPub(info.StreamName)
	// 4. 没有查到流所在节点，不需要触发
	if !exist {
		nazalog.Infof("[%s] pub not exist, ignore.", id)
		return
	}

	pubServer, exist := config.ServerId2Server[pubServerId]
	nazalog.Assert(true, exist)

	// 向汇报节点，发送pull级联拉流的命令，其中包含pub所在节点信息
	// TODO(chef): 还没有测试新的接口start_relay_pull，只是保证可以编译通过
	url := fmt.Sprintf("http://%s/api/ctrl/start_relay_pull", reqServer.ApiAddr)
	var b base.ApiCtrlStartRelayPullReq
	b.Url = fmt.Sprintf("%s://%s/%s/%s?%s", "rtmp", pubServer.RtmpAddr, info.AppName, info.StreamName, config.PullSecretParam)
	//b.Protocol = base.ProtocolRtmp
	//b.Addr = pubServer.RtmpAddr
	//b.AppName = info.AppName
	//b.StreamName = info.StreamName
	//b.UrlParam = config.PullSecretParam

	nazalog.Infof("[%s] ctrl pull. send to %s with %+v", id, reqServer.ApiAddr, b)
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
}

func OnUpdateHandler(w http.ResponseWriter, r *http.Request) {
	id := unique.GenUniqueKey("ReqID")

	var info base.UpdateInfo
	if err := nazahttp.UnmarshalRequestJsonBody(r, &info); err != nil {
		nazalog.Error(err)
		return
	}
	nazalog.Infof("[%s] on_update. info=%+v", id, info)

	var streamNameList []string
	for _, g := range info.Groups {
		// pub exist
		if g.StatPub.SessionId != "" {
			streamNameList = append(streamNameList, g.StreamName)
		}
	}
	dataManager.UpdatePub(info.ServerId, streamNameList)
}

func logHandler(w http.ResponseWriter, r *http.Request) {
	b, _ := ioutil.ReadAll(r.Body)
	nazalog.Infof("r=%+v, body=%s", r, b)
}

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})
	defer nazalog.Sync()
	base.LogoutStartInfo()

	dataManager = datamanager.NewDataManager(datamanager.DmtMemory, config.ServerTimeoutSec)

	l, err := net.Listen("tcp", config.ListenAddr)
	nazalog.Assert(nil, err)

	m := http.NewServeMux()
	m.HandleFunc("/on_pub_start", OnPubStartHandler)
	m.HandleFunc("/on_pub_stop", OnPubStopHandler)
	m.HandleFunc("/on_sub_start", OnSubStartHandler)
	m.HandleFunc("/on_sub_stop", OnSubStopHandler)
	m.HandleFunc("/on_update", OnUpdateHandler)
	m.HandleFunc("/on_rtmp_connect", logHandler)
	m.HandleFunc("/on_server_start", logHandler)

	srv := http.Server{
		Handler: m,
	}
	err = srv.Serve(l)
	nazalog.Assert(nil, err)
}
