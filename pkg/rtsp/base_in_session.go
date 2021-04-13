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
	"sync"
	"time"

	"github.com/q191201771/naza/pkg/nazaatomic"

	"github.com/q191201771/naza/pkg/nazaerrors"
	"github.com/q191201771/naza/pkg/nazastring"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
)

// 聚合PubSession和PullSession，也即流数据是输入类型的session

// BaseInSession会向上层回调两种格式的数据：
// 1. 原始的rtp packet
// 2. rtp合并后的av packet
type BaseInSessionObserver interface {
	OnRTPPacket(pkt rtprtcp.RTPPacket)

	// @param asc: AAC AudioSpecificConfig，注意，如果不存在音频或音频不为AAC，则为nil
	// @param vps, sps, pps 如果都为nil，则没有视频，如果sps, pps不为nil，则vps不为nil是H265，vps为nil是H264
	//
	// 注意，4个参数可能同时为nil
	OnAVConfig(asc, vps, sps, pps []byte)

	// @param pkt: pkt结构体中字段含义见rtprtcp.OnAVPacket
	OnAVPacket(pkt base.AVPacket)
}

type BaseInSession struct {
	uniqueKey  string // 使用上层Session的值
	cmdSession IInterleavedPacketWriter

	observer BaseInSessionObserver

	audioRTPConn     *nazanet.UDPConnection
	videoRTPConn     *nazanet.UDPConnection
	audioRTCPConn    *nazanet.UDPConnection
	videoRTCPConn    *nazanet.UDPConnection
	audioRTPChannel  int
	audioRTCPChannel int
	videoRTPChannel  int
	videoRTCPChannel int

	currConnStat connection.StatAtomic
	prevConnStat connection.Stat
	staleStat    *connection.Stat
	stat         base.StatSession

	mu              sync.Mutex
	rawSDP          []byte           // const after set
	sdpLogicCtx     sdp.LogicContext // const after set
	avPacketQueue   *AVPacketQueue
	audioRRProducer *rtprtcp.RRProducer
	videoRRProducer *rtprtcp.RRProducer

	audioUnpacker rtprtcp.IRTPUnpacker
	videoUnpacker rtprtcp.IRTPUnpacker

	audioSSRC nazaatomic.Uint32
	videoSSRC nazaatomic.Uint32

	// only for debug log
	debugLogMaxCount        uint32
	loggedReadAudioRTPCount nazaatomic.Uint32
	loggedReadVideoRTPCount nazaatomic.Uint32
	loggedReadRTCPCount     nazaatomic.Uint32
	loggedReadSRCount       nazaatomic.Uint32
}

func NewBaseInSession(uniqueKey string, cmdSession IInterleavedPacketWriter) *BaseInSession {
	s := &BaseInSession{
		uniqueKey: uniqueKey,
		stat: base.StatSession{
			Protocol:  base.ProtocolRTSP,
			SessionID: uniqueKey,
			StartTime: time.Now().Format("2006-01-02 15:04:05.999"),
		},
		cmdSession:       cmdSession,
		debugLogMaxCount: 3,
	}
	nazalog.Infof("[%s] lifecycle new rtsp BaseInSession. session=%p", uniqueKey, s)
	return s
}

func NewBaseInSessionWithObserver(uniqueKey string, cmdSession IInterleavedPacketWriter, observer BaseInSessionObserver) *BaseInSession {
	s := NewBaseInSession(uniqueKey, cmdSession)
	s.observer = observer
	return s
}

func (session *BaseInSession) InitWithSDP(rawSDP []byte, sdpLogicCtx sdp.LogicContext) {
	session.mu.Lock()
	session.rawSDP = rawSDP
	session.sdpLogicCtx = sdpLogicCtx
	session.mu.Unlock()

	if session.sdpLogicCtx.IsAudioUnpackable() {
		session.audioUnpacker = rtprtcp.DefaultRTPUnpackerFactory(session.sdpLogicCtx.GetAudioPayloadTypeBase(), session.sdpLogicCtx.AudioClockRate, unpackerItemMaxSize, session.onAVPacketUnpacked)
	} else {
		nazalog.Warnf("[%s] audio unpacker not support for this type yet.", session.uniqueKey)
	}
	if session.sdpLogicCtx.IsVideoUnpackable() {
		session.videoUnpacker = rtprtcp.DefaultRTPUnpackerFactory(session.sdpLogicCtx.GetVideoPayloadTypeBase(), session.sdpLogicCtx.VideoClockRate, unpackerItemMaxSize, session.onAVPacketUnpacked)
	} else {
		nazalog.Warnf("[%s] video unpacker not support this type yet.", session.uniqueKey)
	}

	session.audioRRProducer = rtprtcp.NewRRProducer(session.sdpLogicCtx.AudioClockRate)
	session.videoRRProducer = rtprtcp.NewRRProducer(session.sdpLogicCtx.VideoClockRate)

	if session.sdpLogicCtx.IsAudioUnpackable() && session.sdpLogicCtx.IsVideoUnpackable() {
		session.mu.Lock()
		session.avPacketQueue = NewAVPacketQueue(session.onAVPacket)
		session.mu.Unlock()
	}

	if session.observer != nil {
		session.observer.OnAVConfig(session.sdpLogicCtx.ASC, session.sdpLogicCtx.VPS, session.sdpLogicCtx.SPS, session.sdpLogicCtx.PPS)
	}
}

// 如果没有设置回调监听对象，可以通过该函数设置，调用方保证调用该函数发生在调用InitWithSDP之后
func (session *BaseInSession) SetObserver(observer BaseInSessionObserver) {
	session.observer = observer

	session.observer.OnAVConfig(session.sdpLogicCtx.ASC, session.sdpLogicCtx.VPS, session.sdpLogicCtx.SPS, session.sdpLogicCtx.PPS)
}

func (session *BaseInSession) SetupWithConn(uri string, rtpConn, rtcpConn *nazanet.UDPConnection) error {
	if session.sdpLogicCtx.IsAudioURI(uri) {
		session.audioRTPConn = rtpConn
		session.audioRTCPConn = rtcpConn
	} else if session.sdpLogicCtx.IsVideoURI(uri) {
		session.videoRTPConn = rtpConn
		session.videoRTCPConn = rtcpConn
	} else {
		return ErrRTSP
	}

	go rtpConn.RunLoop(session.onReadRTPPacket)
	go rtcpConn.RunLoop(session.onReadRTCPPacket)

	return nil
}

func (session *BaseInSession) SetupWithChannel(uri string, rtpChannel, rtcpChannel int) error {
	if session.sdpLogicCtx.IsAudioURI(uri) {
		session.audioRTPChannel = rtpChannel
		session.audioRTCPChannel = rtcpChannel
		return nil
	} else if session.sdpLogicCtx.IsVideoURI(uri) {
		session.videoRTPChannel = rtpChannel
		session.videoRTCPChannel = rtcpChannel
		return nil
	}
	return ErrRTSP
}

func (session *BaseInSession) Dispose() error {
	nazalog.Infof("[%s] lifecycle dispose rtsp BaseInSession. session=%p", session.uniqueKey, session)
	var e1, e2, e3, e4 error
	if session.audioRTPConn != nil {
		e1 = session.audioRTPConn.Dispose()
	}
	if session.audioRTCPConn != nil {
		e2 = session.audioRTCPConn.Dispose()
	}
	if session.videoRTPConn != nil {
		e3 = session.videoRTPConn.Dispose()
	}
	if session.videoRTCPConn != nil {
		e4 = session.videoRTCPConn.Dispose()
	}
	return nazaerrors.CombineErrors(e1, e2, e3, e4)
}

func (session *BaseInSession) GetSDP() ([]byte, sdp.LogicContext) {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.rawSDP, session.sdpLogicCtx
}

func (session *BaseInSession) HandleInterleavedPacket(b []byte, channel int) {
	switch channel {
	case session.audioRTPChannel:
		fallthrough
	case session.videoRTPChannel:
		_ = session.handleRTPPacket(b)
	case session.audioRTCPChannel:
		fallthrough
	case session.videoRTCPChannel:
		_ = session.handleRTCPPacket(b, nil)
	default:
		nazalog.Errorf("[%s] read interleaved packet but channel invalid. channel=%d", session.uniqueKey, channel)
	}
}

// 发现pull时，需要先给对端发送数据，才能收到数据
func (session *BaseInSession) WriteRTPRTCPDummy() {
	if session.videoRTPConn != nil {
		_ = session.videoRTPConn.Write(dummyRTPPacket)
	}
	if session.videoRTCPConn != nil {
		_ = session.videoRTCPConn.Write(dummyRTCPPacket)
	}
	if session.audioRTPConn != nil {
		_ = session.audioRTPConn.Write(dummyRTPPacket)
	}
	if session.audioRTCPConn != nil {
		_ = session.audioRTCPConn.Write(dummyRTCPPacket)
	}
}

func (session *BaseInSession) GetStat() base.StatSession {
	session.stat.ReadBytesSum = session.currConnStat.ReadBytesSum.Load()
	session.stat.WroteBytesSum = session.currConnStat.WroteBytesSum.Load()
	return session.stat
}

func (session *BaseInSession) UpdateStat(intervalSec uint32) {
	readBytesSum := session.currConnStat.ReadBytesSum.Load()
	wroteBytesSum := session.currConnStat.WroteBytesSum.Load()
	rDiff := readBytesSum - session.prevConnStat.ReadBytesSum
	session.stat.ReadBitrate = int(rDiff * 8 / 1024 / uint64(intervalSec))
	wDiff := wroteBytesSum - session.prevConnStat.WroteBytesSum
	session.stat.WriteBitrate = int(wDiff * 8 / 1024 / uint64(intervalSec))
	session.stat.Bitrate = session.stat.ReadBitrate
	session.prevConnStat.ReadBytesSum = readBytesSum
	session.prevConnStat.WroteBytesSum = wroteBytesSum
}

func (session *BaseInSession) IsAlive() (readAlive, writeAlive bool) {
	readBytesSum := session.currConnStat.ReadBytesSum.Load()
	wroteBytesSum := session.currConnStat.WroteBytesSum.Load()
	if session.staleStat == nil {
		session.staleStat = new(connection.Stat)
		session.staleStat.ReadBytesSum = readBytesSum
		session.staleStat.WroteBytesSum = wroteBytesSum
		return true, true
	}

	readAlive = !(readBytesSum-session.staleStat.ReadBytesSum == 0)
	writeAlive = !(wroteBytesSum-session.staleStat.WroteBytesSum == 0)
	session.staleStat.ReadBytesSum = readBytesSum
	session.staleStat.WroteBytesSum = wroteBytesSum
	return
}

func (session *BaseInSession) UniqueKey() string {
	return session.uniqueKey
}

// callback by RTPUnpacker
func (session *BaseInSession) onAVPacketUnpacked(pkt base.AVPacket) {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.avPacketQueue != nil {
		session.avPacketQueue.Feed(pkt)
	} else {
		session.observer.OnAVPacket(pkt)
	}
}

// callback by avpacket queue
func (session *BaseInSession) onAVPacket(pkt base.AVPacket) {
	session.observer.OnAVPacket(pkt)
}

// callback by UDPConnection
func (session *BaseInSession) onReadRTPPacket(b []byte, rAddr *net.UDPAddr, err error) bool {
	if err != nil {
		nazalog.Errorf("[%s] read udp packet failed. err=%+v", session.uniqueKey, err)
		return true
	}

	_ = session.handleRTPPacket(b)
	return true
}

// callback by UDPConnection
func (session *BaseInSession) onReadRTCPPacket(b []byte, rAddr *net.UDPAddr, err error) bool {
	if err != nil {
		nazalog.Errorf("[%s] read udp packet failed. err=%+v", session.uniqueKey, err)
		return true
	}

	_ = session.handleRTCPPacket(b, rAddr)
	return true
}

// @param rAddr 对端地址，往对端发送数据时使用，注意，如果nil，则表示是interleaved模式，我们直接往TCP连接发数据
func (session *BaseInSession) handleRTCPPacket(b []byte, rAddr *net.UDPAddr) error {
	session.currConnStat.ReadBytesSum.Add(uint64(len(b)))

	if len(b) <= 0 {
		nazalog.Errorf("[%s] handleRTCPPacket but length invalid. len=%d", session.uniqueKey, len(b))
		return ErrRTSP
	}

	if session.loggedReadRTCPCount.Load() < session.debugLogMaxCount {
		nazalog.Debugf("[%s] LOGPACKET. read rtcp=%s", session.uniqueKey, hex.Dump(nazastring.SubSliceSafety(b, 32)))
		session.loggedReadRTCPCount.Increment()
	}

	packetType := b[1]

	switch packetType {
	case rtprtcp.RTCPPacketTypeSR:
		sr := rtprtcp.ParseSR(b)
		if session.loggedReadSRCount.Load() < session.debugLogMaxCount {
			nazalog.Debugf("[%s] LOGPACKET. %+v", session.uniqueKey, sr)
			session.loggedReadSRCount.Increment()
		}
		var rrBuf []byte
		switch sr.SenderSSRC {
		case session.audioSSRC.Load():
			session.mu.Lock()
			rrBuf = session.audioRRProducer.Produce(sr.GetMiddleNTP())
			session.mu.Unlock()
			if rrBuf != nil {
				if rAddr != nil {
					_ = session.audioRTCPConn.Write2Addr(rrBuf, rAddr)
				} else {
					_ = session.cmdSession.WriteInterleavedPacket(rrBuf, session.audioRTCPChannel)
				}
				session.currConnStat.WroteBytesSum.Add(uint64(len(b)))
			}
		case session.videoSSRC.Load():
			session.mu.Lock()
			rrBuf = session.videoRRProducer.Produce(sr.GetMiddleNTP())
			session.mu.Unlock()
			if rrBuf != nil {
				if rAddr != nil {
					_ = session.videoRTCPConn.Write2Addr(rrBuf, rAddr)
				} else {
					_ = session.cmdSession.WriteInterleavedPacket(rrBuf, session.videoRTCPChannel)
				}
				session.currConnStat.WroteBytesSum.Add(uint64(len(b)))
			}
		default:
			// ffmpeg推流时，会在发送第一个RTP包之前就发送一个SR，所以关闭这个警告日志
			//nazalog.Warnf("[%s] read rtcp sr but senderSSRC invalid. senderSSRC=%d, audio=%d, video=%d",
			//	p.uniqueKey, sr.SenderSSRC, p.audioSSRC, p.videoSSRC)
			return ErrRTSP
		}
	default:
		nazalog.Warnf("[%s] handleRTCPPacket but type unknown. type=%d", session.uniqueKey, b[1])
		return ErrRTSP
	}

	return nil
}

func (session *BaseInSession) handleRTPPacket(b []byte) error {
	session.currConnStat.ReadBytesSum.Add(uint64(len(b)))

	if len(b) < rtprtcp.RTPFixedHeaderLength {
		nazalog.Errorf("[%s] handleRTPPacket but length invalid. len=%d", session.uniqueKey, len(b))
		return ErrRTSP
	}

	packetType := int(b[1] & 0x7F)
	if !session.sdpLogicCtx.IsPayloadTypeOrigin(packetType) {
		nazalog.Errorf("[%s] handleRTPPacket but type invalid. type=%d", session.uniqueKey, packetType)
		return ErrRTSP
	}

	h, err := rtprtcp.ParseRTPHeader(b)
	if err != nil {
		nazalog.Errorf("[%s] handleRTPPacket invalid rtp packet. err=%+v", session.uniqueKey, err)
		return err
	}

	var pkt rtprtcp.RTPPacket
	pkt.Header = h
	pkt.Raw = b

	// 接收数据时，保证了sdp的原始类型对应
	if session.sdpLogicCtx.IsAudioPayloadTypeOrigin(packetType) {
		if session.loggedReadAudioRTPCount.Load() < session.debugLogMaxCount {
			nazalog.Debugf("[%s] LOGPACKET. read audio rtp=%+v, len=%d", session.uniqueKey, h, len(b))
			session.loggedReadAudioRTPCount.Increment()
		}

		session.audioSSRC.Store(h.SSRC)
		session.observer.OnRTPPacket(pkt)
		session.mu.Lock()
		session.audioRRProducer.FeedRTPPacket(h.Seq)
		session.mu.Unlock()

		if session.audioUnpacker != nil {
			session.audioUnpacker.Feed(pkt)
		}
	} else if session.sdpLogicCtx.IsVideoPayloadTypeOrigin(packetType) {
		if session.loggedReadVideoRTPCount.Load() < session.debugLogMaxCount {
			nazalog.Debugf("[%s] LOGPACKET. read video rtp=%+v, len=%d", session.uniqueKey, h, len(b))
			session.loggedReadVideoRTPCount.Increment()
		}

		session.videoSSRC.Store(h.SSRC)
		session.observer.OnRTPPacket(pkt)
		session.mu.Lock()
		session.videoRRProducer.FeedRTPPacket(h.Seq)
		session.mu.Unlock()

		if session.videoUnpacker != nil {
			session.videoUnpacker.Feed(pkt)
		}
	} else {
		// noop 因为前面已经判断过type了，所以永远不会走到这
	}

	return nil
}
