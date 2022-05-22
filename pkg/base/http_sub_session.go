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

	"github.com/q191201771/naza/pkg/connection"
)

// TODO(chef): refactor 更名为BasicHttpSubSession 202205

type HttpSubSession struct {
	HttpSubSessionOption

	suffix      string
	conn        connection.Connection
	sessionStat BasicSessionStat
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
		sessionStat: BasicSessionStat{
			Stat: StatSession{
				SessionId:  option.Uk,
				Protocol:   option.Protocol,
				BaseType:   SessionBaseTypeSubStr,
				StartTime:  ReadableNowTime(),
				RemoteAddr: option.Conn.RemoteAddr().String(),
			},
		},
	}
	return s
}

// ---------------------------------------------------------------------------------------------------------------------
// IServerSessionLifecycle interface
// ---------------------------------------------------------------------------------------------------------------------

func (session *HttpSubSession) RunLoop() error {
	buf := make([]byte, 128)
	_, err := session.conn.Read(buf)
	return err
}

func (session *HttpSubSession) Dispose() error {
	return session.conn.Close()
}

// ---------------------------------------------------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------------------------------------------------
// IObject interface
// ---------------------------------------------------------------------------------------------------------------------

func (session *HttpSubSession) UniqueKey() string {
	return session.Uk
}

// ---------------------------------------------------------------------------------------------------------------------
// ISessionUrlContext interface
// ---------------------------------------------------------------------------------------------------------------------

func (session *HttpSubSession) Url() string {
	return session.UrlCtx.Url
}

func (session *HttpSubSession) AppName() string {
	return session.UrlCtx.PathWithoutLastItem
}

func (session *HttpSubSession) StreamName() string {
	var suffix string
	switch session.Protocol {
	case SessionProtocolFlvStr:
		suffix = ".flv"
	case SessionProtocolTsStr:
		suffix = ".ts"
	default:
		Log.Warnf("[%s] acquire stream name but protocol unknown.", session.Uk)
	}
	return strings.TrimSuffix(session.UrlCtx.LastItemOfPath, suffix)
}

func (session *HttpSubSession) RawQuery() string {
	return session.UrlCtx.RawQuery
}

// ----- ISessionStat --------------------------------------------------------------------------------------------------

func (session *HttpSubSession) GetStat() StatSession {
	return session.sessionStat.GetStat()
}

func (session *HttpSubSession) UpdateStat(intervalSec uint32) {
	session.sessionStat.UpdateStatWitchConn(session.conn, intervalSec)
}

func (session *HttpSubSession) IsAlive() (readAlive, writeAlive bool) {
	return session.sessionStat.IsAliveWitchConn(session.conn)
}

// ---------------------------------------------------------------------------------------------------------------------

func (session *HttpSubSession) write(b []byte) {
	// TODO(chef) handle write error
	_, _ = session.conn.Write(b)
}
