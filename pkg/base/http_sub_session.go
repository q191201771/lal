// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"net"
	"strings"
	"time"

	"github.com/q191201771/naza/pkg/connection"
)

type HTTPSubSession struct {
	HTTPSubSessionOption

	suffix       string
	conn         connection.Connection
	prevConnStat connection.Stat
	staleStat    *connection.Stat
	stat         StatSession
}

type HTTPSubSessionOption struct {
	Conn          net.Conn
	ConnModOption connection.ModOption
	UK            string // unique key
	Protocol      string
	URLCtx        URLContext
	IsWebSocket   bool
	WebSocketKey  string
}

func NewHTTPSubSession(option HTTPSubSessionOption) *HTTPSubSession {
	s := &HTTPSubSession{
		HTTPSubSessionOption: option,
		conn:                 connection.New(option.Conn, option.ConnModOption),
		stat: StatSession{
			Protocol:   option.Protocol,
			SessionID:  option.UK,
			StartTime:  time.Now().Format("2006-01-02 15:04:05.999"),
			RemoteAddr: option.Conn.RemoteAddr().String(),
		},
	}
	return s
}

func (session *HTTPSubSession) RunLoop() error {
	buf := make([]byte, 128)
	_, err := session.conn.Read(buf)
	return err
}

func (session *HTTPSubSession) WriteHTTPResponseHeader(b []byte) {
	if session.IsWebSocket {
		session.Write(UpdateWebSocketHeader(session.WebSocketKey))
	} else {
		session.Write(b)
	}
}

func (session *HTTPSubSession) Write(b []byte) {
	if session.IsWebSocket {
		wsHeader := WSHeader{
			Fin:           true,
			Rsv1:          false,
			Rsv2:          false,
			Rsv3:          false,
			Opcode:        WSO_Binary,
			PayloadLength: uint64(len(b)),
			Masked:        false,
		}
		session.write(MakeWSFrameHeader(wsHeader))
	}
	session.write(b)
}

func (session *HTTPSubSession) Dispose() error {
	return session.conn.Close()
}

func (session *HTTPSubSession) URL() string {
	return session.URLCtx.URL
}

func (session *HTTPSubSession) AppName() string {
	return session.URLCtx.PathWithoutLastItem
}

func (session *HTTPSubSession) StreamName() string {
	var suffix string
	switch session.Protocol {
	case ProtocolHTTPFLV:
		suffix = ".flv"
	case ProtocolHTTPTS:
		suffix = ".ts"
	default:
		Logger.Warnf("[%s] acquire stream name but protocol unknown.", session.UK)
	}
	return strings.TrimSuffix(session.URLCtx.LastItemOfPath, suffix)
}

func (session *HTTPSubSession) RawQuery() string {
	return session.URLCtx.RawQuery
}

func (session *HTTPSubSession) UniqueKey() string {
	return session.UK
}

func (session *HTTPSubSession) GetStat() StatSession {
	currStat := session.conn.GetStat()
	session.stat.ReadBytesSum = currStat.ReadBytesSum
	session.stat.WroteBytesSum = currStat.WroteBytesSum
	return session.stat
}

func (session *HTTPSubSession) UpdateStat(intervalSec uint32) {
	currStat := session.conn.GetStat()
	rDiff := currStat.ReadBytesSum - session.prevConnStat.ReadBytesSum
	session.stat.ReadBitrate = int(rDiff * 8 / 1024 / uint64(intervalSec))
	wDiff := currStat.WroteBytesSum - session.prevConnStat.WroteBytesSum
	session.stat.WriteBitrate = int(wDiff * 8 / 1024 / uint64(intervalSec))
	session.stat.Bitrate = session.stat.WriteBitrate
	session.prevConnStat = currStat
}

func (session *HTTPSubSession) IsAlive() (readAlive, writeAlive bool) {
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

func (session *HTTPSubSession) write(b []byte) {
	_, _ = session.conn.Write(b)
}
