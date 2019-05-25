package rtmp

import (
	"bufio"
	"encoding/hex"
	"github.com/q191201771/lal/bele"
	"github.com/q191201771/lal/log"
	"github.com/q191201771/lal/util"
	"net"
	"net/url"
	"strings"
	"time"
)

// rtmp客户端类型连接的底层实现
// rtmp包的使用者应该优先使用基于ClientSession实现的PushSession和PullSession
type ClientSession struct {
	t              ClientSessionType
	obs            PullSessionObserver
	connectTimeout int64
	doResultChan   chan struct{}
	errChan        chan error
	packer         *MessagePacker
	composer       *Composer
	url            *url.URL
	tcURL          string
	appName        string
	streamName     string
	hs             HandshakeClient
	Conn           net.Conn
	rb             *bufio.Reader
	wb             *bufio.Writer
	peerWinAckSize int

	UniqueKey string
}

type ClientSessionType int

const (
	CSTPullSession ClientSessionType = 1
	CSTPushSession ClientSessionType = 2
)

// set <obs> if <t> equal CSTPullSession
func NewClientSession(t ClientSessionType, obs PullSessionObserver, connectTimeout int64) *ClientSession {
	var uk string
	switch t {
	case CSTPullSession:
		uk = "RTMPPULL"
	case CSTPushSession:
		uk = "RTMPPUSH"
	default:
		panic("unreached.")
	}

	return &ClientSession{
		t:              t,
		obs:            obs,
		connectTimeout: connectTimeout,
		doResultChan:   make(chan struct{}),
		errChan:        make(chan error),
		packer:         NewMessagePacker(),
		composer:       NewComposer(),
		UniqueKey:      util.GenUniqueKey(uk),
	}
}

// block until server reply publish / play start or timeout.
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
		s.errChan <- s.runReadLoop()
	}()

	t := time.NewTimer(time.Duration(s.connectTimeout) * time.Second)

	var ret error
	select {
	case <-s.doResultChan:
		break
	case <-t.C:
		ret = rtmpErr
	}
	t.Stop()
	return ret
}

func (s *ClientSession) WaitLoop() error {
	return <-s.errChan
}

func (s *ClientSession) runReadLoop() error {
	return s.composer.RunLoop(s.rb, s.doMsg)
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
		return s.doDataMessageAMF0(stream)
	case typeidAudio:
		fallthrough
	case typeidVideo:
		s.obs.ReadAvMessageCB(stream.header.msgTypeID, stream.timestampAbs, stream.msg.buf[stream.msg.b:stream.msg.e])
	default:
		log.Errorf("unknown msg type id. typeid=%d", stream.header.msgTypeID)
	}
	return nil
}

func (s *ClientSession) doDataMessageAMF0(stream *Stream) error {
	val, err := stream.msg.peekStringWithType()
	if err != nil {
		return err
	}

	switch val {
	case "|RtmpSampleAccess": // TODO chef: handle this?
		return nil
	default:
		log.Error(val)
		log.Error(hex.Dump(stream.msg.buf[stream.msg.b:stream.msg.e]))
	}
	s.obs.ReadAvMessageCB(stream.header.msgTypeID, stream.timestampAbs, stream.msg.buf[stream.msg.b:stream.msg.e])
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
			s.notifyDoResultSucc()
		default:
			log.Errorf("unknown code. code=%s", code)
		}
	case CSTPullSession:
		switch code {
		case "NetStream.Play.Start":
			log.Info("-----> onStatus('NetStream.Play.Start')")
			s.notifyDoResultSucc()
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
		// composer内部会自动更新peer chunk size.
		log.Infof("-----> Set Chunk Size %d", val)
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
	if err := s.hs.WriteC0C1(s.Conn); err != nil {
		return err
	}
	if err := s.hs.ReadS0S1S2(s.rb); err != nil {
		return err
	}
	if err := s.hs.WriteC2(s.Conn); err != nil {
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

func (s *ClientSession) notifyDoResultSucc() {
	s.doResultChan <- struct{}{}
}
