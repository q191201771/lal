// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpts

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/q191201771/lal/pkg/mpegts"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
)

var tsHTTPResponseHeader []byte

type SubSession struct {
	UniqueKey string
	IsFresh   bool

	scheme string

	pathWithRawQuery string
	headers          map[string]string
	urlCtx           base.URLContext

	conn         connection.Connection
	prevConnStat connection.Stat
	staleStat    *connection.Stat
	stat         base.StatSession
}

func NewSubSession(conn net.Conn, scheme string) *SubSession {
	uk := base.GenUniqueKey(base.UKPTSSubSession)
	s := &SubSession{
		UniqueKey: uk,
		scheme:    scheme,
		IsFresh:   true,
		conn: connection.New(conn, func(option *connection.Option) {
			option.ReadBufSize = readBufSize
			option.WriteChanSize = wChanSize
			option.WriteTimeoutMS = subSessionWriteTimeoutMS
		}),
		stat: base.StatSession{
			Protocol:   base.ProtocolHTTPTS,
			SessionID:  uk,
			StartTime:  time.Now().Format("2006-01-02 15:04:05.999"),
			RemoteAddr: conn.RemoteAddr().String(),
		},
	}
	nazalog.Infof("[%s] lifecycle new httpts SubSession. session=%p, remote addr=%s", uk, s, conn.RemoteAddr().String())
	return s
}

// TODO chef: read request timeout
func (session *SubSession) ReadRequest() (err error) {
	defer func() {
		if err != nil {
			session.Dispose()
		}
	}()

	var requestLine string
	if requestLine, session.headers, err = nazahttp.ReadHTTPHeader(session.conn); err != nil {
		return
	}
	if _, session.pathWithRawQuery, _, err = nazahttp.ParseHTTPRequestLine(requestLine); err != nil {
		return
	}

	rawURL := fmt.Sprintf("%s://%s%s", session.scheme, session.headers["Host"], session.pathWithRawQuery)
	_ = rawURL

	session.urlCtx, err = base.ParseHTTPTSURL(rawURL, session.scheme == "https")
	return
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

func (session *SubSession) UpdateStat(interval uint32) {
	currStat := session.conn.GetStat()
	rDiff := currStat.ReadBytesSum - session.prevConnStat.ReadBytesSum
	session.stat.ReadBitrate = int(rDiff * 8 / 1024 / uint64(interval))
	wDiff := currStat.WroteBytesSum - session.prevConnStat.WroteBytesSum
	session.stat.WriteBitrate = int(wDiff * 8 / 1024 / uint64(interval))
	session.stat.Bitrate = session.stat.WriteBitrate
	session.prevConnStat = currStat
}

func (session *SubSession) GetStat() base.StatSession {
	connStat := session.conn.GetStat()
	session.stat.ReadBytesSum = connStat.ReadBytesSum
	session.stat.WroteBytesSum = connStat.WroteBytesSum
	return session.stat
}

func (session *SubSession) IsAlive() (readAlive, writeAlive bool) {
	currStat := session.conn.GetStat()
	if session.staleStat == nil {
		session.staleStat = new(connection.Stat)
		*session.staleStat = currStat
		return true, true
	}

	readAlive = !(currStat.ReadBytesSum-session.staleStat.ReadBytesSum == 0)
	writeAlive = !(currStat.WroteBytesSum-session.staleStat.WroteBytesSum == 0)
	*session.staleStat = currStat
	return
}

func (session *SubSession) URL() string {
	return session.urlCtx.URL
}

func (session *SubSession) AppName() string {
	return session.urlCtx.PathWithoutLastItem
}

func (session *SubSession) StreamName() string {
	return strings.TrimSuffix(session.urlCtx.LastItemOfPath, ".ts")
}

func (session *SubSession) RawQuery() string {
	return session.urlCtx.RawQuery
}

func (session *SubSession) RemoteAddr() string {
	return session.conn.RemoteAddr().String()
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
