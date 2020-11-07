// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import (
	"net"
	"strings"
	"time"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"
)

// TODO chef: 没有进化成Pub Sub时的超时释放

type ServerSessionObserver interface {
	OnRTMPConnect(session *ServerSession, opa ObjectPairArray)
	OnNewRTMPPubSession(session *ServerSession) // 上层代码应该在这个事件回调中注册音视频数据的监听
	OnNewRTMPSubSession(session *ServerSession)
}

var _ ServerSessionObserver = &Server{}

type PubSessionObserver interface {
	// 注意，回调结束后，内部会复用Payload内存块
	OnReadRTMPAVMsg(msg base.RTMPMsg)
}

func (s *ServerSession) SetPubSessionObserver(observer PubSessionObserver) {
	s.avObserver = observer
}

type ServerSessionType int

const (
	ServerSessionTypeUnknown ServerSessionType = iota // 收到客户端的publish或者play信令之前的类型状态
	ServerSessionTypePub
	ServerSessionTypeSub
)

type ServerSession struct {
	UniqueKey              string // const after init
	AppName                string // const after init
	StreamName             string // const after set
	RawQuery               string //const after set
	StreamNameWithRawQuery string // const after set

	observer      ServerSessionObserver
	t             ServerSessionType
	hs            HandshakeServer
	chunkComposer *ChunkComposer
	packer        *MessagePacker

	conn         connection.Connection
	prevConnStat connection.Stat
	staleStat    *connection.Stat
	stat         base.StatSession

	// only for PubSession
	avObserver PubSessionObserver

	// only for SubSession
	IsFresh bool
}

func NewServerSession(observer ServerSessionObserver, conn net.Conn) *ServerSession {
	uk := unique.GenUniqueKey("RTMPPUBSUB")
	s := &ServerSession{
		conn: connection.New(conn, func(option *connection.Option) {
			option.ReadBufSize = readBufSize
		}),
		stat: base.StatSession{
			Protocol:   base.ProtocolRTMP,
			StartTime:  time.Now().Format("2006-01-02 15:04:05.999"),
			RemoteAddr: conn.RemoteAddr().String(),
		},
		UniqueKey:     uk,
		observer:      observer,
		t:             ServerSessionTypeUnknown,
		chunkComposer: NewChunkComposer(),
		packer:        NewMessagePacker(),
		IsFresh:       true,
	}
	nazalog.Infof("[%s] lifecycle new rtmp ServerSession. session=%p, remote addr=%s", uk, s, conn.RemoteAddr().String())
	return s
}

func (s *ServerSession) RunLoop() (err error) {
	if err = s.handshake(); err != nil {
		return err
	}

	return s.runReadLoop()
}

func (s *ServerSession) AsyncWrite(msg []byte) error {
	_, err := s.conn.Write(msg)
	return err
}

func (s *ServerSession) Flush() error {
	return s.conn.Flush()
}

func (s *ServerSession) Dispose() {
	nazalog.Infof("[%s] lifecycle dispose rtmp ServerSession.", s.UniqueKey)
	_ = s.conn.Close()
}

func (s *ServerSession) GetStat() base.StatSession {
	connStat := s.conn.GetStat()
	s.stat.ReadBytesSum = connStat.ReadBytesSum
	s.stat.WroteBytesSum = connStat.WroteBytesSum
	return s.stat
}

func (s *ServerSession) UpdateStat(interval uint32) {
	currStat := s.conn.GetStat()
	switch s.t {
	case ServerSessionTypePub:
		diff := currStat.ReadBytesSum - s.prevConnStat.ReadBytesSum
		s.stat.Bitrate = int(diff * 8 / 1024 / uint64(interval))
	case ServerSessionTypeSub:
		diff := currStat.WroteBytesSum - s.prevConnStat.WroteBytesSum
		s.stat.Bitrate = int(diff * 8 / 1024 / uint64(interval))
	}
	s.prevConnStat = currStat
}

func (s *ServerSession) IsAlive(interval uint32) (ret bool) {
	currStat := s.conn.GetStat()
	if s.staleStat == nil {
		s.staleStat = new(connection.Stat)
		*s.staleStat = currStat
		return true
	}

	switch s.t {
	case ServerSessionTypePub:
		ret = !(currStat.ReadBytesSum-s.staleStat.ReadBytesSum == 0)
	case ServerSessionTypeSub:
		ret = !(currStat.WroteBytesSum-s.staleStat.WroteBytesSum == 0)
	}
	*s.staleStat = currStat
	return ret
}

func (s *ServerSession) RemoteAddr() string {
	return s.conn.RemoteAddr().String()
}

func (s *ServerSession) runReadLoop() error {
	return s.chunkComposer.RunLoop(s.conn, s.doMsg)
}

func (s *ServerSession) handshake() error {
	if err := s.hs.ReadC0C1(s.conn); err != nil {
		return err
	}
	nazalog.Infof("[%s] < R Handshake C0+C1.", s.UniqueKey)

	nazalog.Infof("[%s] > W Handshake S0+S1+S2.", s.UniqueKey)
	if err := s.hs.WriteS0S1S2(s.conn); err != nil {
		return err
	}

	if err := s.hs.ReadC2(s.conn); err != nil {
		return err
	}
	nazalog.Infof("[%s] < R Handshake C2.", s.UniqueKey)
	return nil
}

func (s *ServerSession) doMsg(stream *Stream) error {
	//log.Debugf("%d %d %v", stream.header.msgTypeID, stream.msgLen, stream.header)
	switch stream.header.MsgTypeID {
	case base.RTMPTypeIDSetChunkSize:
		// noop
		// 因为底层的 chunk composer 已经处理过了，这里就不用处理
	case base.RTMPTypeIDCommandMessageAMF0:
		return s.doCommandMessage(stream)
	case base.RTMPTypeIDCommandMessageAMF3:
		return s.doCommandAFM3Message(stream)
	case base.RTMPTypeIDMetadata:
		return s.doDataMessageAMF0(stream)
	case base.RTMPTypeIDAck:
		return s.doACK(stream)
	case base.RTMPTypeIDAudio:
		fallthrough
	case base.RTMPTypeIDVideo:
		if s.t != ServerSessionTypePub {
			nazalog.Errorf("[%s] read audio/video message but server session not pub type.", s.UniqueKey)
			return ErrRTMP
		}
		s.avObserver.OnReadRTMPAVMsg(stream.toAVMsg())
	default:
		nazalog.Warnf("[%s] read unknown message. typeid=%d, %s", s.UniqueKey, stream.header.MsgTypeID, stream.toDebugString())

	}
	return nil
}

func (s *ServerSession) doACK(stream *Stream) error {
	seqNum := bele.BEUint32(stream.msg.buf[stream.msg.b:stream.msg.e])
	nazalog.Infof("[%s] < R Acknowledgement. ignore. sequence number=%d.", s.UniqueKey, seqNum)
	return nil
}

func (s *ServerSession) doDataMessageAMF0(stream *Stream) error {
	if s.t != ServerSessionTypePub {
		nazalog.Errorf("[%s] read audio/video message but server session not pub type.", s.UniqueKey)
		return ErrRTMP
	}

	val, err := stream.msg.peekStringWithType()
	if err != nil {
		return err
	}

	switch val {
	case "|RtmpSampleAccess":
		nazalog.Warnf("[%s] read data message, ignore it. val=%s", s.UniqueKey, val)
		return nil
	case "@setDataFrame":
		// macos obs
		// skip @setDataFrame
		val, err = stream.msg.readStringWithType()
		val, err := stream.msg.peekStringWithType()
		if err != nil {
			return err
		}
		if val != "onMetaData" {
			nazalog.Errorf("[%s] read unknown data message. val=%s, %s", s.UniqueKey, val, stream.toDebugString())
			return ErrRTMP
		}
	case "onMetaData":
		// noop
	default:
		nazalog.Errorf("[%s] read unknown data message. val=%s, %s", s.UniqueKey, val, stream.toDebugString())
		return nil
	}

	s.avObserver.OnReadRTMPAVMsg(stream.toAVMsg())
	return nil
}

func (s *ServerSession) doCommandMessage(stream *Stream) error {
	cmd, err := stream.msg.readStringWithType()
	if err != nil {
		return err
	}
	tid, err := stream.msg.readNumberWithType()
	if err != nil {
		return err
	}

	switch cmd {
	case "connect":
		return s.doConnect(tid, stream)
	case "createStream":
		return s.doCreateStream(tid, stream)
	case "publish":
		return s.doPublish(tid, stream)
	case "play":
		return s.doPlay(tid, stream)
	case "releaseStream":
		fallthrough
	case "FCPublish":
		fallthrough
	case "FCUnpublish":
		fallthrough
	case "getStreamLength":
		fallthrough
	case "deleteStream":
		nazalog.Debugf("[%s] read command message, ignore it. cmd=%s, %s", s.UniqueKey, cmd, stream.toDebugString())
	default:
		nazalog.Errorf("[%s] read unknown command message. cmd=%s, %s", s.UniqueKey, cmd, stream.toDebugString())
	}
	return nil
}

func (s *ServerSession) doCommandAFM3Message(stream *Stream) error {
	//去除前面的0就是AMF0的数据
	stream.msg.consumed(1)
	return s.doCommandMessage(stream)
}

func (s *ServerSession) doConnect(tid int, stream *Stream) error {
	val, err := stream.msg.readObjectWithType()
	if err != nil {
		return err
	}
	s.AppName, err = val.FindString("app")
	if err != nil {
		return err
	}
	nazalog.Infof("[%s] < R connect('%s').", s.UniqueKey, s.AppName)

	s.observer.OnRTMPConnect(s, val)

	nazalog.Infof("[%s] > W Window Acknowledgement Size %d.", s.UniqueKey, windowAcknowledgementSize)
	if err := s.packer.writeWinAckSize(s.conn, windowAcknowledgementSize); err != nil {
		return err
	}

	nazalog.Infof("[%s] > W Set Peer Bandwidth.", s.UniqueKey)
	if err := s.packer.writePeerBandwidth(s.conn, peerBandwidth, peerBandwidthLimitTypeDynamic); err != nil {
		return err
	}

	nazalog.Infof("[%s] > W SetChunkSize %d.", s.UniqueKey, LocalChunkSize)
	if err := s.packer.writeChunkSize(s.conn, LocalChunkSize); err != nil {
		return err
	}

	nazalog.Infof("[%s] > W _result('NetConnection.Connect.Success').", s.UniqueKey)
	oe, err := val.FindNumber("objectEncoding")
	if oe != 0 && oe != 3 {
		oe = 0
	}
	if err := s.packer.writeConnectResult(s.conn, tid, oe); err != nil {
		return err
	}
	return nil
}

func (s *ServerSession) doCreateStream(tid int, stream *Stream) error {
	nazalog.Infof("[%s] < R createStream().", s.UniqueKey)
	nazalog.Infof("[%s] > W _result().", s.UniqueKey)
	if err := s.packer.writeCreateStreamResult(s.conn, tid); err != nil {
		return err
	}
	return nil
}

func (s *ServerSession) doPublish(tid int, stream *Stream) (err error) {
	if err = stream.msg.readNull(); err != nil {
		return err
	}
	s.StreamNameWithRawQuery, err = stream.msg.readStringWithType()
	if err != nil {
		return err
	}
	ss := strings.Split(s.StreamNameWithRawQuery, "?")
	s.StreamName = ss[0]
	if len(ss) == 2 {
		s.RawQuery = ss[1]
	}

	pubType, err := stream.msg.readStringWithType()
	if err != nil {
		return err
	}
	nazalog.Debugf("[%s] pubType=%s", s.UniqueKey, pubType)
	nazalog.Infof("[%s] < R publish('%s')", s.UniqueKey, s.StreamNameWithRawQuery)

	nazalog.Infof("[%s] > W onStatus('NetStream.Publish.Start').", s.UniqueKey)
	if err := s.packer.writeOnStatusPublish(s.conn, MSID1); err != nil {
		return err
	}

	// 回复完信令后修改 connection 的属性
	s.ModConnProps()

	s.t = ServerSessionTypePub
	s.observer.OnNewRTMPPubSession(s)

	return nil
}

func (s *ServerSession) doPlay(tid int, stream *Stream) (err error) {
	if err = stream.msg.readNull(); err != nil {
		return err
	}
	s.StreamNameWithRawQuery, err = stream.msg.readStringWithType()
	if err != nil {
		return err
	}
	ss := strings.Split(s.StreamNameWithRawQuery, "?")
	s.StreamName = ss[0]
	if len(ss) == 2 {
		s.RawQuery = ss[1]
	}

	nazalog.Infof("[%s] < R play('%s').", s.UniqueKey, s.StreamNameWithRawQuery)
	// TODO chef: start duration reset

	if err := s.packer.writeStreamIsRecorded(s.conn, MSID1); err != nil {
		return err
	}
	if err := s.packer.writeStreamBegin(s.conn, MSID1); err != nil {
		return err
	}

	nazalog.Infof("[%s] > W onStatus('NetStream.Play.Start').", s.UniqueKey)
	if err := s.packer.writeOnStatusPlay(s.conn, MSID1); err != nil {
		return err
	}

	// 回复完信令后修改 connection 的属性
	s.ModConnProps()

	s.t = ServerSessionTypeSub
	s.observer.OnNewRTMPSubSession(s)

	return nil
}

func (s *ServerSession) ModConnProps() {
	s.conn.ModWriteChanSize(wChanSize)
	// TODO chef:
	// 使用合并发送
	// naza.connection 这种方式会导致最后一点数据发送不出去，我们应该使用更好的方式，比如合并发送模式下，Dispose时发送剩余数据
	//
	//s.conn.ModWriteBufSize(writeBufSize)

	switch s.t {
	case ServerSessionTypePub:
		s.conn.ModReadTimeoutMS(serverSessionReadAVTimeoutMS)
	case ServerSessionTypeSub:
		s.conn.ModWriteTimeoutMS(serverSessionWriteAVTimeoutMS)
	}
}
