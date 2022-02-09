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

	"github.com/q191201771/lal/pkg/base"
)

type ServerHandler struct {
	outPath string
}

func NewServerHandler(outPath string) *ServerHandler {
	return &ServerHandler{
		outPath: outPath,
	}
}

func (s *ServerHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	urlCtx, err := base.ParseUrl(base.ParseHttpRequest(req), 80)
	if err != nil {
		Log.Errorf("parse url. err=%+v", err)
		return
	}
	s.ServeHTTPWithUrlCtx(resp, urlCtx)
}

func (s *ServerHandler) ServeHTTPWithUrlCtx(resp http.ResponseWriter, urlCtx base.UrlContext) {
	//Log.Debugf("%+v", req)

	// TODO chef:
	// - check appname in URI path

	filename := urlCtx.LastItemOfPath
	filetype := urlCtx.GetFileType()

	ri := PathStrategy.GetRequestInfo(urlCtx, s.outPath)
	//Log.Debugf("%+v", ri)

	if filename == "" || (filetype != "m3u8" && filetype != "ts") || ri.StreamName == "" || ri.FileNameWithPath == "" {
		Log.Warnf("invalid hls request. url=%+v, request=%+v", urlCtx, ri)
		resp.WriteHeader(404)
		return
	}

	content, err := ReadFile(ri.FileNameWithPath)
	if err != nil {
		Log.Warnf("read hls file failed. request=%+v, err=%+v", ri, err)
		resp.WriteHeader(404)
		return
	}

	switch filetype {
	case "m3u8":
		resp.Header().Add("Content-Type", "application/x-mpegurl")
		resp.Header().Add("Server", base.LalHlsM3u8Server)
	case "ts":
		resp.Header().Add("Content-Type", "video/mp2t")
		resp.Header().Add("Server", base.LalHlsTsServer)
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
