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
)

type SubSession struct {
	core               *base.BasicHttpSubSession
	IsFresh            bool
	ShouldWaitBoundary bool
}

func NewSubSession(conn net.Conn, urlCtx base.UrlContext, isWebSocket bool, websocketKey string) *SubSession {
	s := &SubSession{
		core: base.NewBasicHttpSubSession(base.BasicHttpSubSessionOption{
			Conn: conn,
			ConnModOption: func(option *connection.Option) {
				option.WriteChanSize = SubSessionWriteChanSize
				option.WriteTimeoutMs = SubSessionWriteTimeoutMs
			},
			SessionType:  base.SessionTypeTsSub,
			UrlCtx:       urlCtx,
			IsWebSocket:  isWebSocket,
			WebSocketKey: websocketKey,
		}),
		IsFresh:            true,
		ShouldWaitBoundary: true,
	}
	Log.Infof("[%s] lifecycle new httpts SubSession. session=%p, remote addr=%s", s.UniqueKey(), s, conn.RemoteAddr().String())
	return s
}

// ---------------------------------------------------------------------------------------------------------------------
// IServerSessionLifecycle interface
// ---------------------------------------------------------------------------------------------------------------------

func (session *SubSession) RunLoop() error {
	return session.core.RunLoop()
}

func (session *SubSession) Dispose() error {
	Log.Infof("[%s] lifecycle dispose httpts SubSession.", session.core.UniqueKey())
	return session.core.Dispose()
}

// ---------------------------------------------------------------------------------------------------------------------

func (session *SubSession) WriteHttpResponseHeader() {
	Log.Debugf("[%s] > W http response header.", session.core.UniqueKey())
	session.core.WriteHttpResponseHeader(base.LalTsHttpResponseHeader)
}

func (session *SubSession) Write(b []byte) {
	session.core.Write(b)
}

// ---------------------------------------------------------------------------------------------------------------------
// IObject interface
// ---------------------------------------------------------------------------------------------------------------------

func (session *SubSession) UniqueKey() string {
	return session.core.UniqueKey()
}

// ---------------------------------------------------------------------------------------------------------------------
// ISessionUrlContext interface
// ---------------------------------------------------------------------------------------------------------------------

func (session *SubSession) Url() string {
	return session.core.Url()
}

func (session *SubSession) AppName() string {
	return session.core.AppName()
}

func (session *SubSession) StreamName() string {
	return session.core.StreamName()
}

func (session *SubSession) RawQuery() string {
	return session.core.RawQuery()
}

// ---------------------------------------------------------------------------------------------------------------------
// ISessionStat interface
// ---------------------------------------------------------------------------------------------------------------------

func (session *SubSession) UpdateStat(intervalSec uint32) {
	session.core.UpdateStat(intervalSec)
}

func (session *SubSession) GetStat() base.StatSession {
	return session.core.GetStat()
}

func (session *SubSession) IsAlive() (readAlive, writeAlive bool) {
	return session.core.IsAlive()
}
