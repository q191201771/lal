// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import (
	"net/http"

	"github.com/q191201771/naza/pkg/nazalog"
)

type Server struct {
	addr    string
	outPath string
}

func NewServer(addr string, outPath string) *Server {
	return &Server{
		addr:    addr,
		outPath: outPath,
	}
}

func (s *Server) RunLoop() error {
	nazalog.Infof("start hls listen. addr=%s", s.addr)
	return http.ListenAndServe(s.addr, s)
}

func (s *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	//nazalog.Debugf("%+v", req)

	// TODO chef:
	// - check appname in URI path
	// - DIY 404 response body

	ri := parseRequestInfo(req.RequestURI)
	//nazalog.Debugf("%+v", ri)

	if ri.fileName == "" || ri.streamName == "" || (ri.fileType != "m3u8" && ri.fileType != "ts") {
		nazalog.Warnf("%+v", ri)
		resp.WriteHeader(404)
		return
	}

	content, err := readFileContent(s.outPath, ri)
	if err != nil {
		nazalog.Warnf("%+v", err)
		resp.WriteHeader(404)
		return
	}

	switch ri.fileType {
	case "m3u8":
		resp.Header().Add("Content-Type", "application/x-mpegurl")
		//resp.Header().Add("Content-Type", "application/vnd.apple.mpegurl")
	case "ts":
		resp.Header().Add("Content-Type", "video/mp2t")
	}
	resp.Header().Add("Cache-Control", "no-cache")
	//resp.Header().Add("Access-Control-Allow-Origin", "*")
	//resp.Header().Add("Access-Control-Allow-Credentials", "true")
	//resp.Header().Add("Access-Control-Allow-Methods", "*")
	//resp.Header().Add("Access-Control-Allow-Headers", "Content-Type,Access-Token")
	//resp.Header().Add("Access-Control-Allow-Expose-Headers", "*")
	_, _ = resp.Write(content)
	return
}
