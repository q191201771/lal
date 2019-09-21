package rtmp

import (
	"encoding/hex"
	"github.com/q191201771/nezha/pkg/bele"
	"github.com/q191201771/nezha/pkg/connection"
	"github.com/q191201771/nezha/pkg/log"
	"github.com/q191201771/nezha/pkg/unique"
	"net"
	"net/url"
	"strings"
	"time"
)

// rtmp客户端类型连接的底层实现
// rtmp包的使用者应该优先使用基于ClientSession实现的PushSession和PullSession
type ClientSession struct {
	UniqueKey string

	t                      ClientSessionType
	obs                    PullSessionObserver // only for PullSession
	timeout                ClientSessionTimeout
	doResultChan           chan struct{}
	errChan                chan error
	packer                 *MessagePacker
	chunkComposer          *ChunkComposer
	url                    *url.URL
	tcURL                  string
	appName                string
	streamName             string
	streamNameWithRawQuery string
	hc                     HandshakeClientSimple
	peerWinAckSize         int

	Conn  connection.Connection
	wChan chan []byte
}

type ClientSessionType int

const (
	CSTPullSession ClientSessionType = iota
	CSTPushSession
)

// 单位毫秒，如果为0，则没有超时
type ClientSessionTimeout struct {
	ConnectTimeoutMS int // 建立连接超时
	DoTimeoutMS      int // 从发起连接（包含了建立连接的时间）到收到publish或play信令结果的超时
	ReadAVTimeoutMS  int // 读取音视频数据的超时
	WriteAVTimeoutMS int // 发送音视频数据的超时
}

// @param t: session的类型，只能是推或者拉
// @param obs: 回调结束后，buffer会被重复使用
// @param timeout: 设置各种超时
func NewClientSession(t ClientSessionType, obs PullSessionObserver, timeout ClientSessionTimeout) *ClientSession {
	var uk string
	switch t {
	case CSTPullSession:
		uk = "RTMPPULL"
	case CSTPushSession:
		uk = "RTMPPUSH"
	}
	log.Infof("lifecycle new rtmp client session. [%s]", uk)

	return &ClientSession{
		UniqueKey:     unique.GenUniqueKey(uk),
		t:             t,
		obs:           obs,
		timeout:       timeout,
		doResultChan:  make(chan struct{}),
		errChan:       make(chan error),
		packer:        NewMessagePacker(),
		chunkComposer: NewChunkComposer(),
		wChan:         make(chan []byte, wChanSize),
	}
}

// 阻塞直到收到服务端返回的 publish start / play start 信令 或者超时
func (s *ClientSession) Do(rawURL string) error {
	t := time.NewTimer(time.Duration(s.timeout.DoTimeoutMS) * time.Millisecond)
	select {
	case err := <-s.do(rawURL):
		return err
	case <-t.C:
		return rtmpErr

	}
}

func (s *ClientSession) do(rawURL string) <-chan error {
	ch := make(chan error, 1)
	if err := s.parseURL(rawURL); err != nil {
		ch <- err
		return ch
	}
	if err := s.tcpConnect(); err != nil {
		ch <- err
		return ch
	}

	if err := s.handshake(); err != nil {
		ch <- err
		return ch
	}

	log.Infof("<----- SetChunkSize %d. [%s]", LocalChunkSize, s.UniqueKey)
	if err := s.packer.writeChunkSize(s.Conn, LocalChunkSize); err != nil {
		ch <- err
		return ch
	}

	log.Infof("<----- connect('%s'). [%s]", s.appName, s.UniqueKey)
	if err := s.packer.writeConnect(s.Conn, s.appName, s.tcURL); err != nil {
		ch <- err
		return ch
	}

	go func() {
		s.errChan <- s.runReadLoop()
	}()

	select {
	case <-s.doResultChan:
		ch <- nil
		break
	case err := <-s.errChan:
		ch <- err
		break
	}
	return ch
}

func (s *ClientSession) WaitLoop() error {
	return <-s.errChan
}

// TODO chef: mod to async
func (s *ClientSession) TmpWrite(b []byte) error {
	_, err := s.Conn.Write(b)
	return err
}

func (s *ClientSession) runReadLoop() error {
	return s.chunkComposer.RunLoop(s.Conn, s.doMsg)
}

func (s *ClientSession) doMsg(stream *Stream) error {
	switch stream.header.MsgTypeID {
	case typeidWinAckSize:
		fallthrough
	case typeidBandwidth:
		fallthrough
	case typeidSetChunkSize:
		return s.doProtocolControlMessage(stream)
	case typeidCommandMessageAMF0:
		return s.doCommandMessage(stream)
	case TypeidDataMessageAMF0:
		return s.doDataMessageAMF0(stream)
	case typeidAck:
		return s.doAck(stream)
	case typeidUserControl:
		log.Warnf("read user control message, ignore. [%s]", s.UniqueKey)
	case TypeidAudio:
		fallthrough
	case TypeidVideo:
		s.obs.ReadRTMPAVMsgCB(stream.header, stream.timestampAbs, stream.msg.buf[stream.msg.b:stream.msg.e])
	default:
		log.Errorf("read unknown msg type id. [%s] typeid=%+v", s.UniqueKey, stream.header)
		panic(0)
	}
	return nil
}

func (s *ClientSession) doAck(stream *Stream) error {
	seqNum := bele.BEUint32(stream.msg.buf[stream.msg.b:stream.msg.e])
	log.Infof("-----> Acknowledgement. [%s] ignore. sequence number=%d.", s.UniqueKey, seqNum)
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
	s.obs.ReadRTMPAVMsgCB(stream.header, stream.timestampAbs, stream.msg.buf[stream.msg.b:stream.msg.e])
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
		log.Warnf("-----> onBWDone. ignore. [%s]", s.UniqueKey)
	case "_result":
		return s.doResultMessage(stream, tid)
	case "onStatus":
		return s.doOnStatusMessage(stream, tid)
	default:
		log.Errorf("read unknown cmd. [%s] cmd=%s", s.UniqueKey, cmd)
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
			log.Infof("-----> onStatus('NetStream.Publish.Start'). [%s]", s.UniqueKey)
			s.notifyDoResultSucc()
		default:
			log.Errorf("read on status message but code field unknown. [%s] code=%s", s.UniqueKey, code)
		}
	case CSTPullSession:
		switch code {
		case "NetStream.Play.Start":
			log.Infof("-----> onStatus('NetStream.Play.Start'). [%s]", s.UniqueKey)
			s.notifyDoResultSucc()
		default:
			log.Errorf("read on status message but code field unknown. [%s] code=%s", s.UniqueKey, code)
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
			log.Infof("-----> _result(\"NetConnection.Connect.Success\"). [%s]", s.UniqueKey)
			log.Infof("<----- createStream(). [%s]", s.UniqueKey)
			if err := s.packer.writeCreateStream(s.Conn); err != nil {
				return err
			}
		default:
			log.Errorf("unknown code. [%s] code=%s", s.UniqueKey, code)
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
		log.Infof("-----> _result(). [%s]", s.UniqueKey)
		switch s.t {
		case CSTPullSession:
			log.Infof("<----- play('%s'). [%s]", s.streamNameWithRawQuery, s.UniqueKey)
			if err := s.packer.writePlay(s.Conn, s.streamNameWithRawQuery, sid); err != nil {
				return err
			}
		case CSTPushSession:
			log.Infof("<----- publish('%s'). [%s]", s.streamNameWithRawQuery, s.UniqueKey)
			if err := s.packer.writePublish(s.Conn, s.appName, s.streamNameWithRawQuery, sid); err != nil {
				return err
			}
		}
	default:
		log.Errorf("unknown tid. [%s] tid=%d", s.UniqueKey, tid)
	}
	return nil
}

func (s *ClientSession) doProtocolControlMessage(stream *Stream) error {
	if stream.msg.len() < 4 {
		return rtmpErr
	}
	val := int(bele.BEUint32(stream.msg.buf))

	switch stream.header.MsgTypeID {
	case typeidWinAckSize:
		s.peerWinAckSize = val
		log.Infof("-----> Window Acknowledgement Size: %d. [%s]", s.peerWinAckSize, s.UniqueKey)
	case typeidBandwidth:
		log.Warnf("-----> Set Peer Bandwidth. ignore. [%s]", s.UniqueKey)
	case typeidSetChunkSize:
		// composer内部会自动更新peer chunk size.
		log.Infof("-----> Set Chunk Size %d. [%s]", val, s.UniqueKey)
	default:
		log.Errorf("unknown msg type id. [%s] id=%d", s.UniqueKey, stream.header.MsgTypeID)
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
	// 有的rtmp服务器会使用url后面的参数（比如说用于鉴权），这里把它带上
	s.streamName = strs[1]
	if s.url.RawQuery == "" {
		s.streamNameWithRawQuery = s.streamName
	} else {
		s.streamNameWithRawQuery = s.streamName + "?" + s.url.RawQuery
	}
	log.Debugf("parseURL. [%s] %s %s %s %+v", s.UniqueKey, s.tcURL, s.appName, s.streamNameWithRawQuery, *s.url)

	return nil
}

func (s *ClientSession) handshake() error {
	log.Infof("<----- Handshake C0+C1. [%s]", s.UniqueKey)
	if err := s.hc.WriteC0C1(s.Conn); err != nil {
		return err
	}

	if err := s.hc.ReadS0S1S2(s.Conn); err != nil {
		return err
	}
	log.Infof("-----> Handshake S0+S1+S2. [%s]", s.UniqueKey)

	log.Infof("<----- Handshake C2. [%s]", s.UniqueKey)
	if err := s.hc.WriteC2(s.Conn); err != nil {
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

	var conn net.Conn
	if conn, err = net.DialTimeout("tcp", addr, time.Duration(s.timeout.ConnectTimeoutMS)*time.Millisecond); err != nil {
		return err
	}

	s.Conn = connection.New(conn, connection.Config{
		ReadBufSize: readBufSize,
	})
	return nil
}

func (s *ClientSession) notifyDoResultSucc() {
	s.Conn.ModWriteBufSize(writeBufSize)
	s.Conn.ModReadTimeoutMS(s.timeout.ReadAVTimeoutMS)
	s.Conn.ModWriteTimeoutMS(s.timeout.WriteAVTimeoutMS)
	s.doResultChan <- struct{}{}
}
