// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpts

import (
	"net"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazalog"
)

var tsHTTPResponseHeader []byte

type SubSession struct {
	*base.HTTPSubSession // 直接使用它提供的函数
	IsFresh              bool
}

func NewSubSession(conn net.Conn, urlCtx base.URLContext, isWebSocket bool, websocketKey string) *SubSession {
	uk := base.GenUKTSSubSession()
	s := &SubSession{
		base.NewHTTPSubSession(base.HTTPSubSessionOption{
			Conn: conn,
			ConnModOption: func(option *connection.Option) {
				option.WriteChanSize = SubSessionWriteChanSize
				option.WriteTimeoutMS = SubSessionWriteTimeoutMS
			},
			UK:           uk,
			Protocol:     base.ProtocolHTTPTS,
			URLCtx:       urlCtx,
			IsWebSocket:  isWebSocket,
			WebSocketKey: websocketKey,
		}),
		true,
	}
	nazalog.Infof("[%s] lifecycle new httpts SubSession. session=%p, remote addr=%s", uk, s, conn.RemoteAddr().String())
	return s
}

func (session *SubSession) WriteHTTPResponseHeader() {
	nazalog.Debugf("[%s] > W http response header.", session.UniqueKey())
	session.HTTPSubSession.WriteHTTPResponseHeader(tsHTTPResponseHeader)
}

func (session *SubSession) Dispose() error {
	nazalog.Infof("[%s] lifecycle dispose httpts SubSession.", session.UniqueKey())
	return session.HTTPSubSession.Dispose()
}

func init() {
	tsHTTPResponseHeaderStr := "HTTP/1.1 200 OK\r\n" +
		"Server: " + base.LALHTTPTSSubSessionServer + "\r\n" +
		"Cache-Control: no-cache\r\n" +
		"Content-Type: video/mp2t\r\n" +
		"Connection: close\r\n" +
		"Expires: -1\r\n" +
		"Pragma: no-cache\r\n" +
		"Access-Control-Allow-Credentials: true\r\n" +
		"Access-Control-Allow-Origin: *\r\n" +
		"\r\n"

	tsHTTPResponseHeader = []byte(tsHTTPResponseHeaderStr)
}
