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
	pullReadBufSize = 256
)

type PullSessionObserver interface {
	BaseInSessionObserver
}

type PullSessionOption struct {
	PullTimeoutMS int // 从调用Pull函数，到收到rtsp play response（接收音视频数据的前一步）的超时时间
}

var defaultPullSessionOption = PullSessionOption{
	PullTimeoutMS: 10000,
}

type PullSession struct {
	baseInSession *BaseInSession
	UniqueKey     string            // const after ctor
	option        PullSessionOption // const after ctor

	CmdConn connection.Connection

	cseq      int
	sessionID string

	rawURL string
	urlCtx base.URLContext
}

type ModPullSessionOption func(option *PullSessionOption)

func NewPullSession(observer PullSessionObserver, modOptions ...ModPullSessionOption) *PullSession {
	option := defaultPullSessionOption
	for _, fn := range modOptions {
		fn(&option)
	}

	uk := base.GenUniqueKey(base.UKPRTSPPullSession)
	s := &PullSession{
		UniqueKey: uk,
		option:    option,
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

func (session *PullSession) Pull(rawURL string) error {
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

	dummy := make([]byte, 1) // 用于接收TCP对端关闭FIN信号
	_, err := session.CmdConn.Read(dummy)
	return err
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
	req := PackRequestOptions(session.rawURL, session.cseq)
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

func (session *PullSession) writeDescribe() error {
	session.cseq++
	req := PackRequestDescribe(session.rawURL, session.cseq)
	nazalog.Debugf("[%s] > write describe.", session.UniqueKey)
	if _, err := session.CmdConn.Write([]byte(req)); err != nil {
		return err
	}
	ctx, err := nazahttp.ReadHTTPResponseMessage(session.CmdConn)
	if err != nil {
		return err
	}
	nazalog.Debugf("[%s] < read response. code=%s, body=%s", session.UniqueKey, ctx.StatusCode, string(ctx.Body))

	sdpCtx, err := sdp.ParseSDP2LogicContext(ctx.Body)
	if err != nil {
		return err
	}

	session.baseInSession.InitWithSDP(ctx.Body, sdpCtx)

	return nil
}

func (session *PullSession) writeSetup() error {
	if session.baseInSession.sdpLogicCtx.VideoAControl != "" {
		if err := session.writeOneSetup(session.baseInSession.sdpLogicCtx.VideoAControl); err != nil {
			return err
		}
	}
	if session.baseInSession.sdpLogicCtx.AudioAControl != "" {
		if err := session.writeOneSetup(session.baseInSession.sdpLogicCtx.AudioAControl); err != nil {
			return err
		}
	}
	return nil
}

func (session *PullSession) writeOneSetup(aControl string) error {
	setupURI := fmt.Sprintf("%s/%s", session.rawURL, aControl)
	rtpC, rtpPort, rtcpC, rtcpPort, err := availUDPConnPool.Acquire2()
	if err != nil {
		return err
	}

	session.cseq++
	req := PackRequestSetup(setupURI, session.cseq, session.sessionID, int(rtpPort), int(rtcpPort))
	nazalog.Debugf("[%s] > write setup.", session.UniqueKey)
	if _, err := session.CmdConn.Write([]byte(req)); err != nil {
		return err
	}
	ctx, err := nazahttp.ReadHTTPResponseMessage(session.CmdConn)
	if err != nil {
		return err
	}
	nazalog.Debugf("[%s] < read response. code=%s, ctx=%+v", session.UniqueKey, ctx.StatusCode, ctx)

	session.sessionID = ctx.Headers[HeaderFieldSession]

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

func (session *PullSession) writePlay() error {
	session.cseq++
	req := PackRequestPlay(session.rawURL, session.cseq, session.sessionID)
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
