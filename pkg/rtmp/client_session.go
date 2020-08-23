// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/connection"
	log "github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"
)

var ErrClientSessionTimeout = errors.New("lal.rtmp: client session timeout")

// rtmp 客户端类型连接的底层实现
// package rtmp 的使用者应该优先使用基于 ClientSession 实现的 PushSession 和 PullSession
type ClientSession struct {
	UniqueKey string

	t      ClientSessionType
	option ClientSessionOption

	packer                 *MessagePacker
	chunkComposer          *ChunkComposer
	url                    *url.URL
	tcURL                  string
	appName                string
	streamName             string
	streamNameWithRawQuery string
	hc                     HandshakeClientSimple
	peerWinAckSize         int

	conn         connection.Connection
	doResultChan chan struct{}

	// 只有PullSession使用
	onReadRTMPAVMsg OnReadRTMPAVMsg
}

type ClientSessionType int

const (
	CSTPullSession ClientSessionType = iota
	CSTPushSession
)

type ClientSessionOption struct {
	// 单位毫秒，如果为0，则没有超时
	ConnectTimeoutMS int // 建立连接超时
	DoTimeoutMS      int // 从发起连接（包含了建立连接的时间）到收到publish或play信令结果的超时
	ReadAVTimeoutMS  int // 读取音视频数据的超时
	WriteAVTimeoutMS int // 发送音视频数据的超时
}

var defaultClientSessOption = ClientSessionOption{
	ConnectTimeoutMS: 0,
	DoTimeoutMS:      0,
	ReadAVTimeoutMS:  0,
	WriteAVTimeoutMS: 0,
}

type ModClientSessionOption func(option *ClientSessionOption)

// @param t: session的类型，只能是推或者拉
func NewClientSession(t ClientSessionType, modOptions ...ModClientSessionOption) *ClientSession {
	var uk string
	switch t {
	case CSTPullSession:
		uk = unique.GenUniqueKey("RTMPPULL")
	case CSTPushSession:
		uk = unique.GenUniqueKey("RTMPPUSH")
	}

	option := defaultClientSessOption
	for _, fn := range modOptions {
		fn(&option)
	}

	s := &ClientSession{
		UniqueKey:     uk,
		t:             t,
		option:        option,
		doResultChan:  make(chan struct{}, 1),
		packer:        NewMessagePacker(),
		chunkComposer: NewChunkComposer(),
	}
	log.Infof("[%s] lifecycle new rtmp ClientSession. session=%p", uk, s)
	return s
}

// 阻塞直到收到服务端返回的 publish / play 对应结果的信令或者发生错误
func (s *ClientSession) doWithTimeout(rawURL string) error {
	if s.option.DoTimeoutMS == 0 {
		err := <-s.do(rawURL)
		return err
	}
	t := time.NewTimer(time.Duration(s.option.DoTimeoutMS) * time.Millisecond)
	defer t.Stop()
	select {
	// TODO chef: 这种写法执行不到超时
	case err := <-s.do(rawURL):
		return err
	case <-t.C:
		return ErrClientSessionTimeout
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

	log.Infof("[%s] > W SetChunkSize %d.", s.UniqueKey, LocalChunkSize)
	if err := s.packer.writeChunkSize(s.conn, LocalChunkSize); err != nil {
		ch <- err
		return ch
	}

	log.Infof("[%s] > W connect('%s').", s.UniqueKey, s.appName)
	if err := s.packer.writeConnect(s.conn, s.appName, s.tcURL, s.t == CSTPushSession); err != nil {
		ch <- err
		return ch
	}

	go s.runReadLoop()

	select {
	case <-s.doResultChan:
		ch <- nil
		break
	case err := <-s.conn.Done():
		ch <- err
		break
	}
	return ch
}

func (s *ClientSession) Done() <-chan error {
	return s.conn.Done()
}

func (s *ClientSession) AsyncWrite(msg []byte) error {
	_, err := s.conn.Write(msg)
	return err
}

func (s *ClientSession) Flush() error {
	return s.conn.Flush()
}

func (s *ClientSession) Dispose() {
	log.Infof("[%s] lifecycle dispose rtmp ClientSession.", s.UniqueKey)
	_ = s.conn.Close()
}

func (s *ClientSession) runReadLoop() {
	_ = s.chunkComposer.RunLoop(s.conn, s.doMsg)
}

func (s *ClientSession) doMsg(stream *Stream) error {
	switch stream.header.MsgTypeID {
	case base.RTMPTypeIDWinAckSize:
		fallthrough
	case base.RTMPTypeIDBandwidth:
		fallthrough
	case base.RTMPTypeIDSetChunkSize:
		return s.doProtocolControlMessage(stream)
	case base.RTMPTypeIDCommandMessageAMF0:
		return s.doCommandMessage(stream)
	case base.RTMPTypeIDMetadata:
		return s.doDataMessageAMF0(stream)
	case base.RTMPTypeIDAck:
		return s.doAck(stream)
	case base.RTMPTypeIDUserControl:
		log.Warnf("[%s] read user control message, ignore.", s.UniqueKey)
	case base.RTMPTypeIDAudio:
		fallthrough
	case base.RTMPTypeIDVideo:
		s.onReadRTMPAVMsg(stream.toAVMsg())
	default:
		log.Errorf("[%s] read unknown message. typeid=%d, %s", s.UniqueKey, stream.header.MsgTypeID, stream.toDebugString())
		panic(0)
	}
	return nil
}

func (s *ClientSession) doAck(stream *Stream) error {
	seqNum := bele.BEUint32(stream.msg.buf[stream.msg.b:stream.msg.e])
	log.Infof("[%s] < R Acknowledgement. ignore. sequence number=%d.", s.UniqueKey, seqNum)
	return nil
}

func (s *ClientSession) doDataMessageAMF0(stream *Stream) error {
	val, err := stream.msg.peekStringWithType()
	if err != nil {
		return err
	}

	switch val {
	case "|RtmpSampleAccess":
		log.Debugf("[%s] < R |RtmpSampleAccess, ignore.", s.UniqueKey)
		return nil
	default:
	}
	s.onReadRTMPAVMsg(stream.toAVMsg())
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
		log.Warnf("[%s] < R onBWDone. ignore.", s.UniqueKey)
	case "_result":
		return s.doResultMessage(stream, tid)
	case "onStatus":
		return s.doOnStatusMessage(stream, tid)
	default:
		log.Errorf("[%s] read unknown command message. cmd=%s, %s", s.UniqueKey, cmd, stream.toDebugString())
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
	code, err := infos.FindString("code")
	if err != nil {
		return err
	}
	switch s.t {
	case CSTPushSession:
		switch code {
		case "NetStream.Publish.Start":
			log.Infof("[%s] < R onStatus('NetStream.Publish.Start').", s.UniqueKey)
			s.notifyDoResultSucc()
		default:
			log.Errorf("[%s] read on status message but code field unknown. code=%s", s.UniqueKey, code)
		}
	case CSTPullSession:
		switch code {
		case "NetStream.Play.Start":
			log.Infof("[%s] < R onStatus('NetStream.Play.Start').", s.UniqueKey)
			s.notifyDoResultSucc()
		default:
			log.Errorf("[%s] read on status message but code field unknown. code=%s", s.UniqueKey, code)
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
		code, err := infos.FindString("code")
		if err != nil {
			return err
		}
		switch code {
		case "NetConnection.Connect.Success":
			log.Infof("[%s] < R _result(\"NetConnection.Connect.Success\").", s.UniqueKey)
			log.Infof("[%s] > W createStream().", s.UniqueKey)
			if err := s.packer.writeCreateStream(s.conn); err != nil {
				return err
			}
		default:
			log.Errorf("[%s] unknown code. code=%v", s.UniqueKey, code)
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
		log.Infof("[%s] < R _result().", s.UniqueKey)
		switch s.t {
		case CSTPullSession:
			log.Infof("[%s] > W play('%s').", s.UniqueKey, s.streamNameWithRawQuery)
			if err := s.packer.writePlay(s.conn, s.streamNameWithRawQuery, sid); err != nil {
				return err
			}
		case CSTPushSession:
			log.Infof("[%s] > W publish('%s').", s.UniqueKey, s.streamNameWithRawQuery)
			if err := s.packer.writePublish(s.conn, s.appName, s.streamNameWithRawQuery, sid); err != nil {
				return err
			}
		}
	default:
		log.Errorf("[%s] unknown tid. tid=%d", s.UniqueKey, tid)
	}
	return nil
}

func (s *ClientSession) doProtocolControlMessage(stream *Stream) error {
	if stream.msg.len() < 4 {
		return ErrRTMP
	}
	val := int(bele.BEUint32(stream.msg.buf))

	switch stream.header.MsgTypeID {
	case base.RTMPTypeIDWinAckSize:
		s.peerWinAckSize = val
		log.Infof("[%s] < R Window Acknowledgement Size: %d", s.UniqueKey, s.peerWinAckSize)
	case base.RTMPTypeIDBandwidth:
		// TODO chef: 是否需要关注这个信令
		log.Debugf("[%s] < R Set Peer Bandwidth. ignore.", s.UniqueKey)
	case base.RTMPTypeIDSetChunkSize:
		// composer内部会自动更新peer chunk size.
		log.Infof("[%s] < R Set Chunk Size %d.", s.UniqueKey, val)
	default:
		log.Errorf("[%s] read unknown protocol control message. typeid=%d, %s", s.UniqueKey, stream.header.MsgTypeID, stream.toDebugString())
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
		return ErrRTMP
	}
	index := strings.LastIndexByte(rawURL, '/')
	if index == -1 {
		return ErrRTMP
	}
	s.tcURL = rawURL[:index]
	strs := strings.Split(s.url.Path[1:], "/")
	if len(strs) != 2 {
		return ErrRTMP
	}
	s.appName = strs[0]
	// 有的rtmp服务器会使用url后面的参数（比如说用于鉴权），这里把它带上
	s.streamName = strs[1]
	if s.url.RawQuery == "" {
		s.streamNameWithRawQuery = s.streamName
	} else {
		s.streamNameWithRawQuery = s.streamName + "?" + s.url.RawQuery
	}
	log.Debugf("[%s] parseURL. %s %s %s %+v", s.UniqueKey, s.tcURL, s.appName, s.streamNameWithRawQuery, *s.url)

	return nil
}

func (s *ClientSession) handshake() error {
	log.Infof("[%s] > W Handshake C0+C1.", s.UniqueKey)
	if err := s.hc.WriteC0C1(s.conn); err != nil {
		return err
	}

	if err := s.hc.ReadS0S1S2(s.conn); err != nil {
		return err
	}
	log.Infof("[%s] < R Handshake S0+S1+S2.", s.UniqueKey)

	log.Infof("[%s] > W Handshake C2.", s.UniqueKey)
	if err := s.hc.WriteC2(s.conn); err != nil {
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
	if conn, err = net.DialTimeout("tcp", addr, time.Duration(s.option.ConnectTimeoutMS)*time.Millisecond); err != nil {
		return err
	}

	s.conn = connection.New(conn, func(option *connection.Option) {
		option.ReadBufSize = readBufSize
		option.WriteChanFullBehavior = connection.WriteChanFullBehaviorBlock
	})
	return nil
}

func (s *ClientSession) notifyDoResultSucc() {
	s.conn.ModWriteChanSize(wChanSize)
	s.conn.ModWriteBufSize(writeBufSize)
	s.conn.ModReadTimeoutMS(s.option.ReadAVTimeoutMS)
	s.conn.ModWriteTimeoutMS(s.option.WriteAVTimeoutMS)

	s.doResultChan <- struct{}{}
}
