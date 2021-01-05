// Copyright 2020, Chef.  All rights reserved.
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

const (
	pullReadBufSize              = 256
	writeGetParameterIntervalSec = 10
)

type PullSessionObserver interface {
	BaseInSessionObserver
}

type PullSessionOption struct {
	// 从调用Pull函数，到接收音视频数据的前一步，也即收到rtsp play response的超时时间
	// 如果为0，则没有超时时间
	PullTimeoutMS int

	OverTCP bool // 是否使用interleaved模式，也即是否通过rtsp command tcp连接传输rtp/rtcp数据
}

var defaultPullSessionOption = PullSessionOption{
	PullTimeoutMS: 10000,
	OverTCP:       false,
}

type PullSession struct {
	baseInSession *BaseInSession
	UniqueKey     string            // const after ctor
	option        PullSessionOption // const after ctor

	CmdConn connection.Connection

	cseq      int
	sessionID string
	channel   int

	rawURL string
	urlCtx base.URLContext

	waitErrChan chan error

	methodGetParameterSupported bool

	auth Auth
}

type ModPullSessionOption func(option *PullSessionOption)

func NewPullSession(observer PullSessionObserver, modOptions ...ModPullSessionOption) *PullSession {
	option := defaultPullSessionOption
	for _, fn := range modOptions {
		fn(&option)
	}

	uk := base.GenUniqueKey(base.UKPRTSPPullSession)
	s := &PullSession{
		UniqueKey:   uk,
		option:      option,
		waitErrChan: make(chan error, 1),
	}
	baseInSession := &BaseInSession{
		UniqueKey: uk,
		stat: base.StatSession{
			Protocol:  base.ProtocolRTSP,
			SessionID: uk,
			StartTime: time.Now().Format("2006-01-02 15:04:05.999"),
		},
		observer:   observer,
		cmdSession: s,
	}
	s.baseInSession = baseInSession
	nazalog.Infof("[%s] lifecycle new rtsp PullSession. session=%p", uk, s)
	return s
}

// 如果没有错误发生，阻塞直到接收音视频数据的前一步，也即收到rtsp play response
func (session *PullSession) Pull(rawURL string) error {
	nazalog.Debugf("[%s] pull. url=%s", session.UniqueKey, rawURL)

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if session.option.PullTimeoutMS == 0 {
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(session.option.PullTimeoutMS)*time.Millisecond)
	}
	defer cancel()
	return session.pullContext(ctx, rawURL)
}

// Pull成功后，调用该函数，可阻塞直到拉流结束
func (session *PullSession) Wait() <-chan error {
	return session.waitErrChan
}

func (session *PullSession) Write(channel int, b []byte) error {
	return nil
}

func (session *PullSession) Dispose() error {
	return nil
}

func (session *PullSession) AppName() string {
	return session.urlCtx.PathWithoutLastItem
}

func (session *PullSession) StreamName() string {
	return session.urlCtx.LastItemOfPath
}

func (session *PullSession) RawQuery() string {
	return session.urlCtx.RawQuery
}

// @return 注意，`RemoteAddr`字段返回的是RTSP command TCP连接的地址
func (session *PullSession) GetStat() base.StatSession {
	return session.baseInSession.GetStat()
}

func (session *PullSession) UpdateStat(interval uint32) {
	session.baseInSession.UpdateStat(interval)
}

func (session *PullSession) IsAlive() (readAlive, writeAlive bool) {
	return session.baseInSession.IsAlive()
}

func (session *PullSession) GetSDP() ([]byte, sdp.LogicContext) {
	return session.baseInSession.GetSDP()
}

func (session *PullSession) pullContext(ctx context.Context, rawURL string) error {
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

		if err := session.writeDescribe(); err != nil {
			errChan <- err
			return
		}

		if err := session.writeSetup(); err != nil {
			errChan <- err
			return
		}

		if err := session.writePlay(); err != nil {
			errChan <- err
			return
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

func (session *PullSession) runReadLoop() {
	if !session.methodGetParameterSupported {
		// TCP模式，需要收取数据进行处理
		if session.option.OverTCP {
			var r = bufio.NewReader(session.CmdConn)
			for {
				isInterleaved, packet, channel, err := readInterleaved(r)
				if err != nil {
					session.waitErrChan <- err
					return
				}
				if isInterleaved {
					session.baseInSession.HandleInterleavedPacket(packet, int(channel))
				}
			}
		}

		// not over tcp
		// 接收TCP对端关闭FIN信号
		dummy := make([]byte, 1)
		_, err := session.CmdConn.Read(dummy)
		session.waitErrChan <- err
		return
	}

	// 对端支持get_parameter，需要定时向对端发送get_parameter进行保活

	var r = bufio.NewReader(session.CmdConn)
	t := time.NewTicker(writeGetParameterIntervalSec * time.Millisecond)
	defer t.Stop()

	if session.option.OverTCP {
		for {
			select {
			case <-t.C:
				session.cseq++
				req := PackRequestGetParameter(session.urlCtx.RawURLWithoutUserInfo, session.cseq, session.sessionID)
				if _, err := session.CmdConn.Write([]byte(req)); err != nil {
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
				session.baseInSession.HandleInterleavedPacket(packet, int(channel))
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
			req := PackRequestGetParameter(session.urlCtx.RawURLWithoutUserInfo, session.cseq, session.sessionID)
			if _, err := session.CmdConn.Write([]byte(req)); err != nil {
				session.waitErrChan <- err
				return
			}

			if _, err := nazahttp.ReadHTTPResponseMessage(r); err != nil {
				session.waitErrChan <- err
				return
			}
		default:
			// noop
		}

	}
}

func (session *PullSession) connect(rawURL string) (err error) {
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
	session.CmdConn = connection.New(conn, func(option *connection.Option) {
		option.ReadBufSize = pullReadBufSize
	})
	session.baseInSession.stat.RemoteAddr = conn.RemoteAddr().String()
	return nil
}

func (session *PullSession) writeOptions() error {
	session.cseq++
	req := PackRequestOptions(session.urlCtx.RawURLWithoutUserInfo, session.cseq, "")
	nazalog.Debugf("[%s] > write options.", session.UniqueKey)
	if _, err := session.CmdConn.Write([]byte(req)); err != nil {
		return err
	}
	ctx, err := nazahttp.ReadHTTPResponseMessage(session.CmdConn)
	if err != nil {
		return err
	}
	nazalog.Debugf("[%s] < read response. %s", session.UniqueKey, ctx.StatusCode)

	session.handleOptionMethods(ctx)
	if err := session.handleAuth(ctx); err != nil {
		return err
	}

	return nil
}

func (session *PullSession) writeDescribe() error {
	session.cseq++
	auth := session.auth.MakeAuthorization(MethodDescribe, session.urlCtx.RawURLWithoutUserInfo)
	req := PackRequestDescribe(session.urlCtx.RawURLWithoutUserInfo, session.cseq, auth)
	nazalog.Debugf("[%s] > write describe.", session.UniqueKey)
	if _, err := session.CmdConn.Write([]byte(req)); err != nil {
		return err
	}
	ctx, err := nazahttp.ReadHTTPResponseMessage(session.CmdConn)
	if err != nil {
		return err
	}
	nazalog.Debugf("[%s] < read response. code=%s, body=%s", session.UniqueKey, ctx.StatusCode, string(ctx.Body))

	sdpLogicCtx, err := sdp.ParseSDP2LogicContext(ctx.Body)
	if err != nil {
		return err
	}

	session.baseInSession.InitWithSDP(ctx.Body, sdpLogicCtx)

	return nil
}

func (session *PullSession) writeSetup() error {
	if session.baseInSession.sdpLogicCtx.HasVideoAControl() {
		uri := session.baseInSession.sdpLogicCtx.MakeVideoSetupURI(session.urlCtx.RawURLWithoutUserInfo)
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
	if session.baseInSession.sdpLogicCtx.HasAudioAControl() {
		uri := session.baseInSession.sdpLogicCtx.MakeAudioSetupURI(session.urlCtx.RawURLWithoutUserInfo)
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

func (session *PullSession) writeOneSetup(setupURI string) error {
	rtpC, rtpPort, rtcpC, rtcpPort, err := availUDPConnPool.Acquire2()
	if err != nil {
		return err
	}

	session.cseq++
	auth := session.auth.MakeAuthorization(MethodSetup, session.urlCtx.RawURLWithoutUserInfo)
	req := PackRequestSetup(setupURI, session.cseq, int(rtpPort), int(rtcpPort), session.sessionID, auth)
	nazalog.Debugf("[%s] > write setup.", session.UniqueKey)
	if _, err := session.CmdConn.Write([]byte(req)); err != nil {
		return err
	}
	ctx, err := nazahttp.ReadHTTPResponseMessage(session.CmdConn)
	if err != nil {
		return err
	}
	nazalog.Debugf("[%s] < read response. code=%s, ctx=%+v", session.UniqueKey, ctx.StatusCode, ctx)

	session.sessionID = strings.Split(ctx.Headers[HeaderFieldSession], ";")[0]

	srvRTPPort, srvRTCPPort, err := parseServerPort(ctx.Headers[HeaderFieldTransport])
	if err != nil {
		return err
	}

	rtpConn, err := nazanet.NewUDPConnection(func(option *nazanet.UDPConnectionOption) {
		option.Conn = rtpC
		option.RAddr = net.JoinHostPort(session.urlCtx.Host, fmt.Sprintf("%d", srvRTPPort))
		option.MaxReadPacketSize = rtprtcp.MaxRTPRTCPPacketSize
	})
	if err != nil {
		return err
	}

	rtcpConn, err := nazanet.NewUDPConnection(func(option *nazanet.UDPConnectionOption) {
		option.Conn = rtcpC
		option.RAddr = net.JoinHostPort(session.urlCtx.Host, fmt.Sprintf("%d", srvRTCPPort))
		option.MaxReadPacketSize = rtprtcp.MaxRTPRTCPPacketSize
	})
	if err != nil {
		return err
	}

	if err := session.baseInSession.SetupWithConn(setupURI, rtpConn, rtcpConn); err != nil {
		return err
	}

	return nil
}

func (session *PullSession) writeOneSetupTCP(setupURI string) error {
	rtpChannel := session.channel
	rtcpChannel := session.channel + 1
	session.channel += 2

	session.cseq++
	auth := session.auth.MakeAuthorization(MethodSetup, session.urlCtx.RawURLWithoutUserInfo)
	req := PackRequestSetupTCP(setupURI, session.cseq, rtpChannel, rtcpChannel, session.sessionID, auth)
	nazalog.Debugf("[%s] > write setup.", session.UniqueKey)
	if _, err := session.CmdConn.Write([]byte(req)); err != nil {
		return err
	}
	ctx, err := nazahttp.ReadHTTPResponseMessage(session.CmdConn)
	if err != nil {
		return err
	}
	nazalog.Debugf("[%s] < read response. code=%s, ctx=%+v", session.UniqueKey, ctx.StatusCode, ctx)

	session.sessionID = strings.Split(ctx.Headers[HeaderFieldSession], ";")[0]

	// TODO chef: 这里没有解析回传的channel id了，因为我假定了它和request中的是一致的

	if err := session.baseInSession.SetupWithChannel(setupURI, rtpChannel, rtcpChannel); err != nil {
		return err
	}

	return nil
}

func (session *PullSession) writePlay() error {
	session.baseInSession.WriteRTPRTCPDummy()

	session.cseq++
	auth := session.auth.MakeAuthorization(MethodPlay, session.urlCtx.RawURLWithoutUserInfo)
	req := PackRequestPlay(session.urlCtx.RawURLWithoutUserInfo, session.cseq, session.sessionID, auth)
	nazalog.Debugf("[%s] > write play.", session.UniqueKey)
	if _, err := session.CmdConn.Write([]byte(req)); err != nil {
		return err
	}
	ctx, err := nazahttp.ReadHTTPResponseMessage(session.CmdConn)
	if err != nil {
		return err
	}
	nazalog.Debugf("[%s] < read response. %s", session.UniqueKey, ctx.StatusCode)
	return nil
}

func (session *PullSession) handleOptionMethods(ctx nazahttp.HTTPRespMsgCtx) {
	methods := ctx.Headers["Public"]
	if methods == "" {
		return
	}

	if strings.Contains(methods, MethodGetParameter) {
		session.methodGetParameterSupported = true
	}
}

func (session *PullSession) handleAuth(ctx nazahttp.HTTPRespMsgCtx) error {
	if ctx.Headers[HeaderWWWAuthenticate] == "" {
		return nil
	}

	session.auth.FeedWWWAuthenticate(ctx.Headers[HeaderWWWAuthenticate], session.urlCtx.Username, session.urlCtx.Password)
	auth := session.auth.MakeAuthorization(MethodOptions, session.urlCtx.RawURLWithoutUserInfo)

	session.cseq++
	req := PackRequestOptions(session.urlCtx.RawURLWithoutUserInfo, session.cseq, auth)
	nazalog.Debugf("[%s] > write options.", session.UniqueKey)
	if _, err := session.CmdConn.Write([]byte(req)); err != nil {
		return err
	}
	ctx, err := nazahttp.ReadHTTPResponseMessage(session.CmdConn)
	if err != nil {
		return err
	}
	nazalog.Debugf("[%s] < read response. %s", session.UniqueKey, ctx.StatusCode)
	return nil
}
