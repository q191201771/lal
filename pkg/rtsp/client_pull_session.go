// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"github.com/cfeeling/lal/pkg/base"
	"github.com/cfeeling/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/nazaerrors"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
)

type PullSessionObserver interface {
	BaseInSessionObserver
}

type PullSessionOption struct {
	// 从调用Pull函数，到接收音视频数据的前一步，也即收到rtsp play response的超时时间
	// 如果为0，则没有超时时间
	PullTimeoutMS int

	OverTCP bool // 是否使用interleaved模式，也即是否通过rtsp command tcp连接传输rtp/rtcp数据
}

var defaultPullSessionOption = PullSessionOption{
	PullTimeoutMS: 10000,
	OverTCP:       false,
}

type PullSession struct {
	UniqueKey     string // const after ctor
	cmdSession    *ClientCommandSession
	baseInSession *BaseInSession
}

type ModPullSessionOption func(option *PullSessionOption)

func NewPullSession(observer PullSessionObserver, modOptions ...ModPullSessionOption) *PullSession {
	option := defaultPullSessionOption
	for _, fn := range modOptions {
		fn(&option)
	}

	uk := base.GenUniqueKey(base.UKPRTSPPullSession)
	s := &PullSession{
		UniqueKey: uk,
	}
	cmdSession := NewClientCommandSession(CCSTPullSession, uk, s, func(opt *ClientCommandSessionOption) {
		opt.DoTimeoutMS = option.PullTimeoutMS
		opt.OverTCP = option.OverTCP
	})
	baseInSession := NewBaseInSessionWithObserver(uk, s, observer)
	s.baseInSession = baseInSession
	s.cmdSession = cmdSession
	nazalog.Infof("[%s] lifecycle new rtsp PullSession. session=%p", uk, s)
	return s
}

// 如果没有错误发生，阻塞直到接收音视频数据的前一步，也即收到rtsp play response
func (session *PullSession) Pull(rawURL string) error {
	nazalog.Debugf("[%s] pull. url=%s", session.UniqueKey, rawURL)
	return session.cmdSession.Do(rawURL)
}

// Pull成功后，调用该函数，可阻塞直到拉流结束
func (session *PullSession) Wait() <-chan error {
	return session.cmdSession.Wait()
}

func (session *PullSession) Dispose() error {
	nazalog.Infof("[%s] lifecycle dispose rtsp PullSession. session=%p", session.UniqueKey, session)
	e1 := session.cmdSession.Dispose()
	e2 := session.baseInSession.Dispose()
	return nazaerrors.CombineErrors(e1, e2)
}

func (session *PullSession) GetSDP() ([]byte, sdp.LogicContext) {
	return session.baseInSession.GetSDP()
}

func (session *PullSession) AppName() string {
	return session.cmdSession.AppName()
}

func (session *PullSession) StreamName() string {
	return session.cmdSession.StreamName()
}

func (session *PullSession) RawQuery() string {
	return session.cmdSession.RawQuery()
}

func (session *PullSession) GetStat() base.StatSession {
	stat := session.baseInSession.GetStat()
	stat.RemoteAddr = session.cmdSession.RemoteAddr()
	return stat
}

func (session *PullSession) UpdateStat(interval uint32) {
	session.baseInSession.UpdateStat(interval)
}

func (session *PullSession) IsAlive() (readAlive, writeAlive bool) {
	return session.baseInSession.IsAlive()
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PullSession) OnConnectResult() {
	// noop
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PullSession) OnDescribeResponse(rawSDP []byte, sdpLogicCtx sdp.LogicContext) {
	session.baseInSession.InitWithSDP(rawSDP, sdpLogicCtx)
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PullSession) OnSetupWithConn(uri string, rtpConn, rtcpConn *nazanet.UDPConnection) {
	_ = session.baseInSession.SetupWithConn(uri, rtpConn, rtcpConn)
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PullSession) OnSetupWithChannel(uri string, rtpChannel, rtcpChannel int) {
	_ = session.baseInSession.SetupWithChannel(uri, rtpChannel, rtcpChannel)
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PullSession) OnSetupResult() {
	session.baseInSession.WriteRTPRTCPDummy()
}

// ClientCommandSessionObserver, callback by ClientCommandSession
func (session *PullSession) OnInterleavedPacket(packet []byte, channel int) {
	session.baseInSession.HandleInterleavedPacket(packet, channel)
}

// IInterleavedPacketWriter, callback by BaseInSession
func (session *PullSession) WriteInterleavedPacket(packet []byte, channel int) error {
	return session.cmdSession.WriteInterleavedPacket(packet, channel)
}
