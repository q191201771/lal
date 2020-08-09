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

	"github.com/q191201771/naza/pkg/nazanet"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"

	"github.com/q191201771/naza/pkg/unique"

	"github.com/q191201771/naza/pkg/nazalog"
)

type PubSessionObserver interface {
	OnASC(asc []byte)
	OnSPSPPS(sps, pps []byte)

	// @param pkt: Timestamp返回的是pts经过clockrate换算后的时间戳，单位毫秒
	//             注意，不支持带B帧的视频流，pts和dts永远相同
	OnAVPacket(pkt base.AVPacket)
}

type PubSession struct {
	UniqueKey     string
	StreamName    string // presentation
	observer      PubSessionObserver
	avPacketQueue *AVPacketQueue

	rtpConn         *nazanet.UDPConnection
	rtcpConn        *nazanet.UDPConnection
	audioComposer   *rtprtcp.RTPComposer
	videoComposer   *rtprtcp.RTPComposer
	audioRRProducer *rtprtcp.RRProducer
	videoRRProducer *rtprtcp.RRProducer
	audioSsrc       uint32
	videoSsrc       uint32

	sps []byte
	pps []byte
	asc []byte
}

func NewPubSession(streamName string) *PubSession {
	uk := unique.GenUniqueKey("RTSP")
	ps := &PubSession{
		UniqueKey:  uk,
		StreamName: streamName,
	}
	ps.avPacketQueue = NewAVPacketQueue(ps.onAVPacket)
	return ps
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

	p.audioComposer = rtprtcp.NewRTPComposer(audioPayloadType, audioClockRate, composerItemMaxSize, p.onAVPacketComposed)
	p.videoComposer = rtprtcp.NewRTPComposer(videoPayloadType, videoClockRate, composerItemMaxSize, p.onAVPacketComposed)

	p.audioRRProducer = rtprtcp.NewRRProducer(audioClockRate)
	p.videoRRProducer = rtprtcp.NewRRProducer(videoClockRate)
}

func (p *PubSession) SetRTPConn(conn *net.UDPConn) {
	server := nazanet.NewUDPConnectionWithConn(conn, p.onReadUDPPacket)
	go server.RunLoop()
	p.rtpConn = server
}

func (p *PubSession) SetRTCPConn(conn *net.UDPConn) {
	server := nazanet.NewUDPConnectionWithConn(conn, p.onReadUDPPacket)
	go server.RunLoop()
	p.rtcpConn = server
}

func (p *PubSession) Dispose() {
	if p.rtpConn != nil {
		_ = p.rtpConn.Dispose()
	}
	if p.rtcpConn != nil {
		_ = p.rtcpConn.Dispose()
	}
}

// callback by UDPConnection
// TODO chef: 因为rtp和rtcp使用了两个连接，所以分成两个回调也行
func (p *PubSession) onReadUDPPacket(b []byte, remoteAddr net.Addr, err error) {
	if err != nil {
		nazalog.Errorf("read udp packet failed. err=%+v", err)
		return
	}

	if len(b) < 2 {
		nazalog.Errorf("read udp packet length invalid. len=%d", len(b))
		return
	}

	// try RTCP
	switch b[1] {
	case rtprtcp.RTCPPacketTypeSR:
		//h := rtprtcp.ParseRTCPHeader(b)
		sr := rtprtcp.ParseSR(b)
		var rrBuf []byte
		switch sr.SenderSSRC {
		case p.audioSsrc:
			rrBuf = p.audioRRProducer.Produce(sr.GetMiddleNTP())
		case p.videoSsrc:
			rrBuf = p.videoRRProducer.Produce(sr.GetMiddleNTP())
		}
		_ = p.rtcpConn.Write(rrBuf)
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
			p.videoSsrc = h.Ssrc
			p.videoComposer.Feed(pkt)
			p.videoRRProducer.FeedRTPPacket(h.Seq)
		} else {
			p.audioSsrc = h.Ssrc
			p.audioComposer.Feed(pkt)
			p.audioRRProducer.FeedRTPPacket(h.Seq)
		}

		return
	}

	nazalog.Errorf("unknown PT. pt=%d", b[1])
}

// callback by composer
func (p *PubSession) onAVPacketComposed(pkt base.AVPacket) {
	p.avPacketQueue.Insert(pkt)
}

// callback by avpacket queue
func (p *PubSession) onAVPacket(pkt base.AVPacket) {
	p.observer.OnAVPacket(pkt)
}
