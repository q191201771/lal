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
	"sync/atomic"
	"time"

	"github.com/q191201771/naza/pkg/connection"

	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
)

// TODO chef: 主动发送SR

type SubSession struct {
	UniqueKey  string // const after ctor
	cmdSession *ServerCommandSession
	urlCtx     base.URLContext

	//StreamName string // const after ctor

	rawSDP      []byte           // const after set
	sdpLogicCtx sdp.LogicContext // const after set

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
}

func NewSubSession(urlCtx base.URLContext, cmdSession *ServerCommandSession) *SubSession {
	uk := base.GenUniqueKey(base.UKPRTSPSubSession)
	ss := &SubSession{
		UniqueKey:  uk,
		urlCtx:     urlCtx,
		cmdSession: cmdSession,
		stat: base.StatSession{
			Protocol:   base.ProtocolRTSP,
			SessionID:  uk,
			StartTime:  time.Now().Format("2006-01-02 15:04:05.999"),
			RemoteAddr: cmdSession.conn.RemoteAddr().String(),
		},
		audioRTPChannel: -1,
		videoRTPChannel: -1,
	}
	nazalog.Infof("[%s] lifecycle new rtsp SubSession. session=%p, streamName=%s", uk, ss, urlCtx.LastItemOfPath)
	return ss
}

func (s *SubSession) InitWithSDP(rawSDP []byte, sdpLogicCtx sdp.LogicContext) {
	s.rawSDP = rawSDP
	s.sdpLogicCtx = sdpLogicCtx
}

func (s *SubSession) SetupWithConn(uri string, rtpConn, rtcpConn *nazanet.UDPConnection) error {
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

func (s *SubSession) SetupWithChannel(uri string, rtpChannel, rtcpChannel int, remoteAddr string) error {
	s.stat.RemoteAddr = remoteAddr

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

func (s *SubSession) Dispose() {
	nazalog.Infof("[%s] lifecycle dispose rtsp SubSession.", s.UniqueKey)

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

func (s *SubSession) GetStat() base.StatSession {
	s.stat.ReadBytesSum = atomic.LoadUint64(&s.currConnStat.ReadBytesSum)
	s.stat.WroteBytesSum = atomic.LoadUint64(&s.currConnStat.WroteBytesSum)
	return s.stat
}

func (s *SubSession) UpdateStat(interval uint32) {
	readBytesSum := atomic.LoadUint64(&s.currConnStat.ReadBytesSum)
	wroteBytesSum := atomic.LoadUint64(&s.currConnStat.WroteBytesSum)
	rDiff := readBytesSum - s.prevConnStat.ReadBytesSum
	s.stat.ReadBitrate = int(rDiff * 8 / 1024 / uint64(interval))
	wDiff := wroteBytesSum - s.prevConnStat.WroteBytesSum
	s.stat.WriteBitrate = int(wDiff * 8 / 1024 / uint64(interval))
	s.stat.Bitrate = s.stat.WriteBitrate
	s.prevConnStat.ReadBytesSum = readBytesSum
	s.prevConnStat.WroteBytesSum = wroteBytesSum
}

func (s *SubSession) IsAlive() (readAlive, writeAlive bool) {
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

func (s *SubSession) AppName() string {
	return s.urlCtx.PathWithoutLastItem
}

func (s *SubSession) StreamName() string {
	return s.urlCtx.LastItemOfPath
}

func (s *SubSession) RawQuery() string {
	return s.urlCtx.RawQuery
}

func (s *SubSession) WriteRTPPacket(packet rtprtcp.RTPPacket) {
	atomic.AddUint64(&s.currConnStat.WroteBytesSum, uint64(len(packet.Raw)))

	switch packet.Header.PacketType {
	case base.RTPPacketTypeAVCOrHEVC:
		if s.videoRTPConn != nil {
			_ = s.videoRTPConn.Write(packet.Raw)
		}
		if s.videoRTPChannel != -1 {
			_ = s.cmdSession.Write(s.videoRTPChannel, packet.Raw)
		}
	case base.RTPPacketTypeAAC:
		if s.audioRTPConn != nil {
			_ = s.audioRTPConn.Write(packet.Raw)
		}
		if s.audioRTPChannel != -1 {
			_ = s.cmdSession.Write(s.audioRTPChannel, packet.Raw)
		}
	default:
		nazalog.Errorf("[%s] write rtp packet but type invalid. type=%d", s.UniqueKey, packet.Header.PacketType)
	}
}

func (s *SubSession) onReadUDPPacket(b []byte, rAddr *net.UDPAddr, err error) bool {
	// TODO chef: impl me
	//nazalog.Errorf("[%s] SubSession::onReadUDPPacket. %s", s.UniqueKey, hex.Dump(b))
	return true
}
