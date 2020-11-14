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
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/q191201771/naza/pkg/connection"

	"github.com/q191201771/naza/pkg/nazanet"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"

	"github.com/q191201771/naza/pkg/nazalog"
)

// PubSession会同时向上层回调rtp packet，以及rtp合并后的av packet
type PubSessionObserver interface {
	OnRTPPacket(pkt rtprtcp.RTPPacket)

	// @param asc: AAC AudioSpecificConfig，注意，如果不存在音频，则为nil
	// @param vps: 视频为H264时为nil，视频为H265时不为nil
	OnAVConfig(asc, vps, sps, pps []byte)

	// @param pkt: pkt结构体中字段含义见rtprtcp.OnAVPacket
	OnAVPacket(pkt base.AVPacket)
}

type PubSession struct {
	UniqueKey     string
	StreamName    string // presentation
	observer      PubSessionObserver
	avPacketQueue *AVPacketQueue

	audioUnpacker    *rtprtcp.RTPUnpacker
	videoUnpacker    *rtprtcp.RTPUnpacker
	audioRRProducer  *rtprtcp.RRProducer
	videoRRProducer  *rtprtcp.RRProducer
	audioSsrc        uint32
	videoSsrc        uint32
	audioPayloadType base.AVPacketPT
	videoPayloadType base.AVPacketPT
	audioAControl    string
	videoAControl    string

	audioRTPConn  *nazanet.UDPConnection
	videoRTPConn  *nazanet.UDPConnection
	audioRTCPConn *nazanet.UDPConnection
	videoRTCPConn *nazanet.UDPConnection

	currConnStat connection.Stat
	prevConnStat connection.Stat
	staleStat    *connection.Stat
	stat         base.StatPub

	vps []byte // 如果是H265的话
	sps []byte
	pps []byte
	asc []byte

	m      sync.Mutex
	rawSDP []byte
}

func NewPubSession(streamName string) *PubSession {
	uk := base.GenUniqueKey(base.UKPRTSPPubSession)
	ps := &PubSession{
		UniqueKey:  uk,
		StreamName: streamName,
		stat: base.StatPub{
			StatSession: base.StatSession{
				Protocol:  base.ProtocolRTSP,
				SessionID: uk,
				StartTime: time.Now().Format("2006-01-02 15:04:05.999"),
			},
		},
	}
	nazalog.Infof("[%s] lifecycle new rtsp PubSession. session=%p, streamName=%s", uk, ps, streamName)
	return ps
}

func (p *PubSession) SetObserver(observer PubSessionObserver) {
	p.observer = observer

	p.observer.OnAVConfig(p.asc, p.vps, p.sps, p.pps)
}

func (p *PubSession) InitWithSDP(rawSDP []byte, sdpCtx sdp.SDPContext) {
	p.m.Lock()
	p.rawSDP = rawSDP
	p.m.Unlock()

	var err error

	var audioClockRate int
	var videoClockRate int

	for i, item := range sdpCtx.ARTPMapList {
		switch item.PayloadType {
		case base.RTPPacketTypeAVCOrHEVC:
			videoClockRate = item.ClockRate
			if item.EncodingName == "H265" {
				p.videoPayloadType = base.AVPacketPTHEVC
			} else {
				p.videoPayloadType = base.AVPacketPTAVC
			}
			if i < len(sdpCtx.AControlList) {
				p.videoAControl = sdpCtx.AControlList[i].Value
			}
		case base.RTPPacketTypeAAC:
			audioClockRate = item.ClockRate
			p.audioPayloadType = base.AVPacketPTAAC
			if i < len(sdpCtx.AControlList) {
				p.audioAControl = sdpCtx.AControlList[i].Value
			}
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

	if p.audioPayloadType != 0 && p.videoPayloadType != 0 {
		p.avPacketQueue = NewAVPacketQueue(p.onAVPacket)
	}
}

func (p *PubSession) Setup(uri string, rtpConn, rtcpConn *nazanet.UDPConnection) error {
	if strings.HasSuffix(uri, p.audioAControl) {
		p.audioRTPConn = rtpConn
		p.audioRTCPConn = rtcpConn
	} else if strings.HasSuffix(uri, p.videoAControl) {
		p.videoRTPConn = rtpConn
		p.videoRTCPConn = rtcpConn
	} else {
		return ErrRTSP
	}

	go rtpConn.RunLoop(p.onReadUDPPacket)
	go rtcpConn.RunLoop(p.onReadUDPPacket)

	return nil
}

// TODO chef: dispose后，回调上层
func (p *PubSession) Dispose() {
	nazalog.Infof("[%s] lifecycle dispose rtsp PubSession.", p.UniqueKey)

	if p.audioRTPConn != nil {
		_ = p.audioRTPConn.Dispose()
	}
	if p.audioRTCPConn != nil {
		_ = p.audioRTCPConn.Dispose()
	}
	if p.videoRTPConn != nil {
		_ = p.videoRTPConn.Dispose()
	}
	if p.videoRTCPConn != nil {
		_ = p.videoRTCPConn.Dispose()
	}
}

func (p *PubSession) GetStat() base.StatPub {
	p.stat.ReadBytesSum = atomic.LoadUint64(&p.currConnStat.ReadBytesSum)
	p.stat.WroteBytesSum = atomic.LoadUint64(&p.currConnStat.WroteBytesSum)
	return p.stat
}

func (p *PubSession) UpdateStat(interval uint32) {
	readBytesSum := atomic.LoadUint64(&p.currConnStat.ReadBytesSum)
	wroteBytesSum := atomic.LoadUint64(&p.currConnStat.WroteBytesSum)
	diff := readBytesSum - p.prevConnStat.ReadBytesSum
	p.stat.Bitrate = int(diff * 8 / 1024 / uint64(interval))
	p.prevConnStat.ReadBytesSum = readBytesSum
	p.prevConnStat.WroteBytesSum = wroteBytesSum
}

func (p *PubSession) IsAlive(interval uint32) (ret bool) {
	readBytesSum := atomic.LoadUint64(&p.currConnStat.ReadBytesSum)
	wroteBytesSum := atomic.LoadUint64(&p.currConnStat.WroteBytesSum)
	if p.staleStat == nil {
		p.staleStat = new(connection.Stat)
		p.staleStat.ReadBytesSum = readBytesSum
		p.staleStat.WroteBytesSum = wroteBytesSum
		return true
	}

	ret = !(readBytesSum-p.staleStat.ReadBytesSum == 0)
	p.staleStat.ReadBytesSum = readBytesSum
	p.staleStat.WroteBytesSum = wroteBytesSum
	return ret
}

func (p *PubSession) GetSDP() []byte {
	p.m.Lock()
	defer p.m.Unlock()
	return p.rawSDP
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
			if rrBuf != nil {
				_ = p.audioRTCPConn.Write2Addr(rrBuf, rAddr)
				atomic.AddUint64(&p.currConnStat.WroteBytesSum, uint64(len(b)))
			}
		case p.videoSsrc:
			rrBuf = p.videoRRProducer.Produce(sr.GetMiddleNTP())
			if rrBuf != nil {
				_ = p.videoRTCPConn.Write2Addr(rrBuf, rAddr)
				atomic.AddUint64(&p.currConnStat.WroteBytesSum, uint64(len(b)))
			}
		}

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
			p.observer.OnRTPPacket(pkt)
			p.videoUnpacker.Feed(pkt)
			p.videoRRProducer.FeedRTPPacket(h.Seq)
		} else {
			p.audioSsrc = h.Ssrc
			p.observer.OnRTPPacket(pkt)
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
	if p.avPacketQueue != nil {
		p.avPacketQueue.Feed(pkt)
	} else {
		p.observer.OnAVPacket(pkt)
	}
	//if p.audioUnpacker != nil && p.videoUnpacker != nil {
	//} else {
	//	p.observer.OnAVPacket(pkt)
	//}
}

// callback by avpacket queue
func (p *PubSession) onAVPacket(pkt base.AVPacket) {
	p.observer.OnAVPacket(pkt)
}
