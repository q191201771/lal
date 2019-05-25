package rtmp

import (
	"bufio"
	"github.com/q191201771/lal/log"
	"github.com/q191201771/lal/util"
	"net"
)

type ServerSession struct {
	conn      net.Conn
	rb        *bufio.Reader
	wb        *bufio.Writer
	hs        HandshakeServer
	composer  *Composer
	packer    *MessagePacker
	UniqueKey string
}

func NewServerSession(conn net.Conn) *ServerSession {
	return &ServerSession{
		conn:      conn,
		rb:        bufio.NewReaderSize(conn, readBufSize),
		wb:        bufio.NewWriterSize(conn, writeBufSize),
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
	return nil
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
	log.Debugf("%d %d %v", stream.header.msgTypeID, stream.msgLen, stream.header)
	switch stream.header.msgTypeID {
	case typeidSetChunkSize:
		// TODO chef:
	case typeidCommandMessageAMF0:
		return s.doCommandMessage(stream)

	}
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
	log.Debug(val)
	app, ok := val["app"].(string)
	if !ok {
		return rtmpErr
	}
	log.Infof("-----> connect('%s')", app)

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
	streamName, err := stream.msg.readStringWithType()
	if err != nil {
		return err
	}
	pubType, err := stream.msg.readStringWithType()
	if err != nil {
		return err
	}
	log.Debug(pubType)
	log.Infof("-----> publish('%s')", streamName)
	// TODO chef: hardcode streamID
	if err := s.packer.writeOnStatusPublish(s.conn, 1); err != nil {
		return err
	}
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
	return nil
}
