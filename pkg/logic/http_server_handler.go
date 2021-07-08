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

type HttpServerHandlerObserver interface {
	// 通知上层有新的拉流者
	// 返回值： true则允许拉流，false则关闭连接
	OnNewHttpflvSubSession(session *httpflv.SubSession) bool

	OnDelHttpflvSubSession(session *httpflv.SubSession)

	OnNewHttptsSubSession(session *httpts.SubSession) bool
	OnDelHttptsSubSession(session *httpts.SubSession)
}

type HttpServerHandler struct {
	observer HttpServerHandlerObserver
}

func NewHttpServerHandler(observer HttpServerHandlerObserver) *HttpServerHandler {
	return &HttpServerHandler{
		observer: observer,
	}
}

func (h *HttpServerHandler) ServeSubSession(writer http.ResponseWriter, req *http.Request) {
	var (
		isHttps bool
		scheme  string
	)
	// TODO(chef) 这里scheme直接使用http和https，没有考虑ws和wss，注意，后续的逻辑可能会依赖此处
	if req.TLS == nil {
		isHttps = false
		scheme = "http"
	} else {
		isHttps = true
		scheme = "https"
	}
	rawUrl := fmt.Sprintf("%s://%s%s", scheme, req.Host, req.RequestURI)

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

	if strings.HasSuffix(rawUrl, ".flv") {
		urlCtx, err := base.ParseHttpUrl(rawUrl, isHttps, ".flv")
		if err != nil {
			nazalog.Errorf("parse http url failed. err=%+v", err)
			_ = conn.Close()
			return
		}

		session := httpflv.NewSubSession(conn, urlCtx, isWebSocket, webSocketKey)
		nazalog.Debugf("[%s] < read http request. url=%s", session.UniqueKey(), session.Url())
		if !h.observer.OnNewHttpflvSubSession(session) {
			session.Dispose()
		}
		err = session.RunLoop()
		nazalog.Debugf("[%s] httpflv sub session loop done. err=%v", session.UniqueKey(), err)
		h.observer.OnDelHttpflvSubSession(session)
		return
	}

	if strings.HasSuffix(rawUrl, ".ts") {
		urlCtx, err := base.ParseHttpUrl(rawUrl, isHttps, ".ts")
		if err != nil {
			nazalog.Errorf("parse http url failed. err=%+v", err)
			_ = conn.Close()
			return
		}

		session := httpts.NewSubSession(conn, urlCtx, isWebSocket, webSocketKey)
		nazalog.Debugf("[%s] < read http request. url=%s", session.UniqueKey(), session.Url())
		if !h.observer.OnNewHttptsSubSession(session) {
			session.Dispose()
		}
		err = session.RunLoop()
		nazalog.Debugf("[%s] httpts sub session loop done. err=%v", session.UniqueKey(), err)
		h.observer.OnDelHttptsSubSession(session)
		return
	}
}
