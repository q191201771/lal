// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"net"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"

	"github.com/q191201771/naza/pkg/unique"

	"github.com/q191201771/naza/pkg/nazalog"
)

type PubSessionObserver interface {
	OnASC(asc []byte)
	OnSPSPPS(sps, pps []byte)
	OnAVPacket(pkt base.AVPacket)
}

type PubSession struct {
	UniqueKey  string
	StreamName string // presentation
	observer   PubSessionObserver

	servers       []*UDPServer
	audioComposer *rtprtcp.RTPComposer
	videoComposer *rtprtcp.RTPComposer

	sps []byte
	pps []byte
	asc []byte
}

func NewPubSession(streamName string) *PubSession {
	uk := unique.GenUniqueKey("RTSP")
	return &PubSession{
		UniqueKey:  uk,
		StreamName: streamName,
	}
}

func (p *PubSession) SetObserver(obs PubSessionObserver) {
	p.observer = obs

	if p.sps != nil && p.pps != nil {
		p.observer.OnSPSPPS(p.sps, p.pps)
	}
	if p.asc != nil {
		p.observer.OnASC(p.asc)
	}
}

func (p *PubSession) InitWithSDP(sdpCtx sdp.SDPContext) {
	var err error

	var audioPayloadType int
	var videoPayloadType int
	var audioClockRate int
	var videoClockRate int
	for _, item := range sdpCtx.AFmtPBaseList {
		switch item.Format {
		case base.RTPPacketTypeAVC:
			videoPayloadType = item.Format

			p.sps, p.pps, err = sdp.ParseSPSPPS(item)
			if err != nil {
				nazalog.Errorf("parse sps pps from sdp failed.")
			}
		case base.RTPPacketTypeAAC:
			audioPayloadType = item.Format

			p.asc, err = sdp.ParseASC(item)
			if err != nil {
				nazalog.Errorf("parse asc from sdp failed.")
			}
		default:
			nazalog.Errorf("unknown format. fmt=%d", item.Format)
		}
	}

	for _, item := range sdpCtx.ARTPMapList {
		switch item.PayloadType {
		case base.RTPPacketTypeAVC:
			videoClockRate = item.ClockRate
		case base.RTPPacketTypeAAC:
			audioClockRate = item.ClockRate
		default:
			nazalog.Errorf("unknown payloadType. type=%d", item.PayloadType)
		}
	}

	p.audioComposer = rtprtcp.NewRTPComposer(audioPayloadType, audioClockRate, composerItemMaxSize, p.onAVPacket)
	p.videoComposer = rtprtcp.NewRTPComposer(videoPayloadType, videoClockRate, composerItemMaxSize, p.onAVPacket)
}

func (p *PubSession) AddConn(conn *net.UDPConn) {
	server := NewUDPServerWithConn(conn, p.onReadUDPPacket)
	go server.RunLoop()
	p.servers = append(p.servers, server)
}
func (p *PubSession) onReadUDPPacket(b []byte, addr string, err error) {
	// try RTCP
	switch b[1] {
	case rtprtcp.RTCPPacketTypeSR:
		rtprtcp.ParseRTCPPacket(b)
		return
	}

	// try RTP
	packetType := b[1] & 0x7F
	if packetType == base.RTPPacketTypeAVC || packetType == base.RTPPacketTypeAAC {
		h, err := rtprtcp.ParseRTPPacket(b)
		if err != nil {
			nazalog.Errorf("read invalid rtp packet. err=%+v", err)
		}
		//nazalog.Debugf("%+v", h)
		var pkt rtprtcp.RTPPacket
		pkt.Header = h
		pkt.Raw = b

		if packetType == base.RTPPacketTypeAVC {
			p.videoComposer.Feed(pkt)
		} else {
			p.audioComposer.Feed(pkt)
		}
	} else {
		nazalog.Errorf("unknown PT. pt=%d", packetType)
		rtprtcp.ParseRTCPPacket(b)
	}

}

func (p *PubSession) onAVPacket(pkt base.AVPacket) {
	p.observer.OnAVPacket(pkt)
}
