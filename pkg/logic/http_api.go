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
	"github.com/q191201771/naza/pkg/nazajson"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/q191201771/naza/pkg/nazahttp"

	"github.com/q191201771/lal/pkg/base"
)

type HttpApiServer struct {
	addr string
	sm   *ServerManager

	ln net.Listener
}

func NewHttpApiServer(addr string, sm *ServerManager) *HttpApiServer {
	return &HttpApiServer{
		addr: addr,
		sm:   sm,
	}
}

func (h *HttpApiServer) Listen() (err error) {
	if h.ln, err = net.Listen("tcp", h.addr); err != nil {
		return
	}
	Log.Infof("start httpapi server listen. addr=%s", h.addr)
	return
}

func (h *HttpApiServer) RunLoop() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/stat/lal_info", h.statLalInfoHandler)
	mux.HandleFunc("/api/stat/group", h.statGroupHandler)
	mux.HandleFunc("/api/stat/all_group", h.statAllGroupHandler)
	mux.HandleFunc("/api/ctrl/start_relay_pull", h.ctrlStartRelayPullHandler)
	mux.HandleFunc("/api/ctrl/stop_relay_pull", h.ctrlStopRelayPullHandler)
	mux.HandleFunc("/api/ctrl/kick_session", h.ctrlKickSessionHandler)

	var srv http.Server
	srv.Handler = mux
	return srv.Serve(h.ln)
}

// TODO chef: dispose

func (h *HttpApiServer) statLalInfoHandler(w http.ResponseWriter, req *http.Request) {
	var v base.ApiStatLalInfo
	v.ErrorCode = base.ErrorCodeSucc
	v.Desp = base.DespSucc
	v.Data = h.sm.StatLalInfo()
	feedback(v, w)
}

func (h *HttpApiServer) statAllGroupHandler(w http.ResponseWriter, req *http.Request) {
	var v base.ApiStatAllGroup
	v.ErrorCode = base.ErrorCodeSucc
	v.Desp = base.DespSucc
	v.Data.Groups = h.sm.StatAllGroup()
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

	v.Data = h.sm.StatGroup(streamName)
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

func (h *HttpApiServer) ctrlStartRelayPullHandler(w http.ResponseWriter, req *http.Request) {
	var v base.ApiCtrlStartRelayPull
	var info base.ApiCtrlStartRelayPullReq

	// TODO(chef): feat nazahttp.UnmarshalRequestJsonBody 只能做必填项检查，没法做选填项
	fn := func() error {
		var body []byte
		var err error
		body, err = ioutil.ReadAll(req.Body)
		if err = json.Unmarshal(body, &info); err != nil {
			return err
		}
		var j nazajson.Json
		j, err = nazajson.New(body)
		if !j.Exist("url") {
			return nazahttp.ErrParamMissing
		}

		if !j.Exist("pull_timeout_ms") {
			info.PullTimeoutMs = 5000
		}
		if !j.Exist("pull_retry_num") {
			info.PullRetryNum = -1
		}
		if !j.Exist("auto_stop_pull_after_no_out_ms") {
			info.AutoStopPullAfterNoOutMs = -1
		}
		return nil
	}

	err := fn()
	if err != nil {
		Log.Warnf("http api start pull error. err=%+v", err)
		v.ErrorCode = base.ErrorCodeParamMissing
		v.Desp = base.DespParamMissing
		feedback(v, w)
		return
	}
	Log.Infof("http api start pull. req info=%+v", info)

	resp := h.sm.CtrlStartRelayPull(info)
	feedback(resp, w)
	return
}

func (h *HttpApiServer) ctrlStopRelayPullHandler(w http.ResponseWriter, req *http.Request) {
	var v base.ApiCtrlStopRelayPull

	q := req.URL.Query()
	streamName := q.Get("stream_name")
	if streamName == "" {
		v.ErrorCode = base.ErrorCodeParamMissing
		v.Desp = base.DespParamMissing
		feedback(v, w)
		return
	}

	Log.Infof("http api stop pull. stream_name=%s", streamName)

	resp := h.sm.CtrlStopRelayPull(streamName)
	feedback(resp, w)
	return
}

func (h *HttpApiServer) ctrlKickSessionHandler(w http.ResponseWriter, req *http.Request) {
	var v base.HttpResponseBasic
	var info base.ApiCtrlKickSession

	err := nazahttp.UnmarshalRequestJsonBody(req, &info, "stream_name", "session_id")
	if err != nil {
		Log.Warnf("http api kick out session error. err=%+v", err)
		v.ErrorCode = base.ErrorCodeParamMissing
		v.Desp = base.DespParamMissing
		feedback(v, w)
		return
	}
	Log.Infof("http api kick out session. req info=%+v", info)

	resp := h.sm.CtrlKickSession(info)
	feedback(resp, w)
	return
}

// ---------------------------------------------------------------------------------------------------------------------

func feedback(v interface{}, w http.ResponseWriter) {
	resp, _ := json.Marshal(v)
	w.Header().Add("Server", base.LalHttpApiServer)
	_, _ = w.Write(resp)
}
