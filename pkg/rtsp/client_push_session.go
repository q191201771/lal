// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/nazanet"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
)

// TODO chef:
// - 将push和pull中重复的代码抽象出来
// - 将push和sub中重复的代码抽象出来
// - push有的功能没实现，需要参考pull和sub

const (
	pushReadBufSize = 256
)

type PushSessionOption struct {
	PushTimeoutMS int
	OverTCP       bool
}

var defaultPushSessionOption = PushSessionOption{
	PushTimeoutMS: 10000,
	OverTCP:       false,
}

type PushSession struct {
	UniqueKey string
	option    PushSessionOption

	CmdConn connection.Connection

	cseq      int
	sessionID string
	channel   int

	rawURL string
	urlCtx base.URLContext

	waitErrChan chan error

	methodGetParameterSupported bool

	auth Auth

	rawSDP      []byte
	sdpLogicCtx sdp.LogicContext

	audioRTPConn  *nazanet.UDPConnection
	videoRTPConn  *nazanet.UDPConnection
	audioRTCPConn *nazanet.UDPConnection
	videoRTCPConn *nazanet.UDPConnection
}

type ModPushSessionOption func(option *PushSessionOption)

func NewPushSession(modOptions ...ModPushSessionOption) *PushSession {
	option := defaultPushSessionOption
	for _, fn := range modOptions {
		fn(&option)
	}

	uk := base.GenUniqueKey(base.UKPRTSPPushSession)
	s := &PushSession{
		UniqueKey:   uk,
		option:      option,
		waitErrChan: make(chan error, 1),
	}
	nazalog.Infof("[%s] lifecycle new rtsp PushSession. session=%p", uk, s)
	return s
}

func (session *PushSession) Push(rawURL string, rawSDP []byte, sdpLogicCtx sdp.LogicContext) error {
	nazalog.Debugf("[%s] push. url=%s", session.UniqueKey, rawURL)

	session.rawSDP = rawSDP
	session.sdpLogicCtx = sdpLogicCtx

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if session.option.PushTimeoutMS == 0 {
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(session.option.PushTimeoutMS)*time.Millisecond)
	}
	defer cancel()
	return session.pushContext(ctx, rawURL)
}

func (session *PushSession) Wait() <-chan error {
	return session.waitErrChan
}

func (session *PushSession) WriteRTPPacket(packet rtprtcp.RTPPacket) {
	// 发送数据时，保证和sdp的原始类型对应
	t := int(packet.Header.PacketType)
	if session.sdpLogicCtx.IsAudioPayloadTypeOrigin(t) {
		if session.audioRTPConn != nil {
			_ = session.audioRTPConn.Write(packet.Raw)
		}
	} else if session.sdpLogicCtx.IsVideoPayloadTypeOrigin(t) {
		if session.videoRTPConn != nil {
			_ = session.videoRTPConn.Write(packet.Raw)
		}
	} else {
		nazalog.Errorf("[%s] write rtp packet but type invalid. type=%d", session.UniqueKey, t)
	}
}

func (session *PushSession) pushContext(ctx context.Context, rawURL string) error {
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

		if err := session.writeAnnounce(); err != nil {
			errChan <- err
			return
		}

		if err := session.writeSetup(); err != nil {
			errChan <- err
			return
		}

		if err := session.writeRecord(); err != nil {
			errChan <- err
			return
		}

	}()

	return nil
}

func (session *PushSession) connect(rawURL string) (err error) {
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

	nazalog.Debugf("[%s] < tcp connect. laddr=%s", session.UniqueKey, conn.LocalAddr().String())

	// TODO
	return nil
}

func (session *PushSession) writeOptions() error {
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

func (session *PushSession) writeAnnounce() error {
	session.cseq++
	auth := session.auth.MakeAuthorization(MethodDescribe, session.urlCtx.RawURLWithoutUserInfo)
	req := PackRequestAnnounce(session.urlCtx.RawURLWithoutUserInfo, session.cseq, string(session.rawSDP), auth)
	nazalog.Debugf("[%s] > write announce.", session.UniqueKey)
	if _, err := session.CmdConn.Write([]byte(req)); err != nil {
		return err
	}
	ctx, err := nazahttp.ReadHTTPResponseMessage(session.CmdConn)
	if err != nil {
		return err
	}
	nazalog.Debugf("[%s] < read response. code=%s, body=%s", session.UniqueKey, ctx.StatusCode, string(ctx.Body))

	return nil
}

func (session *PushSession) writeSetup() error {
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

func (session *PushSession) writeOneSetup(setupURI string) error {
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

	if err := session.setupWithConn(setupURI, rtpConn, rtcpConn); err != nil {
		return err
	}

	return nil
}

func (session *PushSession) writeRecord() error {
	session.cseq++
	auth := session.auth.MakeAuthorization(MethodRecord, session.urlCtx.RawURLWithoutUserInfo)
	req := PackRequestRecord(session.urlCtx.RawURLWithoutUserInfo, session.cseq, session.sessionID, auth)
	nazalog.Debugf("[%s] > write record.", session.UniqueKey)
	if _, err := session.CmdConn.Write([]byte(req)); err != nil {
		return err
	}
	ctx, err := nazahttp.ReadHTTPResponseMessage(session.CmdConn)
	if err != nil {
		return err
	}
	nazalog.Debugf("[%s] < read response. code=%s, body=%s", session.UniqueKey, ctx.StatusCode, string(ctx.Body))

	return nil
}

func (session *PushSession) setupWithConn(uri string, rtpConn, rtcpConn *nazanet.UDPConnection) error {
	if session.sdpLogicCtx.IsAudioURI(uri) {
		session.audioRTPConn = rtpConn
		session.audioRTCPConn = rtcpConn
	} else if session.sdpLogicCtx.IsVideoURI(uri) {
		session.videoRTPConn = rtpConn
		session.videoRTCPConn = rtcpConn
	} else {
		return ErrRTSP
	}

	go rtpConn.RunLoop(session.onReadUDPPacket)
	go rtcpConn.RunLoop(session.onReadUDPPacket)

	return nil
}

func (session *PushSession) onReadUDPPacket(b []byte, rAddr *net.UDPAddr, err error) bool {
	// TODO chef: impl me
	//nazalog.Errorf("[%s] SubSession::onReadUDPPacket. %s", s.UniqueKey, hex.Dump(b))
	return true
}

func (session *PushSession) writeOneSetupTCP(setupURI string) error {
	panic("not impl")
	return nil
}

func (session *PushSession) handleOptionMethods(ctx nazahttp.HTTPRespMsgCtx) {
	methods := ctx.Headers["Public"]
	if methods == "" {
		return
	}

	if strings.Contains(methods, MethodGetParameter) {
		session.methodGetParameterSupported = true
	}
}

func (session *PushSession) handleAuth(ctx nazahttp.HTTPRespMsgCtx) error {
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
