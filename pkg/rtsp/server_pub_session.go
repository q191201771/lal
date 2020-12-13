// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"time"

	"github.com/q191201771/naza/pkg/nazanet"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/sdp"

	"github.com/q191201771/naza/pkg/nazalog"
)

type PubSessionObserver interface {
	BaseInSessionObserver
}

type PubSession struct {
	UniqueKey     string
	baseInSession *BaseInSession
	urlCtx        base.URLContext

	observer PubSessionObserver
}

func NewPubSession(urlCtx base.URLContext, cmdSession *ServerCommandSession) *PubSession {
	uk := base.GenUniqueKey(base.UKPRTSPPubSession)
	baseInSession := &BaseInSession{
		UniqueKey: uk,
		stat: base.StatSession{
			Protocol:   base.ProtocolRTSP,
			SessionID:  uk,
			StartTime:  time.Now().Format("2006-01-02 15:04:05.999"),
			RemoteAddr: cmdSession.conn.RemoteAddr().String(),
		},
		cmdSession: cmdSession,
	}
	ps := &PubSession{
		baseInSession: baseInSession,
		UniqueKey:     uk,
		urlCtx:        urlCtx,
	}
	nazalog.Infof("[%s] lifecycle new rtsp PubSession. session=%p, streamName=%s", uk, ps, urlCtx.LastItemOfPath)
	return ps
}

func (s *PubSession) InitWithSDP(rawSDP []byte, sdpLogicCtx sdp.LogicContext) {
	s.baseInSession.InitWithSDP(rawSDP, sdpLogicCtx)
}

func (s *PubSession) SetObserver(observer PubSessionObserver) {
	s.baseInSession.SetObserver(observer)
}

func (s *PubSession) SetupWithConn(uri string, rtpConn, rtcpConn *nazanet.UDPConnection) error {
	return s.baseInSession.SetupWithConn(uri, rtpConn, rtcpConn)
}

func (s *PubSession) SetupWithChannel(uri string, rtpChannel, rtcpChannel int, remoteAddr string) error {
	return s.baseInSession.SetupWithChannel(uri, rtpChannel, rtcpChannel, remoteAddr)
}

func (s *PubSession) Dispose() {
	s.baseInSession.Dispose()
}

func (s *PubSession) GetSDP() ([]byte, sdp.LogicContext) {
	return s.baseInSession.GetSDP()
}

func (s *PubSession) HandleInterleavedPacket(b []byte, channel int) {
	s.baseInSession.HandleInterleavedPacket(b, channel)
}

func (s *PubSession) AppName() string {
	return s.urlCtx.PathWithoutLastItem
}

func (s *PubSession) StreamName() string {
	return s.urlCtx.LastItemOfPath
}

func (s *PubSession) RawQuery() string {
	return s.urlCtx.RawQuery
}

// @return 注意，`RemoteAddr`字段返回的是RTSP command TCP连接的地址
func (s *PubSession) GetStat() base.StatSession {
	return s.baseInSession.GetStat()
}

func (s *PubSession) UpdateStat(interval uint32) {
	s.baseInSession.UpdateStat(interval)
}

func (s *PubSession) IsAlive() (readAlive, writeAlive bool) {
	return s.baseInSession.IsAlive()
}
