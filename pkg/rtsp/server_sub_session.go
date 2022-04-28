// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/nazaerrors"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazanet"
)

type SubSession struct {
	uniqueKey      string // const after ctor
	urlCtx         base.UrlContext
	cmdSession     *ServerCommandSession
	baseOutSession *BaseOutSession

	ShouldWaitVideoKeyFrame bool
}

func NewSubSession(urlCtx base.UrlContext, cmdSession *ServerCommandSession) *SubSession {
	uk := base.GenUkRtspSubSession()
	s := &SubSession{
		uniqueKey:  uk,
		urlCtx:     urlCtx,
		cmdSession: cmdSession,

		ShouldWaitVideoKeyFrame: true,
	}
	baseOutSession := NewBaseOutSession(uk, s)
	s.baseOutSession = baseOutSession
	Log.Infof("[%s] lifecycle new rtsp SubSession. session=%p, streamName=%s", uk, s, urlCtx.LastItemOfPath)
	return s
}

func (session *SubSession) InitWithSdp(sdpCtx sdp.LogicContext) {
	session.baseOutSession.InitWithSdp(sdpCtx)
}

func (session *SubSession) SetupWithConn(uri string, rtpConn, rtcpConn *nazanet.UdpConnection) error {
	return session.baseOutSession.SetupWithConn(uri, rtpConn, rtcpConn)
}

func (session *SubSession) SetupWithChannel(uri string, rtpChannel, rtcpChannel int) error {
	return session.baseOutSession.SetupWithChannel(uri, rtpChannel, rtcpChannel)
}

func (session *SubSession) WriteRtpPacket(packet rtprtcp.RtpPacket) {
	session.baseOutSession.WriteRtpPacket(packet)
}

func (session *SubSession) Dispose() error {
	Log.Infof("[%s] lifecycle dispose rtsp SubSession. session=%p", session.uniqueKey, session)
	e1 := session.baseOutSession.Dispose()
	e2 := session.cmdSession.Dispose()
	return nazaerrors.CombineErrors(e1, e2)
}

func (session *SubSession) HandleInterleavedPacket(b []byte, channel int) {
	session.baseOutSession.HandleInterleavedPacket(b, channel)
}

func (session *SubSession) Url() string {
	return session.urlCtx.Url
}

func (session *SubSession) AppName() string {
	return session.urlCtx.PathWithoutLastItem
}

func (session *SubSession) StreamName() string {
	return session.urlCtx.LastItemOfPath
}

func (session *SubSession) RawQuery() string {
	return session.urlCtx.RawQuery
}

func (session *SubSession) UniqueKey() string {
	return session.uniqueKey
}

func (session *SubSession) GetStat() base.StatSession {
	stat := session.baseOutSession.GetStat()
	stat.RemoteAddr = session.cmdSession.RemoteAddr()
	return stat
}

func (session *SubSession) UpdateStat(intervalSec uint32) {
	session.baseOutSession.UpdateStat(intervalSec)
}

func (session *SubSession) IsAlive() (readAlive, writeAlive bool) {
	return session.baseOutSession.IsAlive()
}

// WriteInterleavedPacket IInterleavedPacketWriter, callback by BaseOutSession
func (session *SubSession) WriteInterleavedPacket(packet []byte, channel int) error {
	return session.cmdSession.WriteInterleavedPacket(packet, channel)
}
