// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/httpts"
	"github.com/q191201771/naza/pkg/nazalog"
)

type HTTPServerHandlerObserver interface {
	// 通知上层有新的拉流者
	// 返回值： true则允许拉流，false则关闭连接
	OnNewHTTPFLVSubSession(session *httpflv.SubSession) bool

	OnDelHTTPFLVSubSession(session *httpflv.SubSession)

	OnNewHTTPTSSubSession(session *httpts.SubSession) bool
	OnDelHTTPTSSubSession(session *httpts.SubSession)
}

type HTTPServerHandler struct {
	observer HTTPServerHandlerObserver
}

func NewHTTPServerHandler(observer HTTPServerHandlerObserver) *HTTPServerHandler {
	return &HTTPServerHandler{
		observer: observer,
	}
}

func (h *HTTPServerHandler) ServeSubSession(writer http.ResponseWriter, req *http.Request) {
	var (
		isHTTPS bool
		scheme  string
	)
	// TODO(chef) 这里scheme直接使用http和https，没有考虑ws和wss，注意，后续的逻辑可能会依赖此处
	if req.TLS == nil {
		isHTTPS = false
		scheme = "http"
	} else {
		isHTTPS = true
		scheme = "https"
	}
	rawURL := fmt.Sprintf("%s://%s%s", scheme, req.Host, req.RequestURI)

	conn, bio, err := writer.(http.Hijacker).Hijack()
	if err != nil {
		nazalog.Errorf("hijack failed. err=%+v", err)
		return
	}
	if bio.Reader.Buffered() != 0 || bio.Writer.Buffered() != 0 {
		nazalog.Errorf("hijack but buffer not empty. rb=%d, wb=%d", bio.Reader.Buffered(), bio.Writer.Buffered())
	}

	var (
		isWebSocket  bool
		webSocketKey string
	)
	if req.Header.Get("Connection") == "Upgrade" && req.Header.Get("Upgrade") == "websocket" {
		isWebSocket = true
		webSocketKey = req.Header.Get("Sec-WebSocket-Key")
	}

	if strings.HasSuffix(rawURL, ".flv") {
		urlCtx, err := base.ParseHTTPURL(rawURL, isHTTPS, ".flv")
		if err != nil {
			nazalog.Errorf("parse http url failed. err=%+v", err)
			_ = conn.Close()
			return
		}

		session := httpflv.NewSubSession(conn, urlCtx, isWebSocket, webSocketKey)
		nazalog.Debugf("[%s] < read http request. url=%s", session.UniqueKey(), session.URL())
		if !h.observer.OnNewHTTPFLVSubSession(session) {
			session.Dispose()
		}
		err = session.RunLoop()
		nazalog.Debugf("[%s] httpflv sub session loop done. err=%v", session.UniqueKey(), err)
		h.observer.OnDelHTTPFLVSubSession(session)
		return
	}

	if strings.HasSuffix(rawURL, ".ts") {
		urlCtx, err := base.ParseHTTPURL(rawURL, isHTTPS, ".ts")
		if err != nil {
			nazalog.Errorf("parse http url failed. err=%+v", err)
			_ = conn.Close()
			return
		}

		session := httpts.NewSubSession(conn, urlCtx, isWebSocket, webSocketKey)
		nazalog.Debugf("[%s] < read http request. url=%s", session.UniqueKey(), session.URL())
		if !h.observer.OnNewHTTPTSSubSession(session) {
			session.Dispose()
		}
		err = session.RunLoop()
		nazalog.Debugf("[%s] httpts sub session loop done. err=%v", session.UniqueKey(), err)
		h.observer.OnDelHTTPTSSubSession(session)
		return
	}
}
