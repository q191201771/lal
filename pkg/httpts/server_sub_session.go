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
	"net/url"
	"strings"
	"time"

	"github.com/q191201771/lal/pkg/mpegts"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"
)

var tsHTTPResponseHeader []byte

type SubSession struct {
	UniqueKey string

	StartTick  int64
	StreamName string
	AppName    string
	URI        string
	Headers    map[string]string

	IsFresh bool

	conn connection.Connection
}

func NewSubSession(conn net.Conn) *SubSession {
	uk := unique.GenUniqueKey("TSSUB")
	s := &SubSession{
		UniqueKey: uk,
		IsFresh:   true,
		conn: connection.New(conn, func(option *connection.Option) {
			option.ReadBufSize = readBufSize
			option.WriteChanSize = wChanSize
			option.WriteTimeoutMS = subSessionWriteTimeoutMS
		}),
	}
	nazalog.Infof("[%s] lifecycle new httpts SubSession. session=%p, remote addr=%s", uk, s, conn.RemoteAddr().String())
	return s
}

// TODO chef: read request timeout
func (session *SubSession) ReadRequest() (err error) {
	session.StartTick = time.Now().Unix()

	defer func() {
		if err != nil {
			session.Dispose()
		}
	}()

	var (
		requestLine string
		method      string
	)
	if requestLine, session.Headers, err = nazahttp.ReadHTTPHeader(session.conn); err != nil {
		return
	}
	if method, session.URI, _, err = nazahttp.ParseHTTPRequestLine(requestLine); err != nil {
		return
	}
	if method != "GET" {
		err = ErrHTTPTS
		return
	}

	var urlObj *url.URL
	if urlObj, err = url.Parse(session.URI); err != nil {
		return
	}
	if !strings.HasSuffix(urlObj.Path, ".ts") {
		err = ErrHTTPTS
		return
	}

	items := strings.Split(urlObj.Path, "/")
	if len(items) != 3 {
		err = ErrHTTPTS
		return
	}
	session.AppName = items[1]
	items = strings.Split(items[2], ".")
	if len(items) < 2 {
		err = ErrHTTPTS
		return
	}
	session.StreamName = items[0]

	return nil
}

func (session *SubSession) RunLoop() error {
	buf := make([]byte, 128)
	_, err := session.conn.Read(buf)
	return err
}

func (session *SubSession) WriteHTTPResponseHeader() {
	nazalog.Debugf("[%s] > W http response header.", session.UniqueKey)
	session.WriteRawPacket(tsHTTPResponseHeader)
}

func (session *SubSession) WriteFragmentHeader() {
	nazalog.Debugf("[%s] > W http response header.", session.UniqueKey)
	session.WriteRawPacket(mpegts.FixedFragmentHeader)
}

func (session *SubSession) WriteRawPacket(pkt []byte) {
	_, _ = session.conn.Write(pkt)
}

func (session *SubSession) Dispose() {
	nazalog.Infof("[%s] lifecycle dispose httpts SubSession.", session.UniqueKey)
	_ = session.conn.Close()
}

func init() {
	tsHTTPResponseHeaderStr := "HTTP/1.1 200 OK\r\n" +
		"Server: " + base.LALHTTPTSSubSessionServer + "\r\n" +
		"Cache-Control: no-cache\r\n" +
		"Content-Type: video/mp2t\r\n" +
		"Connection: close\r\n" +
		"Expires: -1\r\n" +
		"Pragma: no-cache\r\n" +
		"Access-Control-Allow-Origin: *\r\n" +
		"\r\n"

	tsHTTPResponseHeader = []byte(tsHTTPResponseHeaderStr)
}
