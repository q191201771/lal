package rtmp

import (
	"bufio"
	"bytes"
	"github.com/q191201771/lal/bele"
	"github.com/q191201771/lal/log"
	"io"
	"net"
	"net/url"
	"strings"
)

var readBufSize = 4096
var writeBufSize = 4096

var chunkSize = 4096

type PullSession struct {
	Conn           net.Conn
	rb             *bufio.Reader
	ab             *bytes.Buffer
	wb             *bufio.Writer
	url            *url.URL
	tcUrl          string
	appName        string
	streamName     string
	hs             HandShakeClient
	packer         *MessagePacker
	csid2stream    map[int]*Stream
	peerChunkSize  int
	peerWinAckSize int
}

func NewPullSession() *PullSession {
	return &PullSession{
		ab:            &bytes.Buffer{},
		packer:        NewMessagePacker(),
		csid2stream:   make(map[int]*Stream),
		peerChunkSize: defaultChunkSize,
	}
}

func (s *PullSession) Pull(rawurl string) error {
	if err := s.parseUrl(rawurl); err != nil {
		return err
	}
	if err := s.tcpConnect(); err != nil {
		return err
	}

	if err := s.handshake(); err != nil {
		return err
	}
	if err := s.packer.writeChunkSize(s.Conn, chunkSize); err != nil {
		return err
	}
	if err := s.packer.writeConect(s.Conn, s.appName, s.tcUrl); err != nil {
		return err
	}

	// TODO chef:
	go func() {
		s.runReadLoop()
	}()

	return nil
}

func (s *PullSession) runReadLoop() error {
	bootstrap := make([]byte, 11)

	for {
		if _, err := io.ReadAtLeast(s.rb, bootstrap[:1], 1); err != nil {
			return err
		}

		// 5.3.1.1. Chunk Basic Header
		fmt := (bootstrap[0] >> 6) & 0x03
		csid := int(bootstrap[0] & 0x3f)

		switch csid {
		case 0:
			if _, err := io.ReadAtLeast(s.rb, bootstrap[:1], 1); err != nil {
				return err
			}
			csid = 64 + int(bootstrap[0])
		case 1:
			if _, err := io.ReadAtLeast(s.rb, bootstrap[:2], 2); err != nil {
				return err
			}
			csid = 64 + int(bootstrap[0]) + int(bootstrap[1])*256
		}

		stream := s.getOrCreateStream(csid)

		// 5.3.1.2. Chunk Message Header
		switch fmt {
		case 0:
			if _, err := io.ReadAtLeast(s.rb, bootstrap[:11], 11); err != nil {
				return err
			}
			stream.timestampAbs = int(bele.BeUint24(bootstrap))
			stream.msgLen = int(bele.BeUint24(bootstrap[3:]))
			stream.header.msgTypeId = int(uint32(bootstrap[6]))
			stream.header.msgStreamId = int(bele.LeUint32(bootstrap[7:]))
		case 1:
			if _, err := io.ReadAtLeast(s.rb, bootstrap[:7], 7); err != nil {
				return err
			}
			stream.timestampAbs += int(bele.BeUint24(bootstrap))
			stream.msgLen = int(bele.BeUint24(bootstrap[3:]))
			stream.header.msgTypeId = int((bootstrap[6]))
		case 2:
			if _, err := io.ReadAtLeast(s.rb, bootstrap[:4], 4); err != nil {
				return err
			}
			stream.timestampAbs += int(bele.BeUint24(bootstrap))
		case 3:
			// noop
		}

		// 5.3.1.3 Extended Timestamp
		if stream.header.timestamp == maxTimestampInMessageHeader {
			if _, err := io.ReadAtLeast(s.rb, bootstrap[:4], 4); err != nil {
				return err
			}
			stream.header.timestamp = int(bele.BeUint32(bootstrap))
			switch fmt {
			case 0:
				stream.timestampAbs = stream.header.timestamp
			case 1:
				fallthrough
			case 2:
				stream.timestampAbs += stream.header.timestamp
			case 3:
				// noop
			}
		}

		var neededSize int
		if stream.msgLen <= s.peerChunkSize {
			neededSize = stream.msgLen
		} else {
			neededSize := stream.msgLen - stream.msg.len()
			if neededSize > s.peerChunkSize {
				neededSize = s.peerChunkSize
			}
		}

		stream.msg.reserve(neededSize)
		if _, err := io.ReadAtLeast(s.rb, stream.msg.buf[stream.msg.e:neededSize], neededSize); err != nil {
			return err
		}
		stream.msg.produced(neededSize)

		if stream.msg.len() == stream.msgLen {
			log.Infof("%+v", stream)
			s.doMsg(stream)
			stream.msg.clear()
		}
	}
}

func (s *PullSession) doMsg(stream *Stream) error {
	switch stream.header.msgTypeId {
	case typeidWinAckSize:
		fallthrough
	case typeidBandwidth:
		fallthrough
	case typeidSetChunkSize:
		return s.doProtocolControlMessage(stream)
	case typeidCommandMessageAMF0:
		return s.doCommandMessage(stream)
	}
	return nil
}

func (s *PullSession) doCommandMessage(stream *Stream) error {
	log.Info("to be continued.")
	return nil
}

func (s *PullSession) doProtocolControlMessage(stream *Stream) error {
	if stream.msg.len() < 4 {
		return rtmpErr
	}
	val := int(bele.BeUint32(stream.msg.buf))

	switch stream.header.msgTypeId {
	case typeidWinAckSize:
		s.peerWinAckSize = val
		log.Infof("-----> Window Acknowledgement Size: %d", s.peerWinAckSize)
	case typeidBandwidth:
		log.Infof("-----> Set Peer Bandwidth, ignore.")
	case typeidSetChunkSize:
		s.peerChunkSize = val
		log.Infof("-----> Set Chunk Size %d", s.peerChunkSize)
	}
	return nil
}

func (s *PullSession) getOrCreateStream(csid int) *Stream {
	stream, exist := s.csid2stream[csid]
	if !exist {
		stream = NewStream()
		s.csid2stream[csid] = stream
	}
	return stream
}

func (s *PullSession) parseUrl(rawurl string) error {
	var err error
	s.url, err = url.Parse(rawurl)
	if err != nil {
		return err
	}
	if s.url.Scheme != "rtmp" || len(s.url.Host) == 0 || len(s.url.Path) == 0 || s.url.Path[0] != '/' {
		return rtmpErr
	}
	index := strings.LastIndexByte(rawurl, '/')
	if index == -1 {
		return rtmpErr
	}
	s.tcUrl = rawurl[:index]
	strs := strings.Split(s.url.Path[1:], "/")
	if len(strs) != 2 {
		return rtmpErr
	}
	s.appName = strs[0]
	s.streamName = strs[1]
	log.Debugf("%s %s %s %+v", s.tcUrl, s.appName, s.streamName, *s.url)

	return nil
}

func (s *PullSession) tcpConnect() error {
	var err error
	var addr string
	if strings.Contains(s.url.Host, ":") {
		addr = s.url.Host
	} else {
		addr = s.url.Host + ":1935"
	}

	if s.Conn, err = net.Dial("tcp", addr); err != nil {
		return err
	}
	s.rb = bufio.NewReaderSize(s.Conn, readBufSize)
	s.wb = bufio.NewWriterSize(s.Conn, writeBufSize)
	return nil
}

func (s *PullSession) handshake() error {
	if err := s.hs.writeC0C1(s.Conn); err != nil {
		return err
	}
	if err := s.hs.readS0S1S2(s.rb); err != nil {
		return err
	}

	if err := s.hs.writeC2(s.Conn); err != nil {
		return err
	}
	return nil
}
