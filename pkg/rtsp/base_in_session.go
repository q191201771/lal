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

// BaseInSession会向上层回调两种格式的数据(本质上是一份数据，业务方可自由选择使用)：
// 1. 原始的rtp packet
// 2. rtp合并后的av packet
//
type BaseInSessionObserver interface {
	OnSdp(sdpCtx sdp.LogicContext)

	// 回调收到的RTP包
	OnRtpPacket(pkt rtprtcp.RtpPacket)

	// @param pkt: pkt结构体中字段含义见rtprtcp.OnAvPacket
	OnAvPacket(pkt base.AvPacket)
}

type BaseInSession struct {
	uniqueKey  string // 使用上层Session的值
	cmdSession IInterleavedPacketWriter

	observer BaseInSessionObserver

	audioRtpConn     *nazanet.UdpConnection
	videoRtpConn     *nazanet.UdpConnection
	audioRtcpConn    *nazanet.UdpConnection
	videoRtcpConn    *nazanet.UdpConnection
	audioRtpChannel  int
	audioRtcpChannel int
	videoRtpChannel  int
	videoRtcpChannel int

	currConnStat connection.StatAtomic
	prevConnStat connection.Stat
	staleStat    *connection.Stat
	stat         base.StatSession

	mu              sync.Mutex
	sdpCtx          sdp.LogicContext // const after set
	avPacketQueue   *AvPacketQueue
	audioRrProducer *rtprtcp.RrProducer
	videoRrProducer *rtprtcp.RrProducer

	audioUnpacker rtprtcp.IRtpUnpacker
	videoUnpacker rtprtcp.IRtpUnpacker

	audioSsrc nazaatomic.Uint32
	videoSsrc nazaatomic.Uint32

	// only for debug log
	debugLogMaxCount        uint32
	loggedReadAudioRtpCount nazaatomic.Uint32
	loggedReadVideoRtpCount nazaatomic.Uint32
	loggedReadRtcpCount     nazaatomic.Uint32
	loggedReadSrCount       nazaatomic.Uint32

	disposeOnce sync.Once
	waitChan    chan error
}

func NewBaseInSession(uniqueKey string, cmdSession IInterleavedPacketWriter) *BaseInSession {
	s := &BaseInSession{
		uniqueKey: uniqueKey,
		stat: base.StatSession{
			Protocol:  base.ProtocolRtsp,
			SessionId: uniqueKey,
			StartTime: time.Now().Format("2006-01-02 15:04:05.999"),
		},
		cmdSession:       cmdSession,
		debugLogMaxCount: 3,
		waitChan:         make(chan error, 1),
	}
	nazalog.Infof("[%s] lifecycle new rtsp BaseInSession. session=%p", uniqueKey, s)
	return s
}

func NewBaseInSessionWithObserver(uniqueKey string, cmdSession IInterleavedPacketWriter, observer BaseInSessionObserver) *BaseInSession {
	s := NewBaseInSession(uniqueKey, cmdSession)
	s.observer = observer
	return s
}

func (session *BaseInSession) InitWithSdp(sdpCtx sdp.LogicContext) {
	session.mu.Lock()
	session.sdpCtx = sdpCtx
	session.mu.Unlock()

	if session.sdpCtx.IsAudioUnpackable() {
		session.audioUnpacker = rtprtcp.DefaultRtpUnpackerFactory(session.sdpCtx.GetAudioPayloadTypeBase(), session.sdpCtx.AudioClockRate, unpackerItemMaxSize, session.onAvPacketUnpacked)
	} else {
		nazalog.Warnf("[%s] audio unpacker not support for this type yet. logicCtx=%+v", session.uniqueKey, session.sdpCtx)
	}
	if session.sdpCtx.IsVideoUnpackable() {
		session.videoUnpacker = rtprtcp.DefaultRtpUnpackerFactory(session.sdpCtx.GetVideoPayloadTypeBase(), session.sdpCtx.VideoClockRate, unpackerItemMaxSize, session.onAvPacketUnpacked)
	} else {
		nazalog.Warnf("[%s] video unpacker not support this type yet. logicCtx=%+v", session.uniqueKey, session.sdpCtx)
	}

	session.audioRrProducer = rtprtcp.NewRrProducer(session.sdpCtx.AudioClockRate)
	session.videoRrProducer = rtprtcp.NewRrProducer(session.sdpCtx.VideoClockRate)

	if session.sdpCtx.IsAudioUnpackable() && session.sdpCtx.IsVideoUnpackable() {
		session.mu.Lock()
		session.avPacketQueue = NewAvPacketQueue(session.onAvPacket)
		session.mu.Unlock()
	}

	if session.observer != nil {
		session.observer.OnSdp(session.sdpCtx)
	}
}

// 如果没有设置回调监听对象，可以通过该函数设置，调用方保证调用该函数发生在调用InitWithSdp之后
func (session *BaseInSession) SetObserver(observer BaseInSessionObserver) {
	session.observer = observer

	// 避免在当前协程回调，降低业务方使用负担，不必担心设置监听对象和回调函数中锁重入 TODO(chef): 更好的方式
	go func() {
		session.observer.OnSdp(session.sdpCtx)
	}()
}

func (session *BaseInSession) SetupWithConn(uri string, rtpConn, rtcpConn *nazanet.UdpConnection) error {
	if session.sdpCtx.IsAudioUri(uri) {
		session.audioRtpConn = rtpConn
		session.audioRtcpConn = rtcpConn
	} else if session.sdpCtx.IsVideoUri(uri) {
		session.videoRtpConn = rtpConn
		session.videoRtcpConn = rtcpConn
	} else {
		return ErrRtsp
	}

	go rtpConn.RunLoop(session.onReadRtpPacket)
	go rtcpConn.RunLoop(session.onReadRtcpPacket)

	return nil
}

func (session *BaseInSession) SetupWithChannel(uri string, rtpChannel, rtcpChannel int) error {
	if session.sdpCtx.IsAudioUri(uri) {
		session.audioRtpChannel = rtpChannel
		session.audioRtcpChannel = rtcpChannel
		return nil
	} else if session.sdpCtx.IsVideoUri(uri) {
		session.videoRtpChannel = rtpChannel
		session.videoRtcpChannel = rtcpChannel
		return nil
	}
	return ErrRtsp
}

// ---------------------------------------------------------------------------------------------------------------------
// IClientSessionLifecycle interface
// ---------------------------------------------------------------------------------------------------------------------

// Dispose 文档请参考： IClientSessionLifecycle interface
//
func (session *BaseInSession) Dispose() error {
	return session.dispose(nil)
}

// WaitChan 文档请参考： IClientSessionLifecycle interface
//
// 注意，目前只有一种情况，即上层主动调用Dispose函数，此时error为nil
//
func (session *BaseInSession) WaitChan() <-chan error {
	return session.waitChan
}

// ---------------------------------------------------------------------------------------------------------------------

func (session *BaseInSession) GetSdp() sdp.LogicContext {
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.sdpCtx
}

func (session *BaseInSession) HandleInterleavedPacket(b []byte, channel int) {
	switch channel {
	case session.audioRtpChannel:
		fallthrough
	case session.videoRtpChannel:
		_ = session.handleRtpPacket(b)
	case session.audioRtcpChannel:
		fallthrough
	case session.videoRtcpChannel:
		_ = session.handleRtcpPacket(b, nil)
	default:
		nazalog.Errorf("[%s] read interleaved packet but channel invalid. channel=%d", session.uniqueKey, channel)
	}
}

// 发现pull时，需要先给对端发送数据，才能收到数据
func (session *BaseInSession) WriteRtpRtcpDummy() {
	if session.videoRtpConn != nil {
		_ = session.videoRtpConn.Write(dummyRtpPacket)
	}
	if session.videoRtcpConn != nil {
		_ = session.videoRtcpConn.Write(dummyRtcpPacket)
	}
	if session.audioRtpConn != nil {
		_ = session.audioRtpConn.Write(dummyRtpPacket)
	}
	if session.audioRtcpConn != nil {
		_ = session.audioRtcpConn.Write(dummyRtcpPacket)
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
func (session *BaseInSession) onAvPacketUnpacked(pkt base.AvPacket) {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.avPacketQueue != nil {
		session.avPacketQueue.Feed(pkt)
	} else {
		session.observer.OnAvPacket(pkt)
	}
}

// callback by avpacket queue
func (session *BaseInSession) onAvPacket(pkt base.AvPacket) {
	session.observer.OnAvPacket(pkt)
}

// callback by UDPConnection
func (session *BaseInSession) onReadRtpPacket(b []byte, rAddr *net.UDPAddr, err error) bool {
	if err != nil {
		// TODO(chef):
		// read udp [::]:30008: use of closed network connection
		// 可以退出loop，看是在上层退还是下层退，但是要注意每次read都判断的开销
		nazalog.Warnf("[%s] read udp packet failed. err=%+v", session.uniqueKey, err)
		return true
	}

	_ = session.handleRtpPacket(b)
	return true
}

// callback by UDPConnection
func (session *BaseInSession) onReadRtcpPacket(b []byte, rAddr *net.UDPAddr, err error) bool {
	if err != nil {
		nazalog.Warnf("[%s] read udp packet failed. err=%+v", session.uniqueKey, err)
		return true
	}

	_ = session.handleRtcpPacket(b, rAddr)
	return true
}

// @param rAddr 对端地址，往对端发送数据时使用，注意，如果nil，则表示是interleaved模式，我们直接往TCP连接发数据
func (session *BaseInSession) handleRtcpPacket(b []byte, rAddr *net.UDPAddr) error {
	session.currConnStat.ReadBytesSum.Add(uint64(len(b)))

	if len(b) <= 0 {
		nazalog.Errorf("[%s] handleRtcpPacket but length invalid. len=%d", session.uniqueKey, len(b))
		return ErrRtsp
	}

	if session.loggedReadRtcpCount.Load() < session.debugLogMaxCount {
		nazalog.Debugf("[%s] LOGPACKET. read rtcp=%s", session.uniqueKey, hex.Dump(nazastring.SubSliceSafety(b, 32)))
		session.loggedReadRtcpCount.Increment()
	}

	packetType := b[1]

	switch packetType {
	case rtprtcp.RtcpPacketTypeSr:
		sr := rtprtcp.ParseSr(b)
		if session.loggedReadSrCount.Load() < session.debugLogMaxCount {
			nazalog.Debugf("[%s] LOGPACKET. %+v", session.uniqueKey, sr)
			session.loggedReadSrCount.Increment()
		}
		var rrBuf []byte
		switch sr.SenderSsrc {
		case session.audioSsrc.Load():
			session.mu.Lock()
			rrBuf = session.audioRrProducer.Produce(sr.GetMiddleNtp())
			session.mu.Unlock()
			if rrBuf != nil {
				if rAddr != nil {
					_ = session.audioRtcpConn.Write2Addr(rrBuf, rAddr)
				} else {
					_ = session.cmdSession.WriteInterleavedPacket(rrBuf, session.audioRtcpChannel)
				}
				session.currConnStat.WroteBytesSum.Add(uint64(len(b)))
			}
		case session.videoSsrc.Load():
			session.mu.Lock()
			rrBuf = session.videoRrProducer.Produce(sr.GetMiddleNtp())
			session.mu.Unlock()
			if rrBuf != nil {
				if rAddr != nil {
					_ = session.videoRtcpConn.Write2Addr(rrBuf, rAddr)
				} else {
					_ = session.cmdSession.WriteInterleavedPacket(rrBuf, session.videoRtcpChannel)
				}
				session.currConnStat.WroteBytesSum.Add(uint64(len(b)))
			}
		default:
			// ffmpeg推流时，会在发送第一个RTP包之前就发送一个SR，所以关闭这个警告日志
			//nazalog.Warnf("[%s] read rtcp sr but senderSsrc invalid. senderSsrc=%d, audio=%d, video=%d",
			//	p.uniqueKey, sr.SenderSsrc, p.audioSsrc, p.videoSsrc)
			return ErrRtsp
		}
	default:
		nazalog.Warnf("[%s] handleRtcpPacket but type unknown. type=%d", session.uniqueKey, b[1])
		return ErrRtsp
	}

	return nil
}

func (session *BaseInSession) handleRtpPacket(b []byte) error {
	session.currConnStat.ReadBytesSum.Add(uint64(len(b)))

	if len(b) < rtprtcp.RtpFixedHeaderLength {
		nazalog.Errorf("[%s] handleRtpPacket but length invalid. len=%d", session.uniqueKey, len(b))
		return ErrRtsp
	}

	packetType := int(b[1] & 0x7F)
	if !session.sdpCtx.IsPayloadTypeOrigin(packetType) {
		nazalog.Errorf("[%s] handleRtpPacket but type invalid. type=%d", session.uniqueKey, packetType)
		return ErrRtsp
	}

	h, err := rtprtcp.ParseRtpHeader(b)
	if err != nil {
		nazalog.Errorf("[%s] handleRtpPacket invalid rtp packet. err=%+v", session.uniqueKey, err)
		return err
	}

	var pkt rtprtcp.RtpPacket
	pkt.Header = h
	pkt.Raw = b

	// 接收数据时，保证了sdp的原始类型对应
	if session.sdpCtx.IsAudioPayloadTypeOrigin(packetType) {
		if session.loggedReadAudioRtpCount.Load() < session.debugLogMaxCount {
			nazalog.Debugf("[%s] LOGPACKET. read audio rtp=%+v, len=%d", session.uniqueKey, h, len(b))
			session.loggedReadAudioRtpCount.Increment()
		}

		session.audioSsrc.Store(h.Ssrc)
		session.observer.OnRtpPacket(pkt)
		session.mu.Lock()
		session.audioRrProducer.FeedRtpPacket(h.Seq)
		session.mu.Unlock()

		if session.audioUnpacker != nil {
			session.audioUnpacker.Feed(pkt)
		}
	} else if session.sdpCtx.IsVideoPayloadTypeOrigin(packetType) {
		if session.loggedReadVideoRtpCount.Load() < session.debugLogMaxCount {
			nazalog.Debugf("[%s] LOGPACKET. read video rtp=%+v, len=%d", session.uniqueKey, h, len(b))
			session.loggedReadVideoRtpCount.Increment()
		}

		session.videoSsrc.Store(h.Ssrc)
		session.observer.OnRtpPacket(pkt)
		session.mu.Lock()
		session.videoRrProducer.FeedRtpPacket(h.Seq)
		session.mu.Unlock()

		if session.videoUnpacker != nil {
			session.videoUnpacker.Feed(pkt)
		}
	} else {
		// noop 因为前面已经判断过type了，所以永远不会走到这
	}

	return nil
}

func (session *BaseInSession) dispose(err error) error {
	var retErr error
	session.disposeOnce.Do(func() {
		nazalog.Infof("[%s] lifecycle dispose rtsp BaseInSession. session=%p", session.uniqueKey, session)
		var e1, e2, e3, e4 error
		if session.audioRtpConn != nil {
			e1 = session.audioRtpConn.Dispose()
		}
		if session.audioRtcpConn != nil {
			e2 = session.audioRtcpConn.Dispose()
		}
		if session.videoRtpConn != nil {
			e3 = session.videoRtpConn.Dispose()
		}
		if session.videoRtcpConn != nil {
			e4 = session.videoRtcpConn.Dispose()
		}

		session.waitChan <- nil

		retErr = nazaerrors.CombineErrors(e1, e2, e3, e4)
	})
	return retErr
}
