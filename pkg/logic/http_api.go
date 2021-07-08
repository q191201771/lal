// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/q191201771/naza/pkg/nazahttp"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/bininfo"
	"github.com/q191201771/naza/pkg/nazalog"
)

var serverStartTime string

type HttpApiServerObserver interface {
	OnStatAllGroup() []base.StatGroup
	OnStatGroup(streamName string) *base.StatGroup
	OnCtrlStartPull(info base.ApiCtrlStartPullReq)
	OnCtrlKickOutSession(info base.ApiCtrlKickOutSession) base.HttpResponseBasic
}

type HttpApiServer struct {
	addr     string
	observer HttpApiServerObserver
	ln       net.Listener
}

func NewHttpApiServer(addr string, observer HttpApiServerObserver) *HttpApiServer {
	return &HttpApiServer{
		addr:     addr,
		observer: observer,
	}
}

func (h *HttpApiServer) Listen() (err error) {
	if h.ln, err = net.Listen("tcp", h.addr); err != nil {
		return
	}
	nazalog.Infof("start httpapi server listen. addr=%s", h.addr)
	return
}

func (h *HttpApiServer) Runloop() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/list", h.apiListHandler)
	mux.HandleFunc("/api/stat/lal_info", h.statLalInfoHandler)
	mux.HandleFunc("/api/stat/group", h.statGroupHandler)
	mux.HandleFunc("/api/stat/all_group", h.statAllGroupHandler)
	mux.HandleFunc("/api/ctrl/start_pull", h.ctrlStartPullHandler)
	mux.HandleFunc("/api/ctrl/kick_out_session", h.ctrlKickOutSessionHandler)

	var srv http.Server
	srv.Handler = mux
	return srv.Serve(h.ln)
}

// TODO chef: dispose

func (h *HttpApiServer) apiListHandler(w http.ResponseWriter, req *http.Request) {
	// TODO chef: 写完api list页面
	b := []byte(`
<html>
<head><title>lal http api list</title></head>
<body>
<br>
<br>
<p>api接口列表：</p>
<ul>
	<li><a href="/api/list">/api/list</a></li>
	<li><a href="/api/stat/group?stream_name=test110">/api/stat/group?stream_name=test110</a></li>
	<li><a href="/api/stat/all_group">/api/stat/all_group</a></li>
	<li><a href="/api/stat/lal_info">/api/stat/lal_info</a></li>
	<li><a href="/api/ctrl/start_pull?protocol=rtmp&addr=127.0.0.1:1935&app_name=live&stream_name=test110&url_param=token=aaa">/api/ctrl/start_pull?protocol=rtmp&addr=127.0.0.1:1935&app_name=live&stream_name=test110&url_param=token=aaa</a></li>
</ul>
<br>
<p>其他链接：</p>
<ul>
	<li><a href="https://pengrl.com/p/20100/">lal http api接口说明文档</a></li>
	<li><a href="https://github.com/q191201771/lal">lal github地址</a></li>
</ul>
</body>
</html>
`)

	w.Header().Add("Server", base.LalHttpApiServer)
	_, _ = w.Write(b)
}

func (h *HttpApiServer) statLalInfoHandler(w http.ResponseWriter, req *http.Request) {
	var v base.ApiStatLalInfo
	v.ErrorCode = base.ErrorCodeSucc
	v.Desp = base.DespSucc
	v.Data.BinInfo = bininfo.StringifySingleLine()
	v.Data.LalVersion = base.LalVersion
	v.Data.ApiVersion = base.HttpApiVersion
	v.Data.NotifyVersion = base.HttpNotifyVersion
	v.Data.StartTime = serverStartTime
	v.Data.ServerId = config.ServerId
	feedback(v, w)
}

func (h *HttpApiServer) statAllGroupHandler(w http.ResponseWriter, req *http.Request) {
	gs := h.observer.OnStatAllGroup()
	var v base.ApiStatAllGroup
	v.ErrorCode = base.ErrorCodeSucc
	v.Desp = base.DespSucc
	v.Data.Groups = gs
	feedback(v, w)
}

func (h *HttpApiServer) statGroupHandler(w http.ResponseWriter, req *http.Request) {
	var v base.ApiStatGroup

	q := req.URL.Query()
	streamName := q.Get("stream_name")
	if streamName == "" {
		v.ErrorCode = base.ErrorCodeParamMissing
		v.Desp = base.DespParamMissing
		feedback(v, w)
		return
	}

	v.Data = h.observer.OnStatGroup(streamName)
	if v.Data == nil {
		v.ErrorCode = base.ErrorCodeGroupNotFound
		v.Desp = base.DespGroupNotFound
		feedback(v, w)
		return
	}

	v.ErrorCode = base.ErrorCodeSucc
	v.Desp = base.DespSucc
	feedback(v, w)
	return
}

func (h *HttpApiServer) ctrlStartPullHandler(w http.ResponseWriter, req *http.Request) {
	var v base.HttpResponseBasic
	var info base.ApiCtrlStartPullReq

	err := nazahttp.UnmarshalRequestJsonBody(req, &info, "protocol", "addr", "app_name", "stream_name")
	if err != nil {
		nazalog.Warnf("http api start pull error. err=%+v", err)
		v.ErrorCode = base.ErrorCodeParamMissing
		v.Desp = base.DespParamMissing
		feedback(v, w)
		return
	}
	nazalog.Infof("http api start pull. req info=%+v", info)

	h.observer.OnCtrlStartPull(info)
	v.ErrorCode = base.ErrorCodeSucc
	v.Desp = base.DespSucc
	feedback(v, w)
	return
}

func (h *HttpApiServer) ctrlKickOutSessionHandler(w http.ResponseWriter, req *http.Request) {
	var v base.HttpResponseBasic
	var info base.ApiCtrlKickOutSession

	err := nazahttp.UnmarshalRequestJsonBody(req, &info, "stream_name", "session_id")
	if err != nil {
		nazalog.Warnf("http api kick out session error. err=%+v", err)
		v.ErrorCode = base.ErrorCodeParamMissing
		v.Desp = base.DespParamMissing
		feedback(v, w)
		return
	}
	nazalog.Infof("http api kick out session. req info=%+v", info)

	resp := h.observer.OnCtrlKickOutSession(info)
	feedback(resp, w)
	return
}

func feedback(v interface{}, w http.ResponseWriter) {
	resp, _ := json.Marshal(v)
	w.Header().Add("Server", base.LalHttpApiServer)
	_, _ = w.Write(resp)
}

func init() {
	serverStartTime = time.Now().Format("2006-01-02 15:04:05.999")
}
