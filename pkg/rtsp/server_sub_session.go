// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"encoding/hex"
	"net"
	"strings"
	"time"

	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
)

// TODO chef: 主动发送SR

type SubSession struct {
	UniqueKey  string // const after ctor
	StreamName string // const after ctor

	rawSDP      []byte           // const after set
	sdpLogicCtx sdp.LogicContext // const after set

	audioRTPConn  *nazanet.UDPConnection
	videoRTPConn  *nazanet.UDPConnection
	audioRTCPConn *nazanet.UDPConnection
	videoRTCPConn *nazanet.UDPConnection

	stat base.StatPub

	// TCP channels
	aRTPChannel        int
	aRTPControlChannel int
	vRTPChannel        int
	vRTPControlChannel int
}

func NewSubSession(streamName string) *SubSession {
	uk := base.GenUniqueKey(base.UKPRTSPSubSession)
	ss := &SubSession{
		UniqueKey:  uk,
		StreamName: streamName,
		stat: base.StatPub{
			StatSession: base.StatSession{
				Protocol:  base.ProtocolRTSP,
				StartTime: time.Now().Format("2006-01-02 15:04:05.999"),
			},
		},
	}
	nazalog.Infof("[%s] lifecycle new rtsp PubSession. session=%p, streamName=%s", uk, ss, streamName)
	return ss
}

func (s *SubSession) InitWithSDP(rawSDP []byte, sdpLogicCtx sdp.LogicContext) {
	s.rawSDP = rawSDP
	s.sdpLogicCtx = sdpLogicCtx
}

func (p *SubSession) GetSDP() ([]byte, sdp.LogicContext) {
	return p.rawSDP, p.sdpLogicCtx
}

func (p *SubSession) SetTCPVideoRTPChannel(RTPChannel int, RTPControlChannel int) {
	p.vRTPChannel = RTPChannel
	p.vRTPControlChannel = RTPControlChannel
}

func (p *SubSession) SetTCPAudioRTPChannel(RTPChannel int, RTPControlChannel int) {
	p.aRTPChannel = RTPChannel
	p.aRTPControlChannel = RTPControlChannel
}

func (s *SubSession) Setup(uri string, rtpConn, rtcpConn *nazanet.UDPConnection) error {
	if strings.HasSuffix(uri, s.sdpLogicCtx.AudioAControl) {
		s.audioRTPConn = rtpConn
		s.audioRTCPConn = rtcpConn
	} else if strings.HasSuffix(uri, s.sdpLogicCtx.VideoAControl) {
		s.videoRTPConn = rtpConn
		s.videoRTCPConn = rtcpConn
	} else {
		return ErrRTSP
	}

	go rtpConn.RunLoop(s.onReadUDPPacket)
	go rtcpConn.RunLoop(s.onReadUDPPacket)

	return nil
}

func (s *SubSession) onReadUDPPacket(b []byte, rAddr *net.UDPAddr, err error) bool {
	nazalog.Debugf("SubSession::onReadUDPPacket. %s", hex.Dump(b))
	return true
}

//to be continued
//conn可能还不存在，这里涉及到pub和sub是否需要等到setup再回调给上层的问题
func (s *SubSession) WriteRTPPacket(packet rtprtcp.RTPPacket) {
	//switch packet.Header.PacketType {
	//case base.RTPPacketTypeAVCOrHEVC:
	//	s.videoRTPConn.Write(packet.Raw)
	//case base.RTPPacketTypeAAC:
	//	s.audioRTPConn.Write(packet.Raw)
	//}
}
