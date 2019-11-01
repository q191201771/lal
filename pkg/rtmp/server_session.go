// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import (
	"encoding/hex"
	"net"
	"strings"

	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/connection"
	log "github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"
)

// TODO chef: 没有进化成Pub Sub时的超时释放

type ServerSessionObserver interface {
	NewRTMPPubSessionCB(session *ServerSession) // 上层代码应该在这个事件回调中注册音视频数据的监听
	NewRTMPSubSessionCB(session *ServerSession)
}

var _ ServerSessionObserver = &Server{}

type PubSessionObserver interface {
	AVMsgObserver
}

func (s *ServerSession) SetPubSessionObserver(obs PubSessionObserver) {
	s.avObs = obs
}

type ServerSessionType int

const (
	ServerSessionTypeUnknown ServerSessionType = iota // 收到客户端的publish或者play信令之前的类型状态
	ServerSessionTypePub
	ServerSessionTypeSub
)

type ServerSession struct {
	AppName                string
	StreamName             string
	StreamNameWithRawQuery string
	UniqueKey              string

	obs           ServerSessionObserver
	t             ServerSessionType
	hs            HandshakeServer
	chunkComposer *ChunkComposer
	packer        *MessagePacker

	conn connection.Connection

	// only for PubSession
	avObs PubSessionObserver

	// only for SubSession
	IsFresh     bool
	WaitKeyNalu bool
}

func NewServerSession(obs ServerSessionObserver, conn net.Conn) *ServerSession {
	uk := unique.GenUniqueKey("RTMPPUBSUB")
	log.Infof("lifecycle new rtmp server session. [%s]", uk)
	return &ServerSession{
		conn: connection.New(conn, func(option *connection.Option) {
			option.ReadBufSize = readBufSize
		}),
		UniqueKey:     uk,
		obs:           obs,
		t:             ServerSessionTypeUnknown,
		chunkComposer: NewChunkComposer(),
		packer:        NewMessagePacker(),
		IsFresh:       true,
		WaitKeyNalu:   true,
	}
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
	log.Infof("lifecycle dispose rtmp server session. [%s]", s.UniqueKey)
	_ = s.conn.Close()
}

func (s *ServerSession) runReadLoop() error {
	return s.chunkComposer.RunLoop(s.conn, s.doMsg)
}

func (s *ServerSession) handshake() error {
	if err := s.hs.ReadC0C1(s.conn); err != nil {
		return err
	}
	log.Infof("-----> Handshake C0+C1. [%s]", s.UniqueKey)

	log.Infof("<----- Handshake S0+S1+S2. [%s]", s.UniqueKey)
	if err := s.hs.WriteS0S1S2(s.conn); err != nil {
		return err
	}

	if err := s.hs.ReadC2(s.conn); err != nil {
		return err
	}
	log.Infof("-----> Handshake C2. [%s]", s.UniqueKey)
	return nil
}

func (s *ServerSession) doMsg(stream *Stream) error {
	//log.Debugf("%d %d %v", stream.header.msgTypeID, stream.msgLen, stream.header)
	switch stream.header.MsgTypeID {
	case typeidSetChunkSize:
		// noop
		// 因为底层的 chunk composer 已经处理过了，这里就不用处理
	case typeidCommandMessageAMF0:
		return s.doCommandMessage(stream)
	case TypeidDataMessageAMF0:
		return s.doDataMessageAMF0(stream)
	case typeidAck:
		return s.doACK(stream)
	case TypeidAudio:
		fallthrough
	case TypeidVideo:
		if s.t != ServerSessionTypePub {
			log.Errorf("read audio/video message but server session not pub type. [%s]", s.UniqueKey)
			return ErrRTMP
		}
		//log.Infof("t:%d ts:%d len:%d", stream.header.MsgTypeID, stream.header.timestampAbs, stream.msg.e - stream.msg.b)
		s.avObs.OnReadRTMPAVMsg(stream.toAVMsg())
	default:
		log.Warnf("unknown message. [%s] typeid=%d", s.UniqueKey, stream.header.MsgTypeID)

	}
	return nil
}

func (s *ServerSession) doACK(stream *Stream) error {
	seqNum := bele.BEUint32(stream.msg.buf[stream.msg.b:stream.msg.e])
	log.Infof("-----> Acknowledgement. [%s] ignore. sequence number=%d.", s.UniqueKey, seqNum)
	return nil
}

func (s *ServerSession) doDataMessageAMF0(stream *Stream) error {
	if s.t != ServerSessionTypePub {
		log.Errorf("read audio/video message but server session not pub type. [%s]", s.UniqueKey)
		return ErrRTMP
	}

	val, err := stream.msg.peekStringWithType()
	if err != nil {
		return err
	}

	switch val {
	case "|RtmpSampleAccess":
		log.Warn("recv |RtmpSampleAccess. ignore it.")
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
			return ErrRTMP
		}
	case "onMetaData":
		// noop
	default:
		log.Errorf("recv unknown message. val=%s, hex=%s", val, hex.Dump(stream.msg.buf[stream.msg.b:stream.msg.e]))
		return nil
	}

	s.avObs.OnReadRTMPAVMsg(stream.toAVMsg())
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
		log.Warnf("read command message,ignore it. [%s] %s", s.UniqueKey, cmd)
	default:
		log.Errorf("unknown cmd. [%s] cmd=%s", s.UniqueKey, cmd)
	}
	return nil
}

func (s *ServerSession) doConnect(tid int, stream *Stream) error {
	val, err := stream.msg.readObjectWithType()
	if err != nil {
		return err
	}
	var ok bool
	s.AppName, ok = val["app"].(string)
	if !ok {
		return ErrRTMP
	}
	log.Infof("-----> connect('%s'). [%s]", s.AppName, s.UniqueKey)

	log.Infof("<----- Window Acknowledgement Size %d. [%s]", windowAcknowledgementSize, s.UniqueKey)
	if err := s.packer.writeWinAckSize(s.conn, windowAcknowledgementSize); err != nil {
		return err
	}

	log.Infof("<----- Set Peer Bandwidth. [%s]", s.UniqueKey)
	if err := s.packer.writePeerBandwidth(s.conn, peerBandwidth, peerBandwidthLimitTypeDynamic); err != nil {
		return err
	}

	log.Infof("<----- SetChunkSize %d. [%s]", LocalChunkSize, s.UniqueKey)
	if err := s.packer.writeChunkSize(s.conn, LocalChunkSize); err != nil {
		return err
	}

	log.Infof("<---- _result('NetConnection.Connect.Success'). [%s]", s.UniqueKey)
	if err := s.packer.writeConnectResult(s.conn, tid); err != nil {
		return err
	}
	return nil
}

func (s *ServerSession) doCreateStream(tid int, stream *Stream) error {
	log.Infof("-----> createStream(). [%s]", s.UniqueKey)
	log.Infof("<---- _result(). [%s]", s.UniqueKey)
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

	pubType, err := stream.msg.readStringWithType()
	if err != nil {
		return err
	}
	log.Debugf("[%s] pubType=%s", s.UniqueKey, pubType)
	log.Infof("-----> publish('%s') [%s]", s.StreamName, s.UniqueKey)

	log.Infof("<---- onStatus('NetStream.Publish.Start'). [%s]", s.UniqueKey)
	if err := s.packer.writeOnStatusPublish(s.conn, MSID1); err != nil {
		return err
	}

	// 回复完信令后修改 connection 的属性
	s.ModConnProps()

	s.t = ServerSessionTypePub
	s.obs.NewRTMPPubSessionCB(s)

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

	log.Infof("-----> play('%s'). [%s]", s.StreamName, s.UniqueKey)
	// TODO chef: start duration reset

	log.Infof("<----onStatus('NetStream.Play.Start'). [%s]", s.UniqueKey)
	if err := s.packer.writeOnStatusPlay(s.conn, MSID1); err != nil {
		return err
	}

	// 回复完信令后修改 connection 的属性
	s.ModConnProps()

	s.t = ServerSessionTypeSub
	s.obs.NewRTMPSubSessionCB(s)

	return nil
}

func (s *ServerSession) ModConnProps() {
	s.conn.ModWriteChanSize(wChanSize)
	// TODO chef: naza.connection 这种方式会导致最后一点数据发送不出去，我们应该使用更好的方式
	//s.conn.ModWriteBufSize(writeBufSize)

	switch s.t {
	case ServerSessionTypePub:
		s.conn.ModReadTimeoutMS(serverSessionReadAVTimeoutMS)
	case ServerSessionTypeSub:
		s.conn.ModWriteTimeoutMS(serverSessionWriteAVTimeoutMS)
	}
}
