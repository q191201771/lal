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
	"sync"

	"github.com/q191201771/naza/pkg/nazaatomic"

	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/naza/pkg/nazabytes"
	"github.com/q191201771/naza/pkg/nazaerrors"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazanet"
)

// BaseOutSession out的含义是音视频由本端发送至对端
//
type BaseOutSession struct {
	uniqueKey  string
	cmdSession IInterleavedPacketWriter

	sdpCtx sdp.LogicContext

	audioRtpConn     *nazanet.UdpConnection
	videoRtpConn     *nazanet.UdpConnection
	audioRtcpConn    *nazanet.UdpConnection
	videoRtcpConn    *nazanet.UdpConnection
	audioRtpChannel  int
	audioRtcpChannel int
	videoRtpChannel  int
	videoRtcpChannel int

	stat         base.StatSession
	currConnStat connection.StatAtomic
	prevConnStat connection.Stat
	staleStat    *connection.Stat

	// only for debug log
	debugLogMaxCount         int
	loggedWriteAudioRtpCount int
	loggedWriteVideoRtpCount int
	loggedReadRtpCount       nazaatomic.Int32 // 因为音频和视频是两个连接，所以需要原子操作
	loggedReadRtcpCount      nazaatomic.Int32

	disposeOnce sync.Once
	waitChan    chan error
}

func NewBaseOutSession(uniqueKey string, cmdSession IInterleavedPacketWriter) *BaseOutSession {
	s := &BaseOutSession{
		uniqueKey:  uniqueKey,
		cmdSession: cmdSession,
		stat: base.StatSession{
			Protocol:  base.ProtocolRtsp,
			SessionId: uniqueKey,
			StartTime: base.ReadableNowTime(),
		},
		audioRtpChannel:  -1,
		videoRtpChannel:  -1,
		debugLogMaxCount: 3,
		waitChan:         make(chan error, 1),
	}
	Log.Infof("[%s] lifecycle new rtsp BaseOutSession. session=%p", uniqueKey, s)
	return s
}

func (session *BaseOutSession) InitWithSdp(sdpCtx sdp.LogicContext) {
	session.sdpCtx = sdpCtx
}

func (session *BaseOutSession) SetupWithConn(uri string, rtpConn, rtcpConn *nazanet.UdpConnection) error {
	if session.sdpCtx.IsAudioUri(uri) {
		session.audioRtpConn = rtpConn
		session.audioRtcpConn = rtcpConn
	} else if session.sdpCtx.IsVideoUri(uri) {
		session.videoRtpConn = rtpConn
		session.videoRtcpConn = rtcpConn
	} else {
		return nazaerrors.Wrap(base.ErrRtsp)
	}

	go rtpConn.RunLoop(session.onReadRtpPacket)
	go rtcpConn.RunLoop(session.onReadRtcpPacket)

	return nil
}

func (session *BaseOutSession) SetupWithChannel(uri string, rtpChannel, rtcpChannel int) error {
	if session.sdpCtx.IsAudioUri(uri) {
		session.audioRtpChannel = rtpChannel
		session.audioRtcpChannel = rtcpChannel
		return nil
	} else if session.sdpCtx.IsVideoUri(uri) {
		session.videoRtpChannel = rtpChannel
		session.videoRtcpChannel = rtcpChannel
		return nil
	}

	return nazaerrors.Wrap(base.ErrRtsp)
}

// ---------------------------------------------------------------------------------------------------------------------
// IClientSessionLifecycle interface
// ---------------------------------------------------------------------------------------------------------------------

// Dispose 文档请参考： IClientSessionLifecycle interface
//
func (session *BaseOutSession) Dispose() error {
	return session.dispose(nil)
}

// WaitChan 文档请参考： IClientSessionLifecycle interface
//
// 注意，目前只有一种情况，即上层主动调用Dispose函数，此时error为nil
//
func (session *BaseOutSession) WaitChan() <-chan error {
	return session.waitChan
}

// ---------------------------------------------------------------------------------------------------------------------

func (session *BaseOutSession) HandleInterleavedPacket(b []byte, channel int) {
	switch channel {
	case session.audioRtpChannel:
		fallthrough
	case session.videoRtpChannel:
		Log.Warnf("[%s] not supposed to read packet in rtp channel of BaseOutSession. channel=%d, len=%d", session.uniqueKey, channel, len(b))
	case session.audioRtcpChannel:
		fallthrough
	case session.videoRtcpChannel:
		Log.Debugf("[%s] read interleaved rtcp packet. b=%s", session.uniqueKey, hex.Dump(nazabytes.Prefix(b, 32)))
	default:
		Log.Errorf("[%s] read interleaved packet but channel invalid. channel=%d", session.uniqueKey, channel)
	}
}

func (session *BaseOutSession) WriteRtpPacket(packet rtprtcp.RtpPacket) error {
	var err error

	// 发送数据时，保证和sdp的原始类型对应
	t := int(packet.Header.PacketType)
	if session.sdpCtx.IsAudioPayloadTypeOrigin(t) {
		if session.loggedWriteAudioRtpCount < session.debugLogMaxCount {
			Log.Debugf("[%s] LOGPACKET. write audio rtp=%+v", session.uniqueKey, packet.Header)
			session.loggedWriteAudioRtpCount++
		}

		if session.audioRtpConn != nil {
			err = session.audioRtpConn.Write(packet.Raw)
		}
		if session.audioRtpChannel != -1 {
			err = session.cmdSession.WriteInterleavedPacket(packet.Raw, session.audioRtpChannel)
		}
	} else if session.sdpCtx.IsVideoPayloadTypeOrigin(t) {
		if session.loggedWriteVideoRtpCount < session.debugLogMaxCount {
			Log.Debugf("[%s] LOGPACKET. write video rtp=%+v", session.uniqueKey, packet.Header)
			session.loggedWriteVideoRtpCount++
		}

		if session.videoRtpConn != nil {
			err = session.videoRtpConn.Write(packet.Raw)
		}
		if session.videoRtpChannel != -1 {
			err = session.cmdSession.WriteInterleavedPacket(packet.Raw, session.videoRtpChannel)
		}
	} else {
		Log.Errorf("[%s] write rtp packet but type invalid. type=%d", session.uniqueKey, t)
		err = nazaerrors.Wrap(base.ErrRtsp)
	}

	if err == nil {
		session.currConnStat.WroteBytesSum.Add(uint64(len(packet.Raw)))
	}
	return err
}

func (session *BaseOutSession) GetStat() base.StatSession {
	session.stat.ReadBytesSum = session.currConnStat.ReadBytesSum.Load()
	session.stat.WroteBytesSum = session.currConnStat.WroteBytesSum.Load()
	return session.stat
}

func (session *BaseOutSession) UpdateStat(intervalSec uint32) {
	readBytesSum := session.currConnStat.ReadBytesSum.Load()
	wroteBytesSum := session.currConnStat.WroteBytesSum.Load()
	rDiff := readBytesSum - session.prevConnStat.ReadBytesSum
	session.stat.ReadBitrate = int(rDiff * 8 / 1024 / uint64(intervalSec))
	wDiff := wroteBytesSum - session.prevConnStat.WroteBytesSum
	session.stat.WriteBitrate = int(wDiff * 8 / 1024 / uint64(intervalSec))
	session.stat.Bitrate = session.stat.WriteBitrate
	session.prevConnStat.ReadBytesSum = readBytesSum
	session.prevConnStat.WroteBytesSum = wroteBytesSum
}

func (session *BaseOutSession) IsAlive() (readAlive, writeAlive bool) {
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

func (session *BaseOutSession) UniqueKey() string {
	return session.uniqueKey
}

func (session *BaseOutSession) onReadRtpPacket(b []byte, rAddr *net.UDPAddr, err error) bool {
	if session.loggedReadRtpCount.Load() < int32(session.debugLogMaxCount) {
		Log.Debugf("[%s] LOGPACKET. read rtp=%s", session.uniqueKey, hex.Dump(nazabytes.Prefix(b, 32)))
		session.loggedReadRtpCount.Increment()
	}
	return true
}

func (session *BaseOutSession) onReadRtcpPacket(b []byte, rAddr *net.UDPAddr, err error) bool {
	// TODO chef: impl me

	if session.loggedReadRtcpCount.Load() < int32(session.debugLogMaxCount) {
		Log.Debugf("[%s] LOGPACKET. read rtcp=%s", session.uniqueKey, hex.Dump(nazabytes.Prefix(b, 32)))
		session.loggedReadRtcpCount.Increment()
	}
	return true
}

func (session *BaseOutSession) dispose(err error) error {
	var retErr error
	session.disposeOnce.Do(func() {
		Log.Infof("[%s] lifecycle dispose rtsp BaseOutSession. session=%p", session.uniqueKey, session)
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
