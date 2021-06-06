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

type HttpSubSession struct {
	HttpSubSessionOption

	suffix       string
	conn         connection.Connection
	prevConnStat connection.Stat
	staleStat    *connection.Stat
	stat         StatSession
}

type HttpSubSessionOption struct {
	Conn          net.Conn
	ConnModOption connection.ModOption
	Uk            string // unique key
	Protocol      string
	UrlCtx        UrlContext
	IsWebSocket   bool
	WebSocketKey  string
}

func NewHttpSubSession(option HttpSubSessionOption) *HttpSubSession {
	s := &HttpSubSession{
		HttpSubSessionOption: option,
		conn:                 connection.New(option.Conn, option.ConnModOption),
		stat: StatSession{
			Protocol:   option.Protocol,
			SessionId:  option.Uk,
			StartTime:  time.Now().Format("2006-01-02 15:04:05.999"),
			RemoteAddr: option.Conn.RemoteAddr().String(),
		},
	}
	return s
}

func (session *HttpSubSession) RunLoop() error {
	buf := make([]byte, 128)
	_, err := session.conn.Read(buf)
	return err
}

func (session *HttpSubSession) WriteHttpResponseHeader(b []byte) {
	if session.IsWebSocket {
		session.write(UpdateWebSocketHeader(session.WebSocketKey))
	} else {
		session.write(b)
	}
}

func (session *HttpSubSession) Write(b []byte) {
	if session.IsWebSocket {
		wsHeader := WsHeader{
			Fin:           true,
			Rsv1:          false,
			Rsv2:          false,
			Rsv3:          false,
			Opcode:        Wso_Binary,
			PayloadLength: uint64(len(b)),
			Masked:        false,
		}
		session.write(MakeWsFrameHeader(wsHeader))
	}
	session.write(b)
}

func (session *HttpSubSession) Dispose() error {
	return session.conn.Close()
}

func (session *HttpSubSession) Url() string {
	return session.UrlCtx.Url
}

func (session *HttpSubSession) AppName() string {
	return session.UrlCtx.PathWithoutLastItem
}

func (session *HttpSubSession) StreamName() string {
	var suffix string
	switch session.Protocol {
	case ProtocolHttpflv:
		suffix = ".flv"
	case ProtocolHttpts:
		suffix = ".ts"
	default:
		Logger.Warnf("[%s] acquire stream name but protocol unknown.", session.Uk)
	}
	return strings.TrimSuffix(session.UrlCtx.LastItemOfPath, suffix)
}

func (session *HttpSubSession) RawQuery() string {
	return session.UrlCtx.RawQuery
}

func (session *HttpSubSession) UniqueKey() string {
	return session.Uk
}

func (session *HttpSubSession) GetStat() StatSession {
	currStat := session.conn.GetStat()
	session.stat.ReadBytesSum = currStat.ReadBytesSum
	session.stat.WroteBytesSum = currStat.WroteBytesSum
	return session.stat
}

func (session *HttpSubSession) UpdateStat(intervalSec uint32) {
	currStat := session.conn.GetStat()
	rDiff := currStat.ReadBytesSum - session.prevConnStat.ReadBytesSum
	session.stat.ReadBitrate = int(rDiff * 8 / 1024 / uint64(intervalSec))
	wDiff := currStat.WroteBytesSum - session.prevConnStat.WroteBytesSum
	session.stat.WriteBitrate = int(wDiff * 8 / 1024 / uint64(intervalSec))
	session.stat.Bitrate = session.stat.WriteBitrate
	session.prevConnStat = currStat
}

func (session *HttpSubSession) IsAlive() (readAlive, writeAlive bool) {
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

func (session *HttpSubSession) write(b []byte) {
	_, _ = session.conn.Write(b)
}
