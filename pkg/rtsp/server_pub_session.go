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
	"sync/atomic"
	"time"

	"github.com/q191201771/naza/pkg/connection"

	"github.com/q191201771/naza/pkg/nazanet"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"

	"github.com/q191201771/naza/pkg/unique"

	"github.com/q191201771/naza/pkg/nazalog"
)

type PubSessionObserver interface {
	OnASC(asc []byte)
	OnSPSPPS(sps, pps []byte)         // 如果是H264
	OnVPSSPSPPS(vps, sps, pps []byte) // 如果是H265

	// @param pkt: pkt结构体中字段含义见rtprtcp.OnAVPacket
	OnAVPacket(pkt base.AVPacket)
}

type PubSession struct {
	UniqueKey     string
	StreamName    string // presentation
	observer      PubSessionObserver
	avPacketQueue *AVPacketQueue

	rtpConn      *nazanet.UDPConnection
	rtcpConn     *nazanet.UDPConnection
	currConnStat connection.Stat
	prevConnStat connection.Stat
	stat         base.StatPub

	audioUnpacker    *rtprtcp.RTPUnpacker
	videoUnpacker    *rtprtcp.RTPUnpacker
	audioRRProducer  *rtprtcp.RRProducer
	videoRRProducer  *rtprtcp.RRProducer
	audioSsrc        uint32
	videoSsrc        uint32
	audioPayloadType base.AVPacketPT
	videoPayloadType base.AVPacketPT

	vps []byte // 如果是H265的话
	sps []byte
	pps []byte
	asc []byte
}

func NewPubSession(streamName string) *PubSession {
	uk := unique.GenUniqueKey("RTSPPUB")
	ps := &PubSession{
		UniqueKey:  uk,
		StreamName: streamName,
		stat: base.StatPub{
			StatSession: base.StatSession{
				Protocol:  base.ProtocolRTSP,
				StartTime: time.Now().Format("2006-01-02 15:04:05.999"),
			},
		},
	}
	ps.avPacketQueue = NewAVPacketQueue(ps.onAVPacket)
	nazalog.Infof("[%s] lifecycle new rtsp PubSession. session=%p, streamName=%s", uk, ps, streamName)
	return ps
}

func (p *PubSession) SetObserver(observer PubSessionObserver) {
	p.observer = observer

	if p.sps != nil && p.pps != nil {
		if p.vps != nil {
			p.observer.OnVPSSPSPPS(p.vps, p.sps, p.pps)
		} else {
			p.observer.OnSPSPPS(p.sps, p.pps)
		}
	}
	if p.asc != nil {
		p.observer.OnASC(p.asc)
	}
}

func (p *PubSession) InitWithSDP(sdpCtx sdp.SDPContext) {
	var err error

	var audioClockRate int
	var videoClockRate int

	for _, item := range sdpCtx.ARTPMapList {
		switch item.PayloadType {
		case base.RTPPacketTypeAVCOrHEVC:
			videoClockRate = item.ClockRate
			if item.EncodingName == "H265" {
				p.videoPayloadType = base.AVPacketPTHEVC
			} else {
				p.videoPayloadType = base.AVPacketPTAVC
			}
		case base.RTPPacketTypeAAC:
			audioClockRate = item.ClockRate
			p.audioPayloadType = base.AVPacketPTAAC
		default:
			nazalog.Errorf("unknown payloadType. type=%d", item.PayloadType)
		}
	}

	for _, item := range sdpCtx.AFmtPBaseList {
		switch item.Format {
		case base.RTPPacketTypeAVCOrHEVC:
			if p.videoPayloadType == base.AVPacketPTHEVC {
				p.vps, p.sps, p.pps, err = sdp.ParseVPSSPSPPS(item)
			} else {
				p.sps, p.pps, err = sdp.ParseSPSPPS(item)
			}
			if err != nil {
				nazalog.Errorf("parse sps pps from sdp failed.")
			}
		case base.RTPPacketTypeAAC:
			p.asc, err = sdp.ParseASC(item)
			if err != nil {
				nazalog.Errorf("parse asc from sdp failed.")
			}
		default:
			nazalog.Errorf("unknown format. fmt=%d", item.Format)
		}
	}

	p.audioUnpacker = rtprtcp.NewRTPUnpacker(p.audioPayloadType, audioClockRate, unpackerItemMaxSize, p.onAVPacketUnpacked)
	p.videoUnpacker = rtprtcp.NewRTPUnpacker(p.videoPayloadType, videoClockRate, unpackerItemMaxSize, p.onAVPacketUnpacked)

	p.audioRRProducer = rtprtcp.NewRRProducer(audioClockRate)
	p.videoRRProducer = rtprtcp.NewRRProducer(videoClockRate)
}

func (p *PubSession) SetRTPConn(conn *net.UDPConn) {
	server, _ := nazanet.NewUDPConnection(func(option *nazanet.UDPConnectionOption) {
		option.Conn = conn
	})
	p.rtpConn = server

	go server.RunLoop(p.onReadUDPPacket)
}

func (p *PubSession) SetRTCPConn(conn *net.UDPConn) {
	server, _ := nazanet.NewUDPConnection(func(option *nazanet.UDPConnectionOption) {
		option.Conn = conn
	})
	p.rtcpConn = server

	go server.RunLoop(p.onReadUDPPacket)
}

func (p *PubSession) Dispose() {
	if p.rtpConn != nil {
		_ = p.rtpConn.Dispose()
	}
	if p.rtcpConn != nil {
		_ = p.rtcpConn.Dispose()
	}
}

func (p *PubSession) GetStat() base.StatPub {
	p.stat.ReadBytesSum = atomic.LoadUint64(&p.currConnStat.ReadBytesSum)
	p.stat.WroteBytesSum = atomic.LoadUint64(&p.currConnStat.WroteBytesSum)
	return p.stat
}

func (p *PubSession) UpdateStat(tickCount uint32) {
	diff := p.currConnStat.ReadBytesSum - p.prevConnStat.ReadBytesSum
	p.stat.Bitrate = int(diff * 8 / 1024 / 5)
	p.prevConnStat = p.currConnStat
}

// callback by UDPConnection
// TODO yoko: 因为rtp和rtcp使用了两个连接，所以分成两个回调也行
func (p *PubSession) onReadUDPPacket(b []byte, rAddr *net.UDPAddr, err error) bool {
	if err != nil {
		nazalog.Errorf("read udp packet failed. err=%+v", err)
		return true
	}

	atomic.AddUint64(&p.currConnStat.ReadBytesSum, uint64(len(b)))

	if len(b) < 2 {
		nazalog.Errorf("read udp packet length invalid. len=%d", len(b))
		return true
	}

	// try RTCP
	switch b[1] {
	case rtprtcp.RTCPPacketTypeSR:
		sr := rtprtcp.ParseSR(b)
		var rrBuf []byte
		switch sr.SenderSSRC {
		case p.audioSsrc:
			rrBuf = p.audioRRProducer.Produce(sr.GetMiddleNTP())
		case p.videoSsrc:
			rrBuf = p.videoRRProducer.Produce(sr.GetMiddleNTP())
		}

		_ = p.rtcpConn.Write(rrBuf)

		atomic.AddUint64(&p.currConnStat.WroteBytesSum, uint64(len(b)))
		return true
	}

	// try RTP
	packetType := b[1] & 0x7F
	if packetType == base.RTPPacketTypeAVCOrHEVC || packetType == base.RTPPacketTypeAAC {
		h, err := rtprtcp.ParseRTPPacket(b)
		if err != nil {
			nazalog.Errorf("read invalid rtp packet. err=%+v", err)
		}
		//nazalog.Debugf("%+v", h)
		var pkt rtprtcp.RTPPacket
		pkt.Header = h
		pkt.Raw = b

		if packetType == base.RTPPacketTypeAVCOrHEVC {
			p.videoSsrc = h.Ssrc
			p.videoUnpacker.Feed(pkt)
			p.videoRRProducer.FeedRTPPacket(h.Seq)
		} else {
			p.audioSsrc = h.Ssrc
			p.audioUnpacker.Feed(pkt)
			p.audioRRProducer.FeedRTPPacket(h.Seq)
		}

		if p.stat.RemoteAddr == "" {
			p.stat.RemoteAddr = rAddr.String()
		}

		return true
	}

	nazalog.Errorf("unknown PT. pt=%d", b[1])
	return true
}

// callback by RTPUnpacker
func (p *PubSession) onAVPacketUnpacked(pkt base.AVPacket) {
	if p.audioUnpacker != nil && p.videoUnpacker != nil {
		p.avPacketQueue.Feed(pkt)
	} else {
		p.observer.OnAVPacket(pkt)
	}
}

// callback by avpacket queue
func (p *PubSession) onAVPacket(pkt base.AVPacket) {
	p.observer.OnAVPacket(pkt)
}
