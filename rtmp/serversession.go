package rtmp

import (
	"bufio"
	"encoding/hex"
	"github.com/q191201771/lal/log"
	"github.com/q191201771/lal/util"
	"net"
)

type ServerSessionObserver interface {
	NewRTMPPubSessionCB(session *ServerSession)
	NewRTMPSubSessionCB(session *ServerSession)
}

type AVMessageObserver interface {
	// @param t: 8 audio, 9 video, 18 meta
	// after cb, PullSession will use <message>
	ReadAVMessageCB(t int, timestampAbs int, message []byte)
}

type ServerSessionType int

const (
	ServerSessionTypeInit ServerSessionType = iota // 收到客户端的publish或者play信令之前的类型状态
	ServerSessionTypePub
	ServerSessionTypeSub
)

type ServerSession struct {
	AppName    string
	StreamName string
	UniqueKey  string

	obs      ServerSessionObserver
	avObs    AVMessageObserver
	conn     net.Conn
	rb       *bufio.Reader
	wb       *bufio.Writer
	t        ServerSessionType
	hs       HandshakeServer
	composer *Composer
	packer   *MessagePacker
}

func NewServerSession(obs ServerSessionObserver, conn net.Conn) *ServerSession {
	return &ServerSession{
		obs:       obs,
		conn:      conn,
		rb:        bufio.NewReaderSize(conn, readBufSize),
		wb:        bufio.NewWriterSize(conn, writeBufSize),
		t:         ServerSessionTypeInit,
		composer:  NewComposer(),
		packer:    NewMessagePacker(),
		UniqueKey: util.GenUniqueKey("RTMPSERVER"),
	}
}

func (s *ServerSession) RunLoop() error {
	if err := s.handshake(); err != nil {
		return err
	}
	return s.composer.RunLoop(s.rb, s.doMsg)
}

func (s *ServerSession) SetAVMessageObserver(obs AVMessageObserver) {
	s.avObs = obs
}

func (s *ServerSession) handshake() error {
	if err := s.hs.ReadC0C1(s.rb); err != nil {
		return err
	}
	if err := s.hs.WriteS0S1S2(s.conn); err != nil {
		return err
	}
	if err := s.hs.ReadC2(s.rb); err != nil {
		return err
	}
	return nil
}

func (s *ServerSession) doMsg(stream *Stream) error {
	//log.Debugf("%d %d %v", stream.header.msgTypeID, stream.msgLen, stream.header)
	switch stream.header.msgTypeID {
	case typeidSetChunkSize:
		// TODO chef:
	case typeidCommandMessageAMF0:
		return s.doCommandMessage(stream)
	case typeidDataMessageAMF0:
		return s.doDataMessageAMF0(stream)
	case typeidAudio:
		fallthrough
	case typeidVideo:
		s.avObs.ReadAVMessageCB(stream.header.msgTypeID, stream.timestampAbs, stream.msg.buf[stream.msg.b:stream.msg.e])

	}
	return nil
}

func (s *ServerSession) doDataMessageAMF0(stream *Stream) error {
	val, err := stream.msg.peekStringWithType()
	if err != nil {
		return err
	}

	switch val {
	case "|RtmpSampleAccess": // TODO chef: handle this?
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
			return rtmpErr
		}
	case "onMetaData":
		// noop
	default:
		// TODO chef:
		log.Error(val)
		log.Error(hex.Dump(stream.msg.buf[stream.msg.b:stream.msg.e]))
		return nil
	}

	s.avObs.ReadAVMessageCB(stream.header.msgTypeID, stream.timestampAbs, stream.msg.buf[stream.msg.b:stream.msg.e])
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
		log.Warnf("read command message %s,ignore it.", cmd)
	default:
		log.Errorf("unknown cmd. cmd=%s", cmd)
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
		return rtmpErr
	}
	log.Infof("-----> connect('%s')", s.AppName)

	if err := s.packer.writeWinAckSize(s.conn, windowAcknowledgementSize); err != nil {
		return err
	}
	if err := s.packer.writePeerBandwidth(s.conn, peerBandwidth, peerBandwidthLimitTypeDynamic); err != nil {
		return err
	}
	if err := s.packer.writeChunkSize(s.conn, localChunkSize); err != nil {
		return err
	}
	if err := s.packer.writeConnectResult(s.conn, tid); err != nil {
		return err
	}
	return nil
}

func (s *ServerSession) doCreateStream(tid int, stream *Stream) error {
	log.Info("-----> createStream()")
	if err := s.packer.writeCreateStreamResult(s.conn, tid); err != nil {
		return err
	}
	return nil
}

func (s *ServerSession) doPublish(tid int, stream *Stream) error {
	if err := stream.msg.readNull(); err != nil {
		return err
	}
	var err error
	s.StreamName, err = stream.msg.readStringWithType()
	if err != nil {
		return err
	}
	pubType, err := stream.msg.readStringWithType()
	if err != nil {
		return err
	}
	log.Debug(pubType)
	log.Infof("-----> publish('%s')", s.StreamName)
	// TODO chef: hardcode streamID
	if err := s.packer.writeOnStatusPublish(s.conn, 1); err != nil {
		return err
	}
	s.t = ServerSessionTypePub
	s.obs.NewRTMPPubSessionCB(s)
	return nil
}

func (s *ServerSession) doPlay(tid int, stream *Stream) error {
	if err := stream.msg.readNull(); err != nil {
		return err
	}
	streamName, err := stream.msg.readStringWithType()
	if err != nil {
		return err
	}
	log.Infof("-----> play('%s')", streamName)
	// TODO chef: start duration reset

	if err := s.packer.writeOnStatusPublish(s.conn, 1); err != nil {
		return err
	}
	s.t = ServerSessionTypeSub
	s.obs.NewRTMPSubSessionCB(s)
	return nil
}
