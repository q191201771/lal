// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import (
	"net"
	"net/http"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/nazalog"
)

type Server struct {
	addr    string
	outPath string
	ln      net.Listener
	httpSrv *http.Server
}

func NewServer(addr string, outPath string) *Server {
	return &Server{
		addr:    addr,
		outPath: outPath,
	}
}

func (s *Server) Listen() (err error) {
	if s.ln, err = net.Listen("tcp", s.addr); err != nil {
		return
	}
	s.httpSrv = &http.Server{Addr: s.addr, Handler: s}
	nazalog.Infof("start hls server listen. addr=%s", s.addr)
	return
}

func (s *Server) RunLoop() error {
	return s.httpSrv.Serve(s.ln)
}

func (s *Server) Dispose() {
	if err := s.httpSrv.Close(); err != nil {
		nazalog.Error(err)
	}
}

func (s *Server) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	//nazalog.Debugf("%+v", req)

	// TODO chef:
	// - check appname in URI path

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
		resp.Header().Add("Server", base.LALHLSM3U8Server)
	case "ts":
		resp.Header().Add("Content-Type", "video/mp2t")
		resp.Header().Add("Server", base.LALHLSTSServer)
	}
	resp.Header().Add("Cache-Control", "no-cache")
	resp.Header().Add("Access-Control-Allow-Origin", "*")

	_, _ = resp.Write(content)
	return
}

// m3u8文件用这个也行
//resp.Header().Add("Content-Type", "application/vnd.apple.mpegurl")

//resp.Header().Add("Access-Control-Allow-Origin", "*")
//resp.Header().Add("Access-Control-Allow-Credentials", "true")
//resp.Header().Add("Access-Control-Allow-Methods", "*")
//resp.Header().Add("Access-Control-Allow-Headers", "Content-Type,Access-Token")
//resp.Header().Add("Access-Control-Allow-Expose-Headers", "*")
