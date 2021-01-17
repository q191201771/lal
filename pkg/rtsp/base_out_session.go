// Copyright 2021, Chef.  All rights reserved.
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
	"sync/atomic"
	"time"

	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/naza/pkg/nazaerrors"
	"github.com/q191201771/naza/pkg/nazastring"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
)

type BaseOutSession struct {
	UniqueKey  string
	cmdSession IInterleavedPacketWriter

	rawSDP      []byte
	sdpLogicCtx sdp.LogicContext

	audioRTPConn     *nazanet.UDPConnection
	videoRTPConn     *nazanet.UDPConnection
	audioRTCPConn    *nazanet.UDPConnection
	videoRTCPConn    *nazanet.UDPConnection
	audioRTPChannel  int
	audioRTCPChannel int
	videoRTPChannel  int
	videoRTCPChannel int

	stat         base.StatSession
	currConnStat connection.Stat
	prevConnStat connection.Stat
	staleStat    *connection.Stat

	// only for debug log
	debugLogMaxCount         int
	loggedWriteAudioRTPCount int
	loggedWriteVideoRTPCount int
	loggedReadUDPCount       int
}

func NewBaseOutSession(uniqueKey string, cmdSession IInterleavedPacketWriter) *BaseOutSession {
	s := &BaseOutSession{
		UniqueKey:  uniqueKey,
		cmdSession: cmdSession,
		stat: base.StatSession{
			Protocol:  base.ProtocolRTSP,
			SessionID: uniqueKey,
			StartTime: time.Now().Format("2006-01-02 15:04:05.999"),
		},
		audioRTPChannel:  -1,
		videoRTPChannel:  -1,
		debugLogMaxCount: 3,
	}
	nazalog.Infof("[%s] lifecycle new rtsp BaseOutSession. session=%p", uniqueKey, s)
	return s
}

func (session *BaseOutSession) InitWithSDP(rawSDP []byte, sdpLogicCtx sdp.LogicContext) {
	session.rawSDP = rawSDP
	session.sdpLogicCtx = sdpLogicCtx
}

func (session *BaseOutSession) SetupWithConn(uri string, rtpConn, rtcpConn *nazanet.UDPConnection) error {
	if session.sdpLogicCtx.IsAudioURI(uri) {
		session.audioRTPConn = rtpConn
		session.audioRTCPConn = rtcpConn
	} else if session.sdpLogicCtx.IsVideoURI(uri) {
		session.videoRTPConn = rtpConn
		session.videoRTCPConn = rtcpConn
	} else {
		return ErrRTSP
	}

	go rtpConn.RunLoop(session.onReadUDPPacket)
	go rtcpConn.RunLoop(session.onReadUDPPacket)

	return nil
}

func (session *BaseOutSession) SetupWithChannel(uri string, rtpChannel, rtcpChannel int) error {
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

func (session *BaseOutSession) Dispose() error {
	nazalog.Infof("[%s] lifecycle dispose rtsp BaseOutSession. session=%p", session.UniqueKey, session)
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

func (session *BaseOutSession) HandleInterleavedPacket(b []byte, channel int) {
	switch channel {
	case session.audioRTPChannel:
		fallthrough
	case session.videoRTPChannel:
		nazalog.Warnf("[%s] not supposed to read packet in rtp channel of BaseOutSession. channel=%d, len=%d", session.UniqueKey, channel, len(b))
	case session.audioRTCPChannel:
		fallthrough
	case session.videoRTCPChannel:
		nazalog.Debugf("[%s] read interleaved rtcp packet. b=%s", session.UniqueKey, hex.Dump(nazastring.SubSliceSafety(b, 32)))
	default:
		nazalog.Errorf("[%s] read interleaved packet but channel invalid. channel=%d", session.UniqueKey, channel)
	}
}

func (session *BaseOutSession) WriteRTPPacket(packet rtprtcp.RTPPacket) {
	atomic.AddUint64(&session.currConnStat.WroteBytesSum, uint64(len(packet.Raw)))

	// 发送数据时，保证和sdp的原始类型对应
	t := int(packet.Header.PacketType)
	if session.sdpLogicCtx.IsAudioPayloadTypeOrigin(t) {
		if session.loggedWriteAudioRTPCount < session.debugLogMaxCount {
			nazalog.Debugf("[%s] LOGPACKET. write audio rtp=%+v", session.UniqueKey, packet.Header)
			session.loggedWriteAudioRTPCount++
		}

		if session.audioRTPConn != nil {
			_ = session.audioRTPConn.Write(packet.Raw)
		}
		if session.audioRTPChannel != -1 {
			_ = session.cmdSession.WriteInterleavedPacket(packet.Raw, session.audioRTPChannel)
		}
	} else if session.sdpLogicCtx.IsVideoPayloadTypeOrigin(t) {
		if session.loggedWriteVideoRTPCount < session.debugLogMaxCount {
			nazalog.Debugf("[%s] LOGPACKET. write video rtp=%+v", session.UniqueKey, packet.Header)
			session.loggedWriteVideoRTPCount++
		}

		if session.videoRTPConn != nil {
			_ = session.videoRTPConn.Write(packet.Raw)
		}
		if session.videoRTPChannel != -1 {
			_ = session.cmdSession.WriteInterleavedPacket(packet.Raw, session.videoRTPChannel)
		}
	} else {
		nazalog.Errorf("[%s] write rtp packet but type invalid. type=%d", session.UniqueKey, t)
	}
}

func (session *BaseOutSession) GetStat() base.StatSession {
	session.stat.ReadBytesSum = atomic.LoadUint64(&session.currConnStat.ReadBytesSum)
	session.stat.WroteBytesSum = atomic.LoadUint64(&session.currConnStat.WroteBytesSum)
	return session.stat
}

func (session *BaseOutSession) UpdateStat(interval uint32) {
	readBytesSum := atomic.LoadUint64(&session.currConnStat.ReadBytesSum)
	wroteBytesSum := atomic.LoadUint64(&session.currConnStat.WroteBytesSum)
	rDiff := readBytesSum - session.prevConnStat.ReadBytesSum
	session.stat.ReadBitrate = int(rDiff * 8 / 1024 / uint64(interval))
	wDiff := wroteBytesSum - session.prevConnStat.WroteBytesSum
	session.stat.WriteBitrate = int(wDiff * 8 / 1024 / uint64(interval))
	session.stat.Bitrate = session.stat.WriteBitrate
	session.prevConnStat.ReadBytesSum = readBytesSum
	session.prevConnStat.WroteBytesSum = wroteBytesSum
}

func (session *BaseOutSession) IsAlive() (readAlive, writeAlive bool) {
	readBytesSum := atomic.LoadUint64(&session.currConnStat.ReadBytesSum)
	wroteBytesSum := atomic.LoadUint64(&session.currConnStat.WroteBytesSum)
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

func (session *BaseOutSession) onReadUDPPacket(b []byte, rAddr *net.UDPAddr, err error) bool {
	// TODO chef: impl me

	if session.loggedReadUDPCount < session.debugLogMaxCount {
		nazalog.Debugf("[%s] LOGPACKET. read udp=%s", session.UniqueKey, hex.Dump(nazastring.SubSliceSafety(b, 32)))
		session.loggedReadUDPCount++
	}
	return true
}
