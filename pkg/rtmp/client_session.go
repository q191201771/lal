// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/connection"
)

// ClientSession rtmp 客户端类型连接的底层实现
// package rtmp 的使用者应该优先使用基于 ClientSession 实现的 PushSession 和 PullSession
type ClientSession struct {
	uniqueKey string

	onDoResult func()

	// 只有PullSession使用
	onReadRtmpAvMsg OnReadRtmpAvMsg

	t      ClientSessionType
	option ClientSessionOption

	packer        *MessagePacker
	chunkComposer *ChunkComposer
	urlCtx        base.UrlContext
	hc            IHandshakeClient

	conn                  connection.Connection
	prevConnStat          connection.Stat
	staleStat             *connection.Stat
	stat                  base.StatSession
	doResultChan          chan struct{}
	hasNotifyDoResultSucc bool

	debugLogReadUserCtrlMsgCount int
	debugLogReadUserCtrlMsgMax   int

	recvLastAck uint64
	seqNum      uint32

	disposeOnce sync.Once
}

type ClientSessionType int

const (
	CstPullSession ClientSessionType = iota
	CstPushSession
)

type ClientSessionOption struct {
	// 单位毫秒，如果为0，则没有超时
	DoTimeoutMs      int // 从发起连接（包含了建立连接的时间）到收到publish或play信令结果的超时
	ReadAvTimeoutMs  int // 读取音视频数据的超时
	WriteAvTimeoutMs int // 发送音视频数据的超时

	ReadBufSize   int // io层读取音视频数据时的缓冲大小，如果为0，则没有缓冲
	WriteBufSize  int // io层发送音视频数据的缓冲大小，如果为0，则没有缓冲
	WriteChanSize int // io层发送音视频数据的异步队列大小，如果为0，则同步发送

	HandshakeComplexFlag bool // 握手是否使用复杂模式

	PeerWinAckSize int
}

var defaultClientSessOption = ClientSessionOption{
	DoTimeoutMs:          10000,
	ReadAvTimeoutMs:      0,
	WriteAvTimeoutMs:     0,
	ReadBufSize:          0,
	WriteBufSize:         0,
	WriteChanSize:        0,
	HandshakeComplexFlag: false,
	PeerWinAckSize:       0,
}

type ModClientSessionOption func(option *ClientSessionOption)

// NewClientSession @param t: session的类型，只能是推或者拉
func NewClientSession(t ClientSessionType, modOptions ...ModClientSessionOption) *ClientSession {
	var uk string
	switch t {
	case CstPullSession:
		uk = base.GenUkRtmpPullSession()
	case CstPushSession:
		uk = base.GenUkRtmpPushSession()
	}

	option := defaultClientSessOption
	for _, fn := range modOptions {
		fn(&option)
	}

	var hc IHandshakeClient
	if option.HandshakeComplexFlag {
		hc = &HandshakeClientComplex{}
	} else {
		hc = &HandshakeClientSimple{}
	}

	s := &ClientSession{
		uniqueKey:       uk,
		onDoResult:      defaultOnPullResult,
		onReadRtmpAvMsg: defaultOnReadRtmpAvMsg,
		t:               t,
		option:          option,
		doResultChan:    make(chan struct{}, 1),
		packer:          NewMessagePacker(),
		chunkComposer:   NewChunkComposer(),
		stat: base.StatSession{
			Protocol:  base.ProtocolRtmp,
			SessionId: uk,
			StartTime: base.ReadableNowTime(),
		},
		debugLogReadUserCtrlMsgMax: 5,
		hc:                         hc,
	}
	Log.Infof("[%s] lifecycle new rtmp ClientSession. session=%p", uk, s)
	return s
}

// Do 阻塞直到收到服务端返回的 publish / play 对应结果的信令或者发生错误
func (s *ClientSession) Do(rawUrl string) error {
	Log.Debugf("[%s] Do. url=%s", s.uniqueKey, rawUrl)

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if s.option.DoTimeoutMs == 0 {
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(s.option.DoTimeoutMs)*time.Millisecond)
	}
	defer cancel()
	return s.doContext(ctx, rawUrl)
}

func (s *ClientSession) Write(msg []byte) error {
	if s.conn == nil {
		return base.ErrSessionNotStarted
	}
	_, err := s.conn.Write(msg)
	return err
}

func (s *ClientSession) Flush() error {
	if s.conn == nil {
		return base.ErrSessionNotStarted
	}
	return s.conn.Flush()
}

// ---------------------------------------------------------------------------------------------------------------------
// IClientSessionLifecycle interface
// ---------------------------------------------------------------------------------------------------------------------

// Dispose 文档请参考： IClientSessionLifecycle interface
//
func (s *ClientSession) Dispose() error {
	return s.dispose(nil)
}

// WaitChan 文档请参考： IClientSessionLifecycle interface
//
func (s *ClientSession) WaitChan() <-chan error {
	return s.conn.Done()
}

// ---------------------------------------------------------------------------------------------------------------------

func (s *ClientSession) Url() string {
	return s.urlCtx.Url
}

func (s *ClientSession) AppName() string {
	return s.urlCtx.PathWithoutLastItem
}

func (s *ClientSession) StreamName() string {
	return s.urlCtx.LastItemOfPath
}

func (s *ClientSession) RawQuery() string {
	return s.urlCtx.RawQuery
}

func (s *ClientSession) UniqueKey() string {
	return s.uniqueKey
}

func (s *ClientSession) GetStat() base.StatSession {
	connStat := s.conn.GetStat()
	s.stat.ReadBytesSum = connStat.ReadBytesSum
	s.stat.WroteBytesSum = connStat.WroteBytesSum
	return s.stat
}

func (s *ClientSession) UpdateStat(intervalSec uint32) {
	currStat := s.conn.GetStat()
	rDiff := currStat.ReadBytesSum - s.prevConnStat.ReadBytesSum
	s.stat.ReadBitrate = int(rDiff * 8 / 1024 / uint64(intervalSec))
	wDiff := currStat.WroteBytesSum - s.prevConnStat.WroteBytesSum
	s.stat.WriteBitrate = int(wDiff * 8 / 1024 / uint64(intervalSec))
	switch s.t {
	case CstPushSession:
		s.stat.Bitrate = s.stat.WriteBitrate
	case CstPullSession:
		s.stat.Bitrate = s.stat.ReadBitrate
	}
	s.prevConnStat = currStat
}

func (s *ClientSession) IsAlive() (readAlive, writeAlive bool) {
	currStat := s.conn.GetStat()
	if s.staleStat == nil {
		s.staleStat = new(connection.Stat)
		*s.staleStat = currStat
		return true, true
	}

	readAlive = !(currStat.ReadBytesSum-s.staleStat.ReadBytesSum == 0)
	writeAlive = !(currStat.WroteBytesSum-s.staleStat.WroteBytesSum == 0)
	*s.staleStat = currStat
	return
}

func (s *ClientSession) doContext(ctx context.Context, rawUrl string) error {
	errChan := make(chan error, 1)

	go func() {
		if err := s.parseUrl(rawUrl); err != nil {
			errChan <- err
			return
		}
		if err := s.tcpConnect(); err != nil {
			errChan <- err
			return
		}

		if err := s.handshake(); err != nil {
			errChan <- err
			return
		}

		Log.Infof("[%s] > W SetChunkSize %d.", s.uniqueKey, LocalChunkSize)
		if err := s.packer.writeChunkSize(s.conn, LocalChunkSize); err != nil {
			errChan <- err
			return
		}

		Log.Infof("[%s] > W connect('%s'). tcUrl=%s", s.uniqueKey, s.appName(), s.tcUrl())
		if err := s.packer.writeConnect(s.conn, s.appName(), s.tcUrl(), s.t == CstPushSession); err != nil {
			errChan <- err
			return
		}

		s.runReadLoop()
	}()

	select {
	case <-ctx.Done():
		_ = s.dispose(nil)
		return ctx.Err()
	case err := <-errChan:
		_ = s.dispose(err)
		return err
	case <-s.doResultChan:
		return nil
	}
}

func (s *ClientSession) parseUrl(rawUrl string) (err error) {
	s.urlCtx, err = base.ParseRtmpUrl(rawUrl)
	if err != nil {
		return err
	}

	return
}

func (s *ClientSession) tcUrl() string {
	return fmt.Sprintf("%s://%s/%s", s.urlCtx.Scheme, s.urlCtx.StdHost, s.urlCtx.PathWithoutLastItem)
}
func (s *ClientSession) appName() string {
	return s.urlCtx.PathWithoutLastItem
}

func (s *ClientSession) streamNameWithRawQuery() string {
	if s.urlCtx.RawQuery == "" {
		return s.urlCtx.LastItemOfPath
	}
	return fmt.Sprintf("%s?%s", s.urlCtx.LastItemOfPath, s.urlCtx.RawQuery)
}

func (s *ClientSession) tcpConnect() error {
	Log.Infof("[%s] > tcp connect.", s.uniqueKey)
	var err error

	s.stat.RemoteAddr = s.urlCtx.HostWithPort

	var conn net.Conn
	if conn, err = net.Dial("tcp", s.urlCtx.HostWithPort); err != nil {
		return err
	}

	s.conn = connection.New(conn, func(option *connection.Option) {
		option.ReadBufSize = s.option.ReadBufSize
		option.WriteChanFullBehavior = connection.WriteChanFullBehaviorBlock
	})
	return nil
}

func (s *ClientSession) handshake() error {
	Log.Infof("[%s] > W Handshake C0+C1.", s.uniqueKey)
	if err := s.hc.WriteC0C1(s.conn); err != nil {
		return err
	}

	if err := s.hc.ReadS0S1(s.conn); err != nil {
		return err
	}
	Log.Infof("[%s] < R Handshake S0+S1.", s.uniqueKey)

	Log.Infof("[%s] > W Handshake C2.", s.uniqueKey)
	if err := s.hc.WriteC2(s.conn); err != nil {
		return err
	}

	if err := s.hc.ReadS2(s.conn); err != nil {
		return err
	}
	Log.Infof("[%s] < R Handshake S2.", s.uniqueKey)
	return nil
}

func (s *ClientSession) runReadLoop() {
	if err := s.chunkComposer.RunLoop(s.conn, s.doMsg); err != nil {
		_ = s.dispose(err)
	}
}

func (s *ClientSession) doMsg(stream *Stream) error {
	if s.t == CstPullSession {
		if err := s.doRespAcknowledgement(stream); err != nil {
			return err
		}
	}
	switch stream.header.MsgTypeId {
	case base.RtmpTypeIdWinAckSize:
		fallthrough
	case base.RtmpTypeIdBandwidth:
		fallthrough
	case base.RtmpTypeIdSetChunkSize:
		return s.doProtocolControlMessage(stream)
	case base.RtmpTypeIdCommandMessageAmf0:
		return s.doCommandMessage(stream)
	case base.RtmpTypeIdMetadata:
		return s.doDataMessageAmf0(stream)
	case base.RtmpTypeIdAck:
		return s.doAck(stream)
	case base.RtmpTypeIdUserControl:
		return s.doUserControl(stream)
	case base.RtmpTypeIdAudio:
		fallthrough
	case base.RtmpTypeIdVideo:
		s.onReadRtmpAvMsg(stream.toAvMsg())
	default:
		Log.Errorf("[%s] read unknown message. typeid=%d, %s", s.uniqueKey, stream.header.MsgTypeId, stream.toDebugString())
		panic(0)
	}
	return nil
}

func (s *ClientSession) doAck(stream *Stream) error {
	seqNum := bele.BeUint32(stream.msg.buff.Bytes())
	Log.Infof("[%s] < R Acknowledgement. ignore. sequence number=%d.", s.uniqueKey, seqNum)
	return nil
}
func (s *ClientSession) doUserControl(stream *Stream) error {
	userControlType := bele.BeUint16(stream.msg.buff.Bytes())
	if userControlType == uint16(base.RtmpUserControlPingRequest) {
		stream.msg.buff.Skip(2)
		timestamp := bele.BeUint32(stream.msg.buff.Bytes())
		return s.packer.writePingResponse(s.conn, timestamp)
	}

	s.debugLogReadUserCtrlMsgCount++
	if s.debugLogReadUserCtrlMsgCount <= s.debugLogReadUserCtrlMsgMax {
		Log.Warnf("[%s] read user control message, ignore. buf=%s",
			s.uniqueKey, hex.Dump(stream.msg.buff.Peek(32)))
	}
	return nil
}

func (s *ClientSession) doDataMessageAmf0(stream *Stream) error {
	val, err := stream.msg.peekStringWithType()
	if err != nil {
		return err
	}

	switch val {
	case "|RtmpSampleAccess":
		Log.Debugf("[%s] < R |RtmpSampleAccess, ignore.", s.uniqueKey)
		return nil
	default:
	}
	s.onReadRtmpAvMsg(stream.toAvMsg())
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
		Log.Warnf("[%s] < R onBWDone. ignore.", s.uniqueKey)
	case "_result":
		return s.doResultMessage(stream, tid)
	case "onStatus":
		return s.doOnStatusMessage(stream, tid)
	default:
		Log.Errorf("[%s] read unknown command message. cmd=%s, %s", s.uniqueKey, cmd, stream.toDebugString())
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
	case CstPushSession:
		switch code {
		case "NetStream.Publish.Start":
			Log.Infof("[%s] < R onStatus('NetStream.Publish.Start').", s.uniqueKey)
			s.notifyDoResultSucc()
		default:
			Log.Warnf("[%s] read on status message but code field unknown. code=%s", s.uniqueKey, code)
		}
	case CstPullSession:
		switch code {
		case "NetStream.Play.Start":
			Log.Infof("[%s] < R onStatus('NetStream.Play.Start').", s.uniqueKey)
			s.notifyDoResultSucc()
		default:
			Log.Warnf("[%s] read on status message but code field unknown. code=%s", s.uniqueKey, code)
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
			Log.Infof("[%s] < R _result(\"NetConnection.Connect.Success\").", s.uniqueKey)
			Log.Infof("[%s] > W createStream().", s.uniqueKey)
			if err := s.packer.writeCreateStream(s.conn); err != nil {
				return err
			}
		default:
			Log.Errorf("[%s] unknown code. code=%v", s.uniqueKey, code)
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
		Log.Infof("[%s] < R _result().", s.uniqueKey)
		switch s.t {
		case CstPullSession:
			Log.Infof("[%s] > W play('%s').", s.uniqueKey, s.streamNameWithRawQuery())
			if err := s.packer.writePlay(s.conn, s.streamNameWithRawQuery(), sid); err != nil {
				return err
			}
		case CstPushSession:
			Log.Infof("[%s] > W publish('%s').", s.uniqueKey, s.streamNameWithRawQuery())
			if err := s.packer.writePublish(s.conn, s.appName(), s.streamNameWithRawQuery(), sid); err != nil {
				return err
			}
		}
	default:
		Log.Errorf("[%s] unknown tid. tid=%d", s.uniqueKey, tid)
	}
	return nil
}
func (s *ClientSession) doProtocolControlMessage(stream *Stream) error {
	if stream.msg.Len() < 4 {
		return base.NewErrRtmpShortBuffer(4, int(stream.msg.Len()), "ClientSession::doProtocolControlMessage")
	}
	val := int(bele.BeUint32(stream.msg.buff.Bytes()))

	switch stream.header.MsgTypeId {
	case base.RtmpTypeIdWinAckSize:
		s.option.PeerWinAckSize = val
		Log.Infof("[%s] < R Window Acknowledgement Size: %d", s.uniqueKey, s.option.PeerWinAckSize)
	case base.RtmpTypeIdBandwidth:
		// TODO chef: 是否需要关注这个信令
		Log.Warnf("[%s] < R Set Peer Bandwidth. ignore.", s.uniqueKey)
	case base.RtmpTypeIdSetChunkSize:
		// composer内部会自动更新peer chunk size.
		Log.Infof("[%s] < R Set Chunk Size %d.", s.uniqueKey, val)
	default:
		Log.Errorf("[%s] read unknown protocol control message. typeid=%d, %s", s.uniqueKey, stream.header.MsgTypeId, stream.toDebugString())
	}
	return nil
}

func (s *ClientSession) doRespAcknowledgement(stream *Stream) error {
	// https://github.com/q191201771/lal/pull/154
	if s.option.PeerWinAckSize <= 0 {
		return nil
	}
	currStat := s.conn.GetStat()
	delta := uint32(currStat.ReadBytesSum - s.recvLastAck)
	//此次接收小于窗口大小一半，不处理
	if delta < uint32(s.option.PeerWinAckSize/2) {
		return nil
	}
	s.recvLastAck = currStat.ReadBytesSum
	seqNum := s.seqNum + delta
	//当序列号溢出时，将其重置
	if seqNum > 0xf0000000 {
		seqNum = delta
	}
	s.seqNum = seqNum
	//时间戳暂时先发0
	return s.packer.writeAcknowledgement(s.conn, seqNum)
}
func (s *ClientSession) notifyDoResultSucc() {
	// 碰上过对端服务器实现有问题，对于play信令回复了两次相同的结果，我们在这里忽略掉非第一次的回复
	if s.hasNotifyDoResultSucc {
		Log.Warnf("[%s] has notified do result succ already, ignore it", s.uniqueKey)
		return
	}
	s.hasNotifyDoResultSucc = true

	s.conn.ModWriteChanSize(s.option.WriteChanSize)
	//pull有可能还需要小包发送，不使用缓存
	if s.t == CstPushSession {
		s.conn.ModWriteBufSize(s.option.WriteBufSize)
	}

	s.conn.ModReadTimeoutMs(s.option.ReadAvTimeoutMs)
	s.conn.ModWriteTimeoutMs(s.option.WriteAvTimeoutMs)

	s.onDoResult()
	s.doResultChan <- struct{}{}
}

func (s *ClientSession) dispose(err error) error {
	var retErr error
	s.disposeOnce.Do(func() {
		Log.Infof("[%s] lifecycle dispose rtmp ClientSession. err=%+v", s.uniqueKey, err)
		if s.conn == nil {
			retErr = base.ErrSessionNotStarted
			return
		}
		retErr = s.conn.Close()
	})
	return retErr
}

func defaultOnPullResult() {
}

func defaultOnReadRtmpAvMsg(msg base.RtmpMsg) {

}
