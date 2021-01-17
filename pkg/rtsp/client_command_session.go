// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
)

type ClientCommandSessionType int

const (
	readBufSize                   = 256
	writeGetParameterIntervalMSec = 10000
)

const (
	CCSTPullSession ClientCommandSessionType = iota
	CCSTPushSession
)

type ClientCommandSessionOption struct {
	DoTimeoutMS int
	OverTCP     bool
}

var defaultClientCommandSessionOption = ClientCommandSessionOption{
	DoTimeoutMS: 10000,
	OverTCP:     false,
}

type ClientCommandSessionObserver interface {
	OnConnectResult()

	// only for PullSession
	OnDescribeResponse(rawSDP []byte, sdpLogicCtx sdp.LogicContext)

	OnSetupWithConn(uri string, rtpConn, rtcpConn *nazanet.UDPConnection)
	OnSetupWithChannel(uri string, rtpChannel, rtcpChannel int)
	OnSetupResult()

	OnInterleavedPacket(packet []byte, channel int)
}

// Push和Pull共用，封装了客户端底层信令信令部分。
// 业务方应该使用PushSession和PullSession，而不是直接使用ClientCommandSession，除非你确定要这么做。
type ClientCommandSession struct {
	UniqueKey string
	t         ClientCommandSessionType
	observer  ClientCommandSessionObserver
	option    ClientCommandSessionOption

	rawURL string
	urlCtx base.URLContext
	conn   connection.Connection

	cseq                        int
	methodGetParameterSupported bool
	auth                        Auth

	rawSDP      []byte
	sdpLogicCtx sdp.LogicContext

	sessionID string
	channel   int

	waitErrChan chan error
}

type ModClientCommandSessionOption func(option *ClientCommandSessionOption)

func NewClientCommandSession(t ClientCommandSessionType, uniqueKey string, observer ClientCommandSessionObserver, modOptions ...ModClientCommandSessionOption) *ClientCommandSession {
	option := defaultClientCommandSessionOption
	for _, fn := range modOptions {
		fn(&option)
	}
	s := &ClientCommandSession{
		t:           t,
		UniqueKey:   uniqueKey,
		observer:    observer,
		option:      option,
		waitErrChan: make(chan error, 1),
	}
	nazalog.Infof("[%s] lifecycle new rtsp ClientCommandSession. session=%p", uniqueKey, s)
	return s
}

// only for PushSession
func (session *ClientCommandSession) InitWithSDP(rawSDP []byte, sdpLogicCtx sdp.LogicContext) {
	session.rawSDP = rawSDP
	session.sdpLogicCtx = sdpLogicCtx
}

func (session *ClientCommandSession) Do(rawURL string) error {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if session.option.DoTimeoutMS == 0 {
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(session.option.DoTimeoutMS)*time.Millisecond)
	}
	defer cancel()
	return session.doContext(ctx, rawURL)
}

func (session *ClientCommandSession) Wait() <-chan error {
	return session.waitErrChan
}

func (session *ClientCommandSession) Dispose() error {
	nazalog.Infof("[%s] lifecycle dispose rtsp ClientCommandSession. session=%p", session.UniqueKey, session)
	return session.conn.Close()
}

func (session *ClientCommandSession) WriteInterleavedPacket(packet []byte, channel int) error {
	_, err := session.conn.Write(packInterleaved(channel, packet))
	return err
}

func (session *ClientCommandSession) RemoteAddr() string {
	return session.conn.RemoteAddr().String()
}

func (session *ClientCommandSession) AppName() string {
	return session.urlCtx.PathWithoutLastItem
}

func (session *ClientCommandSession) StreamName() string {
	return session.urlCtx.LastItemOfPath
}

func (session *ClientCommandSession) RawQuery() string {
	return session.urlCtx.RawQuery
}

func (session *ClientCommandSession) doContext(ctx context.Context, rawURL string) error {
	errChan := make(chan error, 1)

	go func() {
		if err := session.connect(rawURL); err != nil {
			errChan <- err
			return
		}

		if err := session.writeOptions(); err != nil {
			errChan <- err
			return
		}

		switch session.t {
		case CCSTPullSession:
			if err := session.writeDescribe(); err != nil {
				errChan <- err
				return
			}

			if err := session.writeSetup(); err != nil {
				errChan <- err
				return
			}
			session.observer.OnSetupResult()

			if err := session.writePlay(); err != nil {
				errChan <- err
				return
			}
		case CCSTPushSession:
			if err := session.writeAnnounce(); err != nil {
				errChan <- err
				return
			}

			if err := session.writeSetup(); err != nil {
				errChan <- err
				return
			}
			session.observer.OnSetupResult()

			if err := session.writeRecord(); err != nil {
				errChan <- err
				return
			}
		}

		errChan <- nil
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		if err != nil {
			return err
		}
	}

	go session.runReadLoop()
	return nil
}

func (session *ClientCommandSession) runReadLoop() {
	if !session.methodGetParameterSupported {
		// TCP模式，需要收取数据进行处理
		if session.option.OverTCP {
			var r = bufio.NewReader(session.conn)
			for {
				isInterleaved, packet, channel, err := readInterleaved(r)
				if err != nil {
					session.waitErrChan <- err
					return
				}
				if isInterleaved {
					session.observer.OnInterleavedPacket(packet, int(channel))
				}
			}
		}

		// not over tcp
		// 接收TCP对端关闭FIN信号
		dummy := make([]byte, 1)
		_, err := session.conn.Read(dummy)
		session.waitErrChan <- err
		return
	}

	// 对端支持get_parameter，需要定时向对端发送get_parameter进行保活

	nazalog.Debugf("[%s] start get_parameter timer.", session.UniqueKey)
	var r = bufio.NewReader(session.conn)
	t := time.NewTicker(writeGetParameterIntervalMSec * time.Millisecond)
	defer t.Stop()

	if session.option.OverTCP {
		for {
			select {
			case <-t.C:
				session.cseq++
				if err := session.writeCmd(MethodGetParameter, session.urlCtx.RawURLWithoutUserInfo, nil, ""); err != nil {
					session.waitErrChan <- err
					return
				}
			default:
				// noop
			}

			isInterleaved, packet, channel, err := readInterleaved(r)
			if err != nil {
				session.waitErrChan <- err
				return
			}
			if isInterleaved {
				session.observer.OnInterleavedPacket(packet, int(channel))
			} else {
				if _, err := nazahttp.ReadHTTPResponseMessage(r); err != nil {
					session.waitErrChan <- err
					return
				}
			}
		}
	}

	// not over tcp
	for {
		select {
		case <-t.C:
			session.cseq++
			if _, err := session.writeCmdReadResp(MethodGetParameter, session.urlCtx.RawURLWithoutUserInfo, nil, ""); err != nil {
				session.waitErrChan <- err
				return
			}
		default:
			// noop
		}

	}
}

func (session *ClientCommandSession) connect(rawURL string) (err error) {
	session.rawURL = rawURL

	session.urlCtx, err = base.ParseRTSPURL(rawURL)
	if err != nil {
		return err
	}

	nazalog.Debugf("[%s] > tcp connect.", session.UniqueKey)

	// # 建立连接
	conn, err := net.Dial("tcp", session.urlCtx.HostWithPort)
	if err != nil {
		return err
	}
	session.conn = connection.New(conn, func(option *connection.Option) {
		option.ReadBufSize = readBufSize
	})
	nazalog.Debugf("[%s] < tcp connect. laddr=%s, raddr=%s", session.UniqueKey, conn.LocalAddr().String(), conn.RemoteAddr().String())

	session.observer.OnConnectResult()
	return nil
}
func (session *ClientCommandSession) writeOptions() error {
	ctx, err := session.writeCmdReadResp(MethodOptions, session.urlCtx.RawURLWithoutUserInfo, nil, "")
	if err != nil {
		return err
	}

	methods := ctx.Headers[HeaderPublic]
	if methods == "" {
		return nil
	}
	if strings.Contains(methods, MethodGetParameter) {
		session.methodGetParameterSupported = true
	}
	return nil
}

func (session *ClientCommandSession) writeDescribe() error {
	headers := map[string]string{
		HeaderAccept: HeaderAcceptApplicationSDP,
	}
	ctx, err := session.writeCmdReadResp(MethodDescribe, session.urlCtx.RawURLWithoutUserInfo, headers, "")
	if err != nil {
		return err
	}

	sdpLogicCtx, err := sdp.ParseSDP2LogicContext(ctx.Body)
	if err != nil {
		return err
	}
	session.rawSDP = ctx.Body
	session.sdpLogicCtx = sdpLogicCtx
	session.observer.OnDescribeResponse(session.rawSDP, session.sdpLogicCtx)
	return nil
}

func (session *ClientCommandSession) writeAnnounce() error {
	headers := map[string]string{
		HeaderAccept: HeaderAcceptApplicationSDP,
	}
	_, err := session.writeCmdReadResp(MethodAnnounce, session.urlCtx.RawURLWithoutUserInfo, headers, string(session.rawSDP))
	return err
}

func (session *ClientCommandSession) writeSetup() error {
	if session.sdpLogicCtx.HasVideoAControl() {
		uri := session.sdpLogicCtx.MakeVideoSetupURI(session.urlCtx.RawURLWithoutUserInfo)
		if session.option.OverTCP {
			if err := session.writeOneSetupTCP(uri); err != nil {
				return err
			}
		} else {
			if err := session.writeOneSetup(uri); err != nil {
				return err
			}
		}
	}
	// can't else if
	if session.sdpLogicCtx.HasAudioAControl() {
		uri := session.sdpLogicCtx.MakeAudioSetupURI(session.urlCtx.RawURLWithoutUserInfo)
		if session.option.OverTCP {
			if err := session.writeOneSetupTCP(uri); err != nil {
				return err
			}
		} else {
			if err := session.writeOneSetup(uri); err != nil {
				return err
			}
		}
	}
	return nil
}

func (session *ClientCommandSession) writeOneSetup(setupURI string) error {
	rtpC, lRTPPort, rtcpC, lRTCPPort, err := availUDPConnPool.Acquire2()
	if err != nil {
		return err
	}

	headers := map[string]string{
		HeaderTransport: fmt.Sprintf("RTP/AVP/UDP;unicast;client_port=%d-%d", lRTPPort, lRTCPPort),
	}
	ctx, err := session.writeCmdReadResp(MethodSetup, setupURI, headers, "")
	if err != nil {
		return err
	}

	session.sessionID = strings.Split(ctx.Headers[HeaderSession], ";")[0]

	rRTPPort, rRTCPPort, err := parseServerPort(ctx.Headers[HeaderTransport])
	if err != nil {
		return err
	}

	nazalog.Debugf("[%s] init conn. lRTPPort=%d, lRTCPPort=%d, rRTPPort=%d, rRTCPPort=%d",
		session.UniqueKey, lRTPPort, lRTCPPort, rRTPPort, rRTCPPort)

	rtpConn, err := nazanet.NewUDPConnection(func(option *nazanet.UDPConnectionOption) {
		option.Conn = rtpC
		option.RAddr = net.JoinHostPort(session.urlCtx.Host, fmt.Sprintf("%d", rRTPPort))
		option.MaxReadPacketSize = rtprtcp.MaxRTPRTCPPacketSize
	})
	if err != nil {
		return err
	}

	rtcpConn, err := nazanet.NewUDPConnection(func(option *nazanet.UDPConnectionOption) {
		option.Conn = rtcpC
		option.RAddr = net.JoinHostPort(session.urlCtx.Host, fmt.Sprintf("%d", rRTCPPort))
		option.MaxReadPacketSize = rtprtcp.MaxRTPRTCPPacketSize
	})
	if err != nil {
		return err
	}

	session.observer.OnSetupWithConn(setupURI, rtpConn, rtcpConn)
	return nil
}

func (session *ClientCommandSession) writeOneSetupTCP(setupURI string) error {
	rtpChannel := session.channel
	rtcpChannel := session.channel + 1
	session.channel += 2

	headers := map[string]string{
		HeaderTransport: fmt.Sprintf("RTP/AVP/TCP;unicast;interleaved=%d-%d", rtpChannel, rtcpChannel),
	}
	ctx, err := session.writeCmdReadResp(MethodSetup, setupURI, headers, "")
	if err != nil {
		return err
	}

	session.sessionID = strings.Split(ctx.Headers[HeaderSession], ";")[0]

	// TODO chef: 这里没有解析回传的channel id了，因为我假定了它和request中的是一致的
	session.observer.OnSetupWithChannel(setupURI, rtpChannel, rtcpChannel)
	return nil
}

func (session *ClientCommandSession) writePlay() error {
	headers := map[string]string{
		HeaderRange: "npt=0.000-",
	}
	_, err := session.writeCmdReadResp(MethodPlay, session.urlCtx.RawURLWithoutUserInfo, headers, "")
	return err
}

func (session *ClientCommandSession) writeRecord() error {
	headers := map[string]string{
		HeaderRange: "npt=0.000-",
	}
	_, err := session.writeCmdReadResp(MethodRecord, session.urlCtx.RawURLWithoutUserInfo, headers, "")
	return err
}

func (session *ClientCommandSession) writeCmd(method, uri string, headers map[string]string, body string) error {
	session.cseq++
	if headers == nil {
		headers = make(map[string]string)
	}
	headers[HeaderCSeq] = fmt.Sprintf("%d", session.cseq)
	headers[HeaderUserAgent] = base.LALRTSPPullSessionUA
	if body != "" {
		headers[HeaderContentLength] = fmt.Sprintf("%d", len(body))
	}

	// 鉴权时固定用RawURLWithoutUserInfo
	auth := session.auth.MakeAuthorization(method, session.urlCtx.RawURLWithoutUserInfo)
	if auth != "" {
		headers[HeaderAuthorization] = auth
	}

	if session.sessionID != "" {
		headers[HeaderSession] = session.sessionID
	}

	req := PackRequest(method, uri, headers, body)
	nazalog.Debugf("[%s] > write %s.", session.UniqueKey, method)
	//nazalog.Debugf("[%s] > write %s. req=%s", session.UniqueKey, method, req)
	_, err := session.conn.Write([]byte(req))
	return err
}

// @param headers 可以为nil
// @param body 可以为空
func (session *ClientCommandSession) writeCmdReadResp(method, uri string, headers map[string]string, body string) (ctx nazahttp.HTTPRespMsgCtx, err error) {
	for i := 0; i < 2; i++ {
		if err = session.writeCmd(method, uri, headers, body); err != nil {
			return
		}

		ctx, err = nazahttp.ReadHTTPResponseMessage(session.conn)
		if err != nil {
			return
		}
		nazalog.Debugf("[%s] < read response. version=%s, code=%s, reason=%s, headers=%+v, body=%s",
			session.UniqueKey, ctx.Version, ctx.StatusCode, ctx.Reason, ctx.Headers, string(ctx.Body))

		if ctx.StatusCode != "401" {
			return
		}

		session.auth.FeedWWWAuthenticate(ctx.Headers[HeaderWWWAuthenticate], session.urlCtx.Username, session.urlCtx.Password)
	}

	err = ErrRTSP
	return
}
