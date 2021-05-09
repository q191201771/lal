// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv

import (
	"net"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/connection"

	"github.com/q191201771/naza/pkg/nazalog"
)

var flvHTTPResponseHeader []byte

type SubSession struct {
	*base.HTTPSubSession // 直接使用它提供的函数
	IsFresh              bool
}

func NewSubSession(conn net.Conn, urlCtx base.URLContext, isWebSocket bool, websocketKey string) *SubSession {
	uk := base.GenUKFLVSubSession()
	s := &SubSession{
		base.NewHTTPSubSession(base.HTTPSubSessionOption{
			Conn: conn,
			ConnModOption: func(option *connection.Option) {
				option.WriteChanSize = SubSessionWriteChanSize
				option.WriteTimeoutMS = SubSessionWriteTimeoutMS
			},
			UK:           uk,
			Protocol:     base.ProtocolHTTPFLV,
			URLCtx:       urlCtx,
			IsWebSocket:  isWebSocket,
			WebSocketKey: websocketKey,
		}),
		true,
	}
	nazalog.Infof("[%s] lifecycle new httpflv SubSession. session=%p, remote addr=%s", uk, s, conn.RemoteAddr().String())
	return s
}

func (session *SubSession) WriteHTTPResponseHeader() {
	nazalog.Debugf("[%s] > W http response header.", session.UniqueKey())
	session.HTTPSubSession.WriteHTTPResponseHeader(flvHTTPResponseHeader)
}

func (session *SubSession) WriteFLVHeader() {
	nazalog.Debugf("[%s] > W http flv header.", session.UniqueKey())
	session.Write(FLVHeader)
}

func (session *SubSession) WriteTag(tag *Tag) {
	session.Write(tag.Raw)
}

func (session *SubSession) Dispose() error {
	nazalog.Infof("[%s] lifecycle dispose httpflv SubSession.", session.UniqueKey())
	return session.HTTPSubSession.Dispose()
}

func init() {
	flvHTTPResponseHeaderStr := "HTTP/1.1 200 OK\r\n" +
		"Server: " + base.LALHTTPFLVSubSessionServer + "\r\n" +
		"Cache-Control: no-cache\r\n" +
		"Content-Type: video/x-flv\r\n" +
		"Connection: close\r\n" +
		"Expires: -1\r\n" +
		"Pragma: no-cache\r\n" +
		"Access-Control-Allow-Credentials: true\r\n" +
		"Access-Control-Allow-Origin: *\r\n" +
		"\r\n"

	flvHTTPResponseHeader = []byte(flvHTTPResponseHeaderStr)
}
