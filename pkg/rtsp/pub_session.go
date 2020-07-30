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

	"github.com/q191201771/naza/pkg/nazalog"
)

type PubSession struct {
	StreamName   string // presentation
	onAVPacketFn OnAVPacket

	servers     []*UDPServer
	audioStream *Stream
	videoStream *Stream
}

func NewPubSession(streamName string) *PubSession {
	return &PubSession{
		StreamName: streamName,
	}
}

func (p *PubSession) SetOnAVPacket(onAVPacket OnAVPacket) {
	p.onAVPacketFn = onAVPacket
}

func (p *PubSession) InitWithSDP(sdp SDP) {
	var audioPayloadType int
	var videoPayloadType int
	var audioClockRate int
	var videoClockRate int
	for _, item := range sdp.AFmtPBaseList {
		switch item.Format {
		case RTPPacketTypeAVC:
			videoPayloadType = item.Format
		case RTPPacketTypeAAC:
			audioPayloadType = item.Format
		default:
			nazalog.Errorf("unknown format. fmt=%d", item.Format)
		}
	}

	for _, item := range sdp.ARTPMapList {
		switch item.PayloadType {
		case RTPPacketTypeAVC:
			videoClockRate = item.ClockRate
		case RTPPacketTypeAAC:
			audioClockRate = item.ClockRate
		default:
			nazalog.Errorf("unknown payloadType. type=%d", item.PayloadType)
		}
	}
	p.audioStream = NewStream(audioPayloadType, audioClockRate, p.onAVPacket)
	p.videoStream = NewStream(videoPayloadType, videoClockRate, p.onAVPacket)
}

func (p *PubSession) AddConn(conn *net.UDPConn) {
	server := NewUDPServerWithConn(conn, p.onReadUDPPacket)
	go server.RunLoop()
	p.servers = append(p.servers, server)
}
func (p *PubSession) onReadUDPPacket(b []byte, addr string, err error) {
	if len(b) <= 0 {
		return
	}

	// try RTCP
	switch b[1] {
	case RTCPPacketTypeSR:
		parseRTCPPacket(b)
		return
	}

	// try RTP
	packetType := b[1] & 0x7F
	switch packetType {
	case RTPPacketTypeAVC:
		h, err := parseRTPPacket(b)
		if err != nil {
			nazalog.Errorf("read invalid rtp packet. err=%+v", err)
		}
		nazalog.Debugf("%+v", h)
		var pkt RTPPacket
		pkt.header = h
		pkt.raw = b
		p.videoStream.FeedAVCPacket(pkt)
	case RTPPacketTypeAAC:
		h, err := parseRTPPacket(b)
		if err != nil {
			nazalog.Errorf("read invalid rtp packet. err=%+v", err)
		}
		nazalog.Debugf("%+v", h)
		var pkt RTPPacket
		pkt.header = h
		pkt.raw = b
		p.audioStream.FeedAACPacket(pkt)
	default:
		nazalog.Errorf("unknown PT. pt=%d", packetType)
		parseRTCPPacket(b)
	}
}

func (p *PubSession) onAVPacket(pkt AVPacket) {
	p.onAVPacketFn(pkt)
}
// 资源释放
func (p *PubSession) Dispose() {
	// p.conn会在Server.handleTCPConnect中关闭,所以在这里不关闭
	// 关闭关联的udp
	for _, udp := range p.servers {
		//客户端断开的时间第一时间释放本机占用的socket_point,避免在海量连接时资源不足
		udp.conn.Close()
	}
}