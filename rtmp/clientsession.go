package rtmp

import (
	"bufio"
	"github.com/q191201771/lal/bele"
	"github.com/q191201771/lal/log"
	"io"
	"net"
	"net/url"
	"strings"
)

var readBufSize = 4096
var writeBufSize = 4096

type ClientSession struct {
	t              ClientSessionType
	obs            PullSessionObserver
	doResultChan   chan error
	packer         *MessagePacker
	csid2stream    map[int]*Stream
	peerChunkSize  int
	url            *url.URL
	tcURL          string
	appName        string
	streamName     string
	hs             HandshakeClient
	Conn           net.Conn
	rb             *bufio.Reader
	wb             *bufio.Writer
	peerWinAckSize int
}

type ClientSessionType int

const (
	CSTPullSession ClientSessionType = 1
	CSTPushSession ClientSessionType = 2
)

// set <obs> if <t> equal CSTPullSession
func NewClientSession(t ClientSessionType, obs PullSessionObserver) *ClientSession {
	return &ClientSession{
		t:             t,
		obs:           obs,
		doResultChan:  make(chan error),
		packer:        NewMessagePacker(),
		csid2stream:   make(map[int]*Stream),
		peerChunkSize: defaultChunkSize,
	}
}

// block until server reply play start or other error occur.
func (s *ClientSession) Do(rawURL string) error {
	if err := s.parseURL(rawURL); err != nil {
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
	if err := s.packer.writeConnect(s.Conn, s.appName, s.tcURL); err != nil {
		return err
	}

	go func() {
		err := s.runReadLoop()
		s.doResultChan <- err
	}()

	doResult := <-s.doResultChan

	return doResult
}

func (s *ClientSession) runReadLoop() error {
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
		default:
			// noop
		}

		stream := s.getOrCreateStream(csid)

		// 5.3.1.2. Chunk Message Header
		switch fmt {
		case 0:
			if _, err := io.ReadAtLeast(s.rb, bootstrap[:11], 11); err != nil {
				return err
			}
			stream.header.timestamp = int(bele.BEUint24(bootstrap))
			stream.timestampAbs = stream.header.timestamp
			stream.msgLen = int(bele.BEUint24(bootstrap[3:]))
			stream.header.msgTypeID = int(uint32(bootstrap[6]))
			stream.header.msgStreamID = int(bele.LEUint32(bootstrap[7:]))

			stream.msg.reserve(stream.msgLen)
		case 1:
			if _, err := io.ReadAtLeast(s.rb, bootstrap[:7], 7); err != nil {
				return err
			}
			stream.header.timestamp = int(bele.BEUint24(bootstrap))
			stream.timestampAbs += stream.header.timestamp
			stream.msgLen = int(bele.BEUint24(bootstrap[3:]))
			stream.header.msgTypeID = int((bootstrap[6]))
		case 2:
			if _, err := io.ReadAtLeast(s.rb, bootstrap[:4], 4); err != nil {
				return err
			}
			stream.header.timestamp = int(bele.BEUint24(bootstrap))
			stream.timestampAbs += stream.header.timestamp
		case 3:
			// noop
		}

		// 5.3.1.3 Extended Timestamp
		if stream.header.timestamp == maxTimestampInMessageHeader {
			if _, err := io.ReadAtLeast(s.rb, bootstrap[:4], 4); err != nil {
				return err
			}
			stream.header.timestamp = int(bele.BEUint32(bootstrap))
			switch fmt {
			case 0:
				stream.timestampAbs = stream.header.timestamp
			case 1:
				fallthrough
			case 2:
				stream.timestampAbs = stream.timestampAbs - maxTimestampInMessageHeader + stream.header.timestamp
			case 3:
				// noop
			}
		}

		var neededSize int
		if stream.msgLen <= s.peerChunkSize {
			neededSize = stream.msgLen
		} else {
			neededSize = stream.msgLen - stream.msg.len()
			if neededSize > s.peerChunkSize {
				neededSize = s.peerChunkSize
			}
		}

		//stream.msg.reserve(neededSize)
		if _, err := io.ReadAtLeast(s.rb, stream.msg.buf[stream.msg.e:stream.msg.e+neededSize], neededSize); err != nil {
			return err
		}
		stream.msg.produced(neededSize)

		if stream.msg.len() == stream.msgLen {
			//log.Debugf("%+v", stream)
			if err := s.doMsg(stream); err != nil {
				return err
			}
			stream.msg.clear()
		}
	}
}

func (s *ClientSession) doMsg(stream *Stream) error {
	switch stream.header.msgTypeID {
	case typeidWinAckSize:
		fallthrough
	case typeidBandwidth:
		fallthrough
	case typeidSetChunkSize:
		return s.doProtocolControlMessage(stream)
	case typeidCommandMessageAMF0:
		return s.doCommandMessage(stream)
	case typeidUserControl:
		log.Warn("user control message. ignore.")
	case typeidDataMessageAMF0:
		fallthrough
	case typeidAudio:
		fallthrough
	case typeidVideo:
		s.obs.ReadAvMessageCB(stream.header.msgTypeID, stream.timestampAbs, stream.msg.buf[stream.msg.b:stream.msg.e])
	default:
		log.Errorf("unknown msg type id. typeid=%d", stream.header.msgTypeID)
	}
	return nil
}

func (s *ClientSession) doCommandMessage(stream *Stream) error {
	cmd, err := stream.msg.readStringWithType()
	if err != nil {
		return err
	}

	tid, err := stream.msg.readNumberWithType()
	if err != nil {
		return err
	}

	switch cmd {
	case "onBWDone":
		log.Warn("-----> onBWDone. ignore")
	case "_result":
		return s.doResultMessage(stream, tid)
	case "onStatus":
		return s.doOnStatusMessage(stream, tid)
	default:
		log.Errorf("unknown cmd. cmd=%s", cmd)
	}

	return nil
}

func (s *ClientSession) doOnStatusMessage(stream *Stream, tid int) error {
	if err := stream.msg.readNull(); err != nil {
		return err
	}
	infos, err := stream.msg.readObjectWithType()
	if err != nil {
		return err
	}
	code, ok := infos["code"]
	if !ok {
		return rtmpErr
	}
	switch s.t {
	case CSTPushSession:
		switch code {
		case "NetStream.Publish.Start":
			log.Info("-----> onStatus('NetStream.Publish.Start')")
			s.notifyPullResultSucc()
		default:
			log.Errorf("unknown code. code=%s", code)
		}
	case CSTPullSession:
		switch code {
		case "NetStream.Play.Start":
			log.Info("-----> onStatus('NetStream.Play.Start')")
			s.notifyPullResultSucc()
		default:
			log.Errorf("unknown code. code=%s", code)
		}
	}

	return nil
}

func (s *ClientSession) doResultMessage(stream *Stream, tid int) error {
	switch tid {
	case tidClientConnect:
		_, err := stream.msg.readObjectWithType()
		if err != nil {
			return err
		}
		infos, err := stream.msg.readObjectWithType()
		if err != nil {
			return err
		}
		code, ok := infos["code"].(string)
		if !ok {
			return rtmpErr
		}
		switch code {
		case "NetConnection.Connect.Success":
			log.Info("-----> _result(\"NetConnection.Connect.Success\")")
			if err := s.packer.writeCreateStream(s.Conn); err != nil {
				return err
			}
		default:
			log.Errorf("unknown code. code=%s", code)
		}
	case tidClientCreateStream:
		err := stream.msg.readNull()
		if err != nil {
			return err
		}
		sid, err := stream.msg.readNumberWithType()
		if err != nil {
			return err
		}
		log.Info("-----> _result()")
		switch s.t {
		case CSTPullSession:
			if err := s.packer.writePlay(s.Conn, s.streamName, sid); err != nil {
				return err
			}
		case CSTPushSession:
			if err := s.packer.writePublish(s.Conn, s.appName, s.streamName, sid); err != nil {
				return err
			}
		}
	default:
		log.Errorf("unknown tid. tid=%d", tid)
	}
	return nil
}

func (s *ClientSession) doProtocolControlMessage(stream *Stream) error {
	if stream.msg.len() < 4 {
		return rtmpErr
	}
	val := int(bele.BEUint32(stream.msg.buf))

	switch stream.header.msgTypeID {
	case typeidWinAckSize:
		s.peerWinAckSize = val
		log.Infof("-----> Window Acknowledgement Size: %d", s.peerWinAckSize)
	case typeidBandwidth:
		log.Warn("-----> Set Peer Bandwidth. ignore")
	case typeidSetChunkSize:
		s.peerChunkSize = val
		log.Infof("-----> Set Chunk Size %d", s.peerChunkSize)
	default:
		log.Errorf("unknown msg type id. id=%d", stream.header.msgTypeID)
	}
	return nil
}

func (s *ClientSession) parseURL(rawURL string) error {
	var err error
	s.url, err = url.Parse(rawURL)
	if err != nil {
		return err
	}
	if s.url.Scheme != "rtmp" || len(s.url.Host) == 0 || len(s.url.Path) == 0 || s.url.Path[0] != '/' {
		return rtmpErr
	}
	index := strings.LastIndexByte(rawURL, '/')
	if index == -1 {
		return rtmpErr
	}
	s.tcURL = rawURL[:index]
	strs := strings.Split(s.url.Path[1:], "/")
	if len(strs) != 2 {
		return rtmpErr
	}
	s.appName = strs[0]
	s.streamName = strs[1]
	log.Debugf("%s %s %s %+v", s.tcURL, s.appName, s.streamName, *s.url)

	return nil
}

func (s *ClientSession) handshake() error {
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

func (s *ClientSession) tcpConnect() error {
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

func (s *ClientSession) getOrCreateStream(csid int) *Stream {
	stream, exist := s.csid2stream[csid]
	if !exist {
		stream = NewStream()
		s.csid2stream[csid] = stream
	}
	return stream
}

func (s *ClientSession) notifyPullResultSucc() {
	s.doResultChan <- nil
}
