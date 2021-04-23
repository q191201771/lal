// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/nazahttp"

	"github.com/q191201771/naza/pkg/connection"

	"github.com/q191201771/naza/pkg/nazalog"
)

var flvHTTPResponseHeader []byte

type SubSession struct {
	uniqueKey string
	IsFresh   bool

	scheme string

	pathWithRawQuery string
	headers          map[string]string
	urlCtx           base.URLContext

	conn         connection.Connection
	prevConnStat connection.Stat
	staleStat    *connection.Stat
	stat         base.StatSession
	isWebSocket  bool
}

func NewSubSession(conn net.Conn, scheme string) *SubSession {
	uk := base.GenUKFLVSubSession()
	s := &SubSession{
		uniqueKey: uk,
		scheme:    scheme,
		IsFresh:   true,
		conn: connection.New(conn, func(option *connection.Option) {
			option.ReadBufSize = readBufSize
			option.WriteChanSize = wChanSize
			option.WriteTimeoutMS = subSessionWriteTimeoutMS
		}),
		stat: base.StatSession{
			Protocol:   base.ProtocolHTTPFLV,
			SessionID:  uk,
			StartTime:  time.Now().Format("2006-01-02 15:04:05.999"),
			RemoteAddr: conn.RemoteAddr().String(),
		},
	}
	nazalog.Infof("[%s] lifecycle new httpflv SubSession. session=%p, remote addr=%s", uk, s, conn.RemoteAddr().String())
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

	session.urlCtx, err = base.ParseHTTPFLVURL(rawURL, session.scheme == "https")
	if session.headers["Connection"] == "Upgrade" && session.headers["Upgrade"] == "websocket" {
		session.isWebSocket = true
		//回复升级为websocket
		session.writeRawPacket(base.UpdateWebSocketHeader(session.headers["Sec-WebSocket-Key"]))
	}
	return
}

func (session *SubSession) RunLoop() error {
	buf := make([]byte, 128)
	_, err := session.conn.Read(buf)
	return err
}

func (session *SubSession) WriteHTTPResponseHeader() {
	nazalog.Debugf("[%s] > W http response header.", session.uniqueKey)
	if session.isWebSocket {

	} else {
		session.WriteRawPacket(flvHTTPResponseHeader)
	}
}

func (session *SubSession) WriteFLVHeader() {
	nazalog.Debugf("[%s] > W http flv header.", session.uniqueKey)
	session.WriteRawPacket(FLVHeader)

}

func (session *SubSession) WriteTag(tag *Tag) {
	session.WriteRawPacket(tag.Raw)

}

func (session *SubSession) WriteRawPacket(pkt []byte) {
	if session.isWebSocket {
		wsHeader := base.WSHeader{
			Fin:           true,
			Rsv1:          false,
			Rsv2:          false,
			Rsv3:          false,
			Opcode:        base.WSO_Binary,
			PayloadLength: uint64(len(pkt)),
			Masked:        false,
		}
		session.writeRawPacket(base.MakeWSFrameHeader(wsHeader))
	}
	session.writeRawPacket(pkt)
}
func (session *SubSession) writeRawPacket(pkt []byte) {
	_, _ = session.conn.Write(pkt)
}

func (session *SubSession) Dispose() error {
	nazalog.Infof("[%s] lifecycle dispose httpflv SubSession.", session.uniqueKey)
	return session.conn.Close()
}

func (session *SubSession) URL() string {
	return session.urlCtx.URL
}

func (session *SubSession) AppName() string {
	return session.urlCtx.PathWithoutLastItem
}

func (session *SubSession) StreamName() string {
	return strings.TrimSuffix(session.urlCtx.LastItemOfPath, ".flv")
}

func (session *SubSession) RawQuery() string {
	return session.urlCtx.RawQuery
}

func (session *SubSession) UniqueKey() string {
	return session.uniqueKey
}

func (session *SubSession) GetStat() base.StatSession {
	currStat := session.conn.GetStat()
	session.stat.ReadBytesSum = currStat.ReadBytesSum
	session.stat.WroteBytesSum = currStat.WroteBytesSum
	return session.stat
}

func (session *SubSession) UpdateStat(intervalSec uint32) {
	currStat := session.conn.GetStat()
	rDiff := currStat.ReadBytesSum - session.prevConnStat.ReadBytesSum
	session.stat.ReadBitrate = int(rDiff * 8 / 1024 / uint64(intervalSec))
	wDiff := currStat.WroteBytesSum - session.prevConnStat.WroteBytesSum
	session.stat.WriteBitrate = int(wDiff * 8 / 1024 / uint64(intervalSec))
	session.stat.Bitrate = session.stat.WriteBitrate
	session.prevConnStat = currStat
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
