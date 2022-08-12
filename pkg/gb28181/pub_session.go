// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package gb28181

import (
	"fmt"
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazanet"
	"net"
	"sync"
)

type PubSession struct {
	unpacker *PsUnpacker

	streamName string

	hookOnReadUdpPacket nazanet.OnReadUdpPacket

	disposeOnce sync.Once
	conn        *nazanet.UdpConnection
	sessionStat base.BasicSessionStat
}

func NewPubSession() *PubSession {
	return &PubSession{
		unpacker:    NewPsUnpacker(),
		sessionStat: base.NewBasicSessionStat(base.SessionTypePsPub, ""),
	}
}

// WithOnAvPacket 设置音视频的回调
//
//  @param onAvPacket 见 PsUnpacker.WithOnAvPacket 的注释
//
func (session *PubSession) WithOnAvPacket(onAvPacket base.OnAvPacketFunc) *PubSession {
	session.unpacker.WithOnAvPacket(onAvPacket)
	return session
}

func (session *PubSession) WithStreamName(streamName string) *PubSession {
	session.streamName = streamName
	return session
}

// WithHookReadUdpPacket
//
// 将udp接收数据返回给上层。
// 注意，底层的解析逻辑依然走。
// 可以用这个方式来截取数据进行调试。
//
func (session *PubSession) WithHookReadUdpPacket(fn nazanet.OnReadUdpPacket) *PubSession {
	session.hookOnReadUdpPacket = fn
	return session
}

// Listen
//
// 注意，当`port`参数为0时，内部会自动选择一个可用端口监听，并通过返回值返回该端口
//
func (session *PubSession) Listen(port int) (int, error) {
	var err error
	var uconn *net.UDPConn
	var addr string

	if port == 0 {
		uconn, _, err = defaultUdpConnPoll.Acquire()
		if err != nil {
			return -1, err
		}

		port = uconn.LocalAddr().(*net.UDPAddr).Port
	} else {
		addr = fmt.Sprintf(":%d", port)
	}

	session.conn, err = nazanet.NewUdpConnection(func(option *nazanet.UdpConnectionOption) {
		option.LAddr = addr
		option.Conn = uconn
	})
	return port, err
}

// RunLoop ...
//
func (session *PubSession) RunLoop() error {
	err := session.conn.RunLoop(func(b []byte, raddr *net.UDPAddr, err error) bool {
		if len(b) == 0 && err != nil {
			return false
		}

		if session.hookOnReadUdpPacket != nil {
			session.hookOnReadUdpPacket(b, raddr, err)
		}

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

func (session *PubSession) dispose(err error) error {
	var retErr error
	session.disposeOnce.Do(func() {
		Log.Infof("[%s] lifecycle dispose gb28181 PubSession. err=%+v", session.UniqueKey(), err)
		if session.conn == nil {
			retErr = base.ErrSessionNotStarted
			return
		}
		retErr = session.conn.Dispose()
	})
	return retErr
}
