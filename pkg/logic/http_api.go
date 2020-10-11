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

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/bininfo"
	"github.com/q191201771/naza/pkg/nazalog"
)

const httpAPIVersion = "v0.0.1"

const (
	ErrorCodeSucc          = 0
	DespSucc               = "succ"
	ErrorCodeGroupNotFound = 1001
	DespGroupNotFound      = "group not found"
)

var startTime string

type HTTPAPIServerObserver interface {
	OnStatAllGroup() []base.StatGroup
	OnStatGroup(streamName string) *base.StatGroup
}

type HTTPAPIServer struct {
	addr     string
	observer HTTPAPIServerObserver
	ln       net.Listener
}

type HTTPResponseBasic struct {
	ErrorCode int    `json:"error_code"`
	Desp      string `json:"desp"`
}

type APIStatLALInfo struct {
	HTTPResponseBasic
	Data struct {
		BinInfo    string `json:"bin_info"`
		LalVersion string `json:"lal_version"`
		APIVersion string `json:"api_version"`
		StartTime  string `json:"start_time"`
	} `json:"data"`
}

type APIStatAllGroup struct {
	HTTPResponseBasic
	Data struct {
		Groups []base.StatGroup `json:"groups"`
	} `json:"data"`
}

type APIStatGroup struct {
	HTTPResponseBasic
	Data *base.StatGroup `json:"data"`
}

func NewHTTPAPIServer(addr string, observer HTTPAPIServerObserver) *HTTPAPIServer {
	return &HTTPAPIServer{
		addr:     addr,
		observer: observer,
	}
}

func (h *HTTPAPIServer) Listen() (err error) {
	if h.ln, err = net.Listen("tcp", h.addr); err != nil {
		return
	}
	nazalog.Infof("start httpapi server listen. addr=%s", h.addr)
	return
}

func (h *HTTPAPIServer) Runloop() error {
	mux := http.NewServeMux()

	//mux.HandleFunc("/api/list", h.apiListHandler)
	mux.HandleFunc("/api/stat/lal_info", h.statLALInfoHandler)
	mux.HandleFunc("/api/stat/group", h.statGroupHandler)
	mux.HandleFunc("/api/stat/all_group", h.statAllGroupHandler)

	var srv http.Server
	srv.Handler = mux
	return srv.Serve(h.ln)
}

// TODO chef: dispose

func (h *HTTPAPIServer) apiListHandler(w http.ResponseWriter, req *http.Request) {
	// TODO chef: 写完api list页面
	b := []byte(`
<html>
<head><title>lal http api list</title></head>
<body>
<br>
<br>
<ul>
<li><a href="https://pengrl.com">lal http api接口文档</li>
</ul>
</body>
</html>
`)
	w.Header().Add("Server", base.LALHTTPAPIServer)
	_, _ = w.Write(b)
}

func (h *HTTPAPIServer) statLALInfoHandler(w http.ResponseWriter, req *http.Request) {
	var v APIStatLALInfo
	v.ErrorCode = ErrorCodeSucc
	v.Desp = DespSucc
	v.Data.BinInfo = bininfo.StringifySingleLine()
	v.Data.LalVersion = base.LALVersion
	v.Data.APIVersion = httpAPIVersion
	v.Data.StartTime = startTime
	resp, _ := json.Marshal(v)
	w.Header().Add("Server", base.LALHTTPAPIServer)
	_, _ = w.Write(resp)
}

func (h *HTTPAPIServer) statAllGroupHandler(w http.ResponseWriter, req *http.Request) {
	gs := h.observer.OnStatAllGroup()

	var v APIStatAllGroup
	v.ErrorCode = ErrorCodeSucc
	v.Desp = DespSucc
	v.Data.Groups = gs
	resp, _ := json.Marshal(v)
	w.Header().Add("Server", base.LALHTTPAPIServer)
	_, _ = w.Write(resp)
}

func (h *HTTPAPIServer) statGroupHandler(w http.ResponseWriter, req *http.Request) {
	var v APIStatGroup

	q := req.URL.Query()
	streamName := q.Get("stream_name")
	if streamName == "" {
		v.ErrorCode = ErrorCodeGroupNotFound
		v.Desp = DespGroupNotFound
	} else {
		v.Data = h.observer.OnStatGroup(streamName)
		if v.Data == nil {
			v.ErrorCode = ErrorCodeGroupNotFound
			v.Desp = DespGroupNotFound
		} else {
			v.ErrorCode = ErrorCodeSucc
			v.Desp = DespSucc
		}
	}

	resp, _ := json.Marshal(v)
	w.Header().Add("Server", base.LALHTTPAPIServer)
	_, _ = w.Write(resp)
}

func init() {
	startTime = time.Now().Format("2006-01-02 15:04:05.999")
}
