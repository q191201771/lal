// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package gb28181

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazanet"
	"net"
	"sync"
)

type PubSession struct {
	unpacker *PsUnpacker

	streamName string

	conn        *nazanet.UdpConnection
	sessionStat base.BasicSessionStat

	disposeOnce sync.Once
}

func NewPubSession() *PubSession {
	return &PubSession{
		unpacker:    NewPsUnpacker(),
		sessionStat: base.NewBasicSessionStat(base.SessionTypePsPub, ""),
	}
}

func (session *PubSession) WithOnAvPacket(onAvPacket base.OnAvPacketFunc) *PubSession {
	session.unpacker.WithOnAvPacket(onAvPacket)
	return session
}

func (session *PubSession) WithStreamName(streamName string) *PubSession {
	session.streamName = streamName
	return session
}

func (session *PubSession) RunLoop(addr string) error {
	var err error
	session.conn, err = nazanet.NewUdpConnection(func(option *nazanet.UdpConnectionOption) {
		option.LAddr = addr
	})
	if err != nil {
		return err
	}
	err = session.conn.RunLoop(func(b []byte, raddr *net.UDPAddr, err error) bool {
		session.sessionStat.AddReadBytes(len(b))
		session.unpacker.FeedRtpPacket(b)
		return true
	})
	return err
}

// ----- IServerSessionLifecycle ---------------------------------------------------------------------------------------

func (session *PubSession) Dispose() error {
	return session.dispose(nil)
}

// ----- ISessionUrlContext --------------------------------------------------------------------------------------------

func (session *PubSession) Url() string {
	Log.Warnf("[%s] PubSession.Url() is not implemented", session.UniqueKey())
	return "invalid"
}

func (session *PubSession) AppName() string {
	Log.Warnf("[%s] PubSession.AppName() is not implemented", session.UniqueKey())
	return "invalid"
}

func (session *PubSession) StreamName() string {
	// 如果stream name没有设置，则使用session的unique key作为stream name
	if session.streamName == "" {
		return session.UniqueKey()
	}
	return session.streamName
}

func (session *PubSession) RawQuery() string {
	Log.Warnf("[%s] PubSession.RawQuery() is not implemented", session.UniqueKey())
	return "invalid"
}

// ----- IObject -------------------------------------------------------------------------------------------------------

func (session *PubSession) UniqueKey() string {
	return session.sessionStat.UniqueKey()
}

// ----- ISessionStat --------------------------------------------------------------------------------------------------

func (session *PubSession) UpdateStat(intervalSec uint32) {
	session.sessionStat.UpdateStat(intervalSec)
}

func (session *PubSession) GetStat() base.StatSession {
	return session.sessionStat.GetStat()
}

func (session *PubSession) IsAlive() (readAlive, writeAlive bool) {
	return session.sessionStat.IsAlive()
}

// ---------------------------------------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------------------------------------

func (session *PubSession) dispose(err error) error {
	var retErr error
	session.disposeOnce.Do(func() {
		Log.Infof("[%s] lifecycle dispose rtmp ServerSession. err=%+v", session.UniqueKey(), err)
		if session.conn == nil {
			retErr = base.ErrSessionNotStarted
			return
		}
		retErr = session.conn.Dispose()
	})
	return retErr
}
