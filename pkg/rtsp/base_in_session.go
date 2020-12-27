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

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
)

type BaseCommandSession interface {
	Write(channel int, b []byte) error
	Dispose() error
}

// 聚合PubSession和PullSession，也即流数据是输入类型的session

// BaseInSession会向上层回调两种格式的数据：
// 1. 原始的rtp packet
// 2. rtp合并后的av packet
type BaseInSessionObserver interface {
	OnRTPPacket(pkt rtprtcp.RTPPacket)

	// @param asc: AAC AudioSpecificConfig，注意，如果不存在音频，则为nil
	// @param vps: 视频为H264时为nil，视频为H265时不为nil
	OnAVConfig(asc, vps, sps, pps []byte)

	// @param pkt: pkt结构体中字段含义见rtprtcp.OnAVPacket
	OnAVPacket(pkt base.AVPacket)
}

type BaseInSession struct {
	UniqueKey  string // 使用上层Session的值
	cmdSession BaseCommandSession

	observer BaseInSessionObserver

	audioRTPConn     *nazanet.UDPConnection
	videoRTPConn     *nazanet.UDPConnection
	audioRTCPConn    *nazanet.UDPConnection
	videoRTCPConn    *nazanet.UDPConnection
	audioRTPChannel  int
	audioRTCPChannel int
	videoRTPChannel  int
	videoRTCPChannel int

	currConnStat connection.Stat
	prevConnStat connection.Stat
	staleStat    *connection.Stat
	stat         base.StatSession

	m           sync.Mutex
	rawSDP      []byte           // const after set
	sdpLogicCtx sdp.LogicContext // const after set

	avPacketQueue   *AVPacketQueue
	audioUnpacker   *rtprtcp.RTPUnpacker
	videoUnpacker   *rtprtcp.RTPUnpacker
	audioRRProducer *rtprtcp.RRProducer
	videoRRProducer *rtprtcp.RRProducer
	audioSSRC       uint32
	videoSSRC       uint32
}

func (s *BaseInSession) InitWithSDP(rawSDP []byte, sdpLogicCtx sdp.LogicContext) {
	s.m.Lock()
	s.rawSDP = rawSDP
	s.sdpLogicCtx = sdpLogicCtx
	s.m.Unlock()

	if isSupportType(s.sdpLogicCtx.AudioPayloadType) {
		s.audioUnpacker = rtprtcp.NewRTPUnpacker(s.sdpLogicCtx.AudioPayloadType, s.sdpLogicCtx.AudioClockRate, unpackerItemMaxSize, s.onAVPacketUnpacked)
	} else {
		nazalog.Warnf("[%s] audio unpacker not support yet. origin type=%d", s.UniqueKey, s.sdpLogicCtx.AudioPayloadTypeOrigin)
	}
	if isSupportType(s.sdpLogicCtx.VideoPayloadType) {
		s.videoUnpacker = rtprtcp.NewRTPUnpacker(s.sdpLogicCtx.VideoPayloadType, s.sdpLogicCtx.VideoClockRate, unpackerItemMaxSize, s.onAVPacketUnpacked)
	} else {
		nazalog.Warnf("[%s] video unpacker not support yet. origin type=%d", s.UniqueKey, s.sdpLogicCtx.AudioPayloadTypeOrigin)
	}

	s.audioRRProducer = rtprtcp.NewRRProducer(s.sdpLogicCtx.AudioClockRate)
	s.videoRRProducer = rtprtcp.NewRRProducer(s.sdpLogicCtx.VideoClockRate)

	if s.sdpLogicCtx.HasAudio && s.sdpLogicCtx.HasVideo {
		s.avPacketQueue = NewAVPacketQueue(s.onAVPacket)
	}

	if s.observer != nil {
		s.observer.OnAVConfig(s.sdpLogicCtx.ASC, s.sdpLogicCtx.VPS, s.sdpLogicCtx.SPS, s.sdpLogicCtx.PPS)
	}
}

// 如果没有设置回调监听对象，可以通过该函数设置，调用方保证调用该函数发生在调用InitWithSDP之后
func (s *BaseInSession) SetObserver(observer BaseInSessionObserver) {
	s.observer = observer

	// TODO chef: 这里的判断应该去掉
	if s.sdpLogicCtx.ASC != nil && s.sdpLogicCtx.SPS != nil {
		s.observer.OnAVConfig(s.sdpLogicCtx.ASC, s.sdpLogicCtx.VPS, s.sdpLogicCtx.SPS, s.sdpLogicCtx.PPS)
	}
}

func (s *BaseInSession) SetupWithConn(uri string, rtpConn, rtcpConn *nazanet.UDPConnection) error {
	if strings.HasSuffix(uri, s.sdpLogicCtx.AudioAControl) {
		s.audioRTPConn = rtpConn
		s.audioRTCPConn = rtcpConn
	} else if strings.HasSuffix(uri, s.sdpLogicCtx.VideoAControl) {
		s.videoRTPConn = rtpConn
		s.videoRTCPConn = rtcpConn
	} else {
		return ErrRTSP
	}

	go rtpConn.RunLoop(s.onReadRTPPacket)
	go rtcpConn.RunLoop(s.onReadRTCPPacket)

	return nil
}

func (s *BaseInSession) SetupWithChannel(uri string, rtpChannel, rtcpChannel int) error {
	if strings.HasSuffix(uri, s.sdpLogicCtx.AudioAControl) {
		s.audioRTPChannel = rtpChannel
		s.audioRTCPChannel = rtcpChannel
		return nil
	}
	if strings.HasSuffix(uri, s.sdpLogicCtx.VideoAControl) {
		s.videoRTPChannel = rtpChannel
		s.videoRTCPChannel = rtcpChannel
		return nil
	}
	return ErrRTSP
}

func (s *BaseInSession) Dispose() {
	nazalog.Infof("[%s] lifecycle dispose rtsp BaseInSession.", s.UniqueKey)

	if s.audioRTPConn != nil {
		_ = s.audioRTPConn.Dispose()
	}
	if s.audioRTCPConn != nil {
		_ = s.audioRTCPConn.Dispose()
	}
	if s.videoRTPConn != nil {
		_ = s.videoRTPConn.Dispose()
	}
	if s.videoRTCPConn != nil {
		_ = s.videoRTCPConn.Dispose()
	}

	_ = s.cmdSession.Dispose()
}

func (s *BaseInSession) GetSDP() ([]byte, sdp.LogicContext) {
	s.m.Lock()
	defer s.m.Unlock()
	return s.rawSDP, s.sdpLogicCtx
}

func (s *BaseInSession) HandleInterleavedPacket(b []byte, channel int) {
	switch channel {
	case s.audioRTPChannel:
		fallthrough
	case s.videoRTPChannel:
		_ = s.handleRTPPacket(b)
	case s.audioRTCPChannel:
		fallthrough
	case s.videoRTCPChannel:
		// TODO chef: 这个地方有bug，处理RTCP包则推流会失败，有可能是我的RTCP RR包打的有问题
		//_ = p.handleRTCPPacket(b, nil)
	default:
		nazalog.Errorf("[%s] read interleaved packet but channel invalid. channel=%d", s.UniqueKey, channel)
	}

}

func (s *BaseInSession) GetStat() base.StatSession {
	s.stat.ReadBytesSum = atomic.LoadUint64(&s.currConnStat.ReadBytesSum)
	s.stat.WroteBytesSum = atomic.LoadUint64(&s.currConnStat.WroteBytesSum)
	return s.stat
}

func (s *BaseInSession) UpdateStat(interval uint32) {
	readBytesSum := atomic.LoadUint64(&s.currConnStat.ReadBytesSum)
	wroteBytesSum := atomic.LoadUint64(&s.currConnStat.WroteBytesSum)
	rDiff := readBytesSum - s.prevConnStat.ReadBytesSum
	s.stat.ReadBitrate = int(rDiff * 8 / 1024 / uint64(interval))
	wDiff := wroteBytesSum - s.prevConnStat.WroteBytesSum
	s.stat.WriteBitrate = int(wDiff * 8 / 1024 / uint64(interval))
	s.stat.Bitrate = s.stat.ReadBitrate
	s.prevConnStat.ReadBytesSum = readBytesSum
	s.prevConnStat.WroteBytesSum = wroteBytesSum
}

func (s *BaseInSession) IsAlive() (readAlive, writeAlive bool) {
	readBytesSum := atomic.LoadUint64(&s.currConnStat.ReadBytesSum)
	wroteBytesSum := atomic.LoadUint64(&s.currConnStat.WroteBytesSum)
	if s.staleStat == nil {
		s.staleStat = new(connection.Stat)
		s.staleStat.ReadBytesSum = readBytesSum
		s.staleStat.WroteBytesSum = wroteBytesSum
		return true, true
	}

	readAlive = !(readBytesSum-s.staleStat.ReadBytesSum == 0)
	writeAlive = !(wroteBytesSum-s.staleStat.WroteBytesSum == 0)
	s.staleStat.ReadBytesSum = readBytesSum
	s.staleStat.WroteBytesSum = wroteBytesSum
	return
}

// 发现pull时，需要先给对端发送数据，才能收到数据
func (s *BaseInSession) WriteRTPRTCPDummy() {
	if s.videoRTPConn != nil {
		_ = s.videoRTPConn.Write(dummyRTPPacket)
	}
	if s.videoRTCPConn != nil {
		_ = s.videoRTCPConn.Write(dummyRTCPPacket)
	}
	if s.audioRTPConn != nil {
		_ = s.audioRTPConn.Write(dummyRTPPacket)
	}
	if s.audioRTCPConn != nil {
		_ = s.audioRTCPConn.Write(dummyRTCPPacket)
	}
}

// callback by RTPUnpacker
func (s *BaseInSession) onAVPacketUnpacked(pkt base.AVPacket) {
	if s.avPacketQueue != nil {
		s.avPacketQueue.Feed(pkt)
	} else {
		s.observer.OnAVPacket(pkt)
	}
}

// callback by avpacket queue
func (s *BaseInSession) onAVPacket(pkt base.AVPacket) {
	s.observer.OnAVPacket(pkt)
}

// callback by UDPConnection
func (s *BaseInSession) onReadRTPPacket(b []byte, rAddr *net.UDPAddr, err error) bool {
	if err != nil {
		nazalog.Errorf("[%s] read udp packet failed. err=%+v", s.UniqueKey, err)
		return true
	}

	_ = s.handleRTPPacket(b)
	return true
}

// callback by UDPConnection
func (s *BaseInSession) onReadRTCPPacket(b []byte, rAddr *net.UDPAddr, err error) bool {
	if err != nil {
		nazalog.Errorf("[%s] read udp packet failed. err=%+v", s.UniqueKey, err)
		return true
	}

	_ = s.handleRTCPPacket(b, rAddr)
	return true
}

// @param rAddr 对端地址，往对端发送数据时使用，注意，如果nil，则表示是interleaved模式，我们直接往TCP连接发数据
func (s *BaseInSession) handleRTCPPacket(b []byte, rAddr *net.UDPAddr) error {
	atomic.AddUint64(&s.currConnStat.ReadBytesSum, uint64(len(b)))

	if len(b) <= 0 {
		nazalog.Errorf("[%s] handleRTCPPacket length invalid. len=%d", s.UniqueKey, len(b))
		return ErrRTSP
	}

	packetType := b[1]
	if packetType != rtprtcp.RTCPPacketTypeSR {
		return ErrRTSP
	}

	switch b[1] {
	case rtprtcp.RTCPPacketTypeSR:
		sr := rtprtcp.ParseSR(b)
		//nazalog.Debugf("%+v", sr)
		var rrBuf []byte
		switch sr.SenderSSRC {
		case s.audioSSRC:
			rrBuf = s.audioRRProducer.Produce(sr.GetMiddleNTP())
			if rrBuf != nil {
				if rAddr != nil {
					_ = s.audioRTCPConn.Write2Addr(rrBuf, rAddr)
				} else {
					_ = s.cmdSession.Write(s.audioRTCPChannel, rrBuf)
				}
				atomic.AddUint64(&s.currConnStat.WroteBytesSum, uint64(len(b)))
			}
		case s.videoSSRC:
			rrBuf = s.videoRRProducer.Produce(sr.GetMiddleNTP())
			if rrBuf != nil {
				if rAddr != nil {
					_ = s.videoRTCPConn.Write2Addr(rrBuf, rAddr)
				} else {
					_ = s.cmdSession.Write(s.videoRTCPChannel, rrBuf)
				}
				atomic.AddUint64(&s.currConnStat.WroteBytesSum, uint64(len(b)))
			}
		default:
			// ffmpeg推流时，会在发送第一个RTP包之前就发送一个SR，所以关闭这个警告日志
			//nazalog.Warnf("[%s] read rtcp sr but senderSSRC invalid. senderSSRC=%d, audio=%d, video=%d",
			//	p.UniqueKey, sr.SenderSSRC, p.audioSSRC, p.videoSSRC)
			return ErrRTSP
		}
	default:
		nazalog.Errorf("[%s] read rtcp packet but type invalid. type=%d", s.UniqueKey, b[1])
		return ErrRTSP
	}

	return nil
}

func (s *BaseInSession) handleRTPPacket(b []byte) error {
	atomic.AddUint64(&s.currConnStat.ReadBytesSum, uint64(len(b)))

	if len(b) < rtprtcp.RTPFixedHeaderLength {
		nazalog.Errorf("[%s] read udp packet length invalid. len=%d", s.UniqueKey, len(b))
		return ErrRTSP
	}

	packetType := int(b[1] & 0x7F)
	if packetType != s.sdpLogicCtx.AudioPayloadTypeOrigin && packetType != s.sdpLogicCtx.VideoPayloadTypeOrigin {
		return ErrRTSP
	}

	h, err := rtprtcp.ParseRTPPacket(b)
	if err != nil {
		nazalog.Errorf("[%s] read invalid rtp packet. err=%+v", s.UniqueKey, err)
		return err
	}

	//nazalog.Debugf("%+v", h)
	var pkt rtprtcp.RTPPacket
	pkt.Header = h
	pkt.Raw = b

	switch packetType {
	case s.sdpLogicCtx.VideoPayloadTypeOrigin:
		s.videoSSRC = h.SSRC
		s.observer.OnRTPPacket(pkt)
		s.videoRRProducer.FeedRTPPacket(h.Seq)

		if s.videoUnpacker != nil {
			s.videoUnpacker.Feed(pkt)
		}
	case s.sdpLogicCtx.AudioPayloadTypeOrigin:
		s.audioSSRC = h.SSRC
		s.observer.OnRTPPacket(pkt)
		s.audioRRProducer.FeedRTPPacket(h.Seq)

		if s.audioUnpacker != nil {
			s.audioUnpacker.Feed(pkt)
		}
	default:
		// 因为前面已经判断过type了，所以永远不会走到这
	}

	return nil
}
