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
const CodeSucc = 0
const DespSucc = "succ"

var startTime string

type HTTPAPIServer struct {
	addr string
	ln   net.Listener
}

type HTTPResponseBasic struct {
	Code int    `json:"code"`
	Desp string `json:"desp"`
}

type APILalInfo struct {
	HTTPResponseBasic
	BinInfo    string `json:"bin_info"`
	LalVersion string `json:"lal_version"`
	APIVersion string `json:"api_version"`
	StartTime  string `json:"start_time"`
}

type APIStatAllGroup struct {
	HTTPResponseBasic
	Groups []StatGroupItem `json:"groups"`
}

type StatGroupItem struct {
	StreamName  string `json:"stream_name"`
	AudioCodec  string `json:"audio_codec"`
	VideoCodec  string `json:"video_codec"`
	VideoWidth  string `json:"video_width"`
	VideoHeight string `json:"video_height"`
}

type StatPub struct {
	StatSession
}

type StatSub struct {
	StatSession
}

type StatSession struct {
	Protocol   string `json:"protocol"`
	StartTime  string `json:"start_time"`
	RemoteAddr string `json:"remote_addr"`
	Bitrate    string `json:"bitrate"`
}

func NewHTTPAPIServer(addr string) *HTTPAPIServer {
	return &HTTPAPIServer{
		addr: addr,
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

	mux.HandleFunc("/api/lal_info", h.lalInfo)
	mux.HandleFunc("/api/stat/group", h.statGroup)
	mux.HandleFunc("/api/stat/all_group", h.statAllGroup)

	var srv http.Server
	srv.Handler = mux
	return srv.Serve(h.ln)
}

func (h *HTTPAPIServer) lalInfo(w http.ResponseWriter, req *http.Request) {
	var v APILalInfo
	v.Code = CodeSucc
	v.Desp = DespSucc
	v.BinInfo = bininfo.StringifySingleLine()
	v.LalVersion = base.LALVersion
	v.APIVersion = httpAPIVersion
	v.StartTime = startTime
	resp, _ := json.Marshal(v)
	w.Header().Add("Server", base.LALHTTPAPIServer)
	_, _ = w.Write(resp)
}

func (h *HTTPAPIServer) statGroup(w http.ResponseWriter, req *http.Request) {

}

func (h *HTTPAPIServer) statAllGroup(w http.ResponseWriter, req *http.Request) {

}

func init() {
	startTime = time.Now().String()
}
