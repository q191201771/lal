package rtmp

import (
	"bufio"
	"net"
)

type ServerSession struct {
	conn net.Conn
	rb   *bufio.Reader
	wb   *bufio.Writer
	hs   HandshakeServer
}

func NewServerSession(conn net.Conn) *ServerSession {
	return &ServerSession{
		conn: conn,
		rb:   bufio.NewReaderSize(conn, readBufSize),
		wb:   bufio.NewWriterSize(conn, writeBufSize),
	}
}

func (s *ServerSession) RunLoop() error {
	if err := s.handshake(); err != nil {
		return err
	}
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
