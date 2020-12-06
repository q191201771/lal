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
	baseInSession *BaseInSession
	UniqueKey     string
	StreamName    string // presentation

	observer PubSessionObserver
}

func NewPubSession(streamName string, cmdSession *ServerCommandSession) *PubSession {
	uk := base.GenUniqueKey(base.UKPRTSPPubSession)
	baseInSession := &BaseInSession{
		UniqueKey: uk,
		stat: base.StatPub{
			StatSession: base.StatSession{
				Protocol:  base.ProtocolRTSP,
				SessionID: uk,
				StartTime: time.Now().Format("2006-01-02 15:04:05.999"),
			},
		},
		cmdSession: cmdSession,
	}
	ps := &PubSession{
		baseInSession: baseInSession,
		UniqueKey:     uk,
		StreamName:    streamName,
	}
	nazalog.Infof("[%s] lifecycle new rtsp PubSession. session=%p, streamName=%s", uk, ps, streamName)
	return ps
}

func (p *PubSession) InitWithSDP(rawSDP []byte, sdpLogicCtx sdp.LogicContext) {
	p.baseInSession.InitWithSDP(rawSDP, sdpLogicCtx)
}

func (p *PubSession) SetObserver(observer PubSessionObserver) {
	p.baseInSession.SetObserver(observer)
}

func (p *PubSession) SetupWithConn(uri string, rtpConn, rtcpConn *nazanet.UDPConnection) error {
	return p.baseInSession.SetupWithConn(uri, rtpConn, rtcpConn)
}

func (p *PubSession) SetupWithChannel(uri string, rtpChannel, rtcpChannel int, remoteAddr string) error {
	return p.baseInSession.SetupWithChannel(uri, rtpChannel, rtcpChannel, remoteAddr)
}

func (p *PubSession) Dispose() {
	p.baseInSession.Dispose()
}

func (p *PubSession) GetSDP() ([]byte, sdp.LogicContext) {
	return p.baseInSession.GetSDP()
}

func (p *PubSession) HandleInterleavedPacket(b []byte, channel int) {
	p.baseInSession.HandleInterleavedPacket(b, channel)
}

func (p *PubSession) GetStat() base.StatPub {
	return p.baseInSession.GetStat()
}

func (p *PubSession) UpdateStat(interval uint32) {
	p.baseInSession.UpdateStat(interval)
}

func (p *PubSession) IsAlive(interval uint32) (ret bool) {
	return p.baseInSession.IsAlive(interval)
}
