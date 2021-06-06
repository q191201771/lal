// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/nazaerrors"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
)

type PushSessionOption struct {
	PushTimeoutMs int
	OverTcp       bool
}

var defaultPushSessionOption = PushSessionOption{
	PushTimeoutMs: 10000,
	OverTcp:       false,
}

type PushSession struct {
	uniqueKey      string
	cmdSession     *ClientCommandSession
	baseOutSession *BaseOutSession
}

type ModPushSessionOption func(option *PushSessionOption)

func NewPushSession(modOptions ...ModPushSessionOption) *PushSession {
	option := defaultPushSessionOption
	for _, fn := range modOptions {
		fn(&option)
	}

	uk := base.GenUkRtspPushSession()
	s := &PushSession{
		uniqueKey: uk,
	}
	cmdSession := NewClientCommandSession(CcstPushSession, uk, s, func(opt *ClientCommandSessionOption) {
		opt.DoTimeoutMs = option.PushTimeoutMs
		opt.OverTcp = option.OverTcp
	})
	baseOutSession := NewBaseOutSession(uk, s)
	s.cmdSession = cmdSession
	s.baseOutSession = baseOutSession
	nazalog.Infof("[%s] lifecycle new rtsp PushSession. session=%p", uk, s)
	return s
}

// 阻塞直到和对端完成推流前，握手部分的工作（也即收到RTSP Record response），或者发生错误
func (session *PushSession) Push(rawUrl string, rawSdp []byte, sdpLogicCtx sdp.LogicContext) error {
	nazalog.Debugf("[%s] push. url=%s", session.uniqueKey, rawUrl)
	session.cmdSession.InitWithSdp(rawSdp, sdpLogicCtx)
	session.baseOutSession.InitWithSdp(rawSdp, sdpLogicCtx)
	return session.cmdSession.Do(rawUrl)
}

func (session *PushSession) WriteRtpPacket(packet rtprtcp.RtpPacket) {
	session.baseOutSession.WriteRtpPacket(packet)
}

// 文档请参考： interface IClientSessionLifecycle
func (session *PushSession) Dispose() error {
	nazalog.Infof("[%s] lifecycle dispose rtsp PushSession. session=%p", session.uniqueKey, session)
	e1 := session.cmdSession.Dispose()
	e2 := session.baseOutSession.Dispose()
	return nazaerrors.CombineErrors(e1, e2)
}

// 文档请参考： interface IClientSessionLifecycle
func (session *PushSession) WaitChan() <-chan error {
	return session.cmdSession.WaitChan()
}

// 文档请参考： interface ISessionUrlContext
func (session *PushSession) Url() string {
	return session.cmdSession.Url()
}

// 文档请参考： interface ISessionUrlContext
func (session *PushSession) AppName() string {
	return session.cmdSession.AppName()
}

// 文档请参考： interface ISessionUrlContext
func (session *PushSession) StreamName() string {
	return session.cmdSession.StreamName()
}

// 文档请参考： interface ISessionUrlContext
func (session *PushSession) RawQuery() string {
	return session.cmdSession.RawQuery()
}

// 文档请参考： interface IObject
func (session *PushSession) UniqueKey() string {
	return session.uniqueKey
}

// 文档请参考： interface ISessionStat
func (session *PushSession) GetStat() base.StatSession {
	stat := session.baseOutSession.GetStat()
	stat.RemoteAddr = session.cmdSession.RemoteAddr()
	return stat
}

// 文档请参考： interface ISessionStat
func (session *PushSession) UpdateStat(intervalSec uint32) {
	session.baseOutSession.UpdateStat(intervalSec)
}

// 文档请参考： interface ISessionStat
func (session *PushSession) IsAlive() (readAlive, writeAlive bool) {
	return session.baseOutSession.IsAlive()
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PushSession) OnConnectResult() {
	// noop
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PushSession) OnDescribeResponse(rawSdp []byte, sdpLogicCtx sdp.LogicContext) {
	// noop
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PushSession) OnSetupWithConn(uri string, rtpConn, rtcpConn *nazanet.UdpConnection) {
	_ = session.baseOutSession.SetupWithConn(uri, rtpConn, rtcpConn)
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PushSession) OnSetupWithChannel(uri string, rtpChannel, rtcpChannel int) {
	_ = session.baseOutSession.SetupWithChannel(uri, rtpChannel, rtcpChannel)
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PushSession) OnSetupResult() {
	// noop
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PushSession) OnInterleavedPacket(packet []byte, channel int) {
	session.baseOutSession.HandleInterleavedPacket(packet, channel)
}

// IInterleavedPacketWriter, callback by BaseOutSession
func (session *PushSession) WriteInterleavedPacket(packet []byte, channel int) error {
	return session.cmdSession.WriteInterleavedPacket(packet, channel)
}
