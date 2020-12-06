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
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
)

type ServerCommandSessionObserver interface {
	// @brief  Announce阶段回调
	// @return 如果返回false，则表示上层要强制关闭这个推流请求
	OnNewRTSPPubSession(session *PubSession) bool

	// @brief Describe阶段回调
	// @return ok  如果返回false，则表示上层要强制关闭这个拉流请求
	// @return sdp
	OnNewRTSPSubSessionDescribe(session *SubSession) (ok bool, sdp []byte)

	// @brief Describe阶段回调
	// @return ok  如果返回false，则表示上层要强制关闭这个拉流请求
	OnNewRTSPSubSessionPlay(session *SubSession) bool
}

type ServerCommandSession struct {
	UniqueKey string                       // const after ctor
	observer  ServerCommandSessionObserver // const after ctor
	conn      net.Conn

	pubSession *PubSession
	subSession *SubSession
}

func NewServerCommandSession(observer ServerCommandSessionObserver, conn net.Conn) *ServerCommandSession {
	uk := base.GenUniqueKey(base.UKPRTSPServerCommandSession)
	s := &ServerCommandSession{
		UniqueKey: uk,
		observer:  observer,
		conn:      conn,
	}

	nazalog.Infof("[%s] lifecycle new rtmp ServerSession. session=%p, remote addr=%s", uk, s, conn.RemoteAddr().String())
	return s
}

func (s *ServerCommandSession) RunLoop() error {
	return s.runCmdLoop()
}

func (s *ServerCommandSession) Dispose() error {
	return s.conn.Close()
}

// 使用RTSP TCP命令连接，向对端发送RTP数据
func (s *ServerCommandSession) Write(channel int, b []byte) error {
	_, err := s.conn.Write(packInterleaved(channel, b))
	return err
}

func (s *ServerCommandSession) runCmdLoop() error {
	var r = bufio.NewReader(s.conn)

Loop:
	for {
		isInterleaved, packet, channel, err := readInterleaved(r)
		if err != nil {
			nazalog.Errorf("[%s] read interleaved error. err=%+v", s.UniqueKey, err)
			break Loop
		}
		if isInterleaved {
			// TODO chef: 考虑subSession的情况
			if s.pubSession == nil {
				nazalog.Errorf("[%s] read interleaved packet but pubSession not exist.", s.UniqueKey)
				break Loop
			}
			s.pubSession.HandleInterleavedPacket(packet, int(channel))
			continue
		}

		// 读取一个message
		requestCtx, err := nazahttp.ReadHTTPRequestMessage(r)
		if err != nil {
			nazalog.Errorf("[%s] read rtsp message error. err=%+v", s.UniqueKey, err)
			break Loop
		}

		nazalog.Debugf("[%s] read http request. method=%s, uri=%s, headers=%+v, body=%s", s.UniqueKey, requestCtx.Method, requestCtx.URI, requestCtx.Headers, string(requestCtx.Body))

		var handleMsgErr error
		switch requestCtx.Method {
		case MethodOptions:
			// pub, sub
			handleMsgErr = s.handleOptions(requestCtx)
		case MethodAnnounce:
			// pub
			handleMsgErr = s.handleAnnounce(requestCtx)
		case MethodDescribe:
			// sub
			handleMsgErr = s.handleDescribe(requestCtx)
		case MethodSetup:
			// pub, sub
			handleMsgErr = s.handleSetup(requestCtx)
		case MethodRecord:
			// pub
			handleMsgErr = s.handleRecord(requestCtx)
		case MethodPlay:
			// sub
			handleMsgErr = s.handlePlay(requestCtx)
		case MethodTeardown:
			// pub
			handleMsgErr = s.handleTeardown(requestCtx)
			break Loop
		default:
			nazalog.Errorf("[%s] unknown rtsp message. method=%s", s.UniqueKey, requestCtx.Method)
		}
		if handleMsgErr != nil {
			nazalog.Errorf("[%s] handle rtsp message error. err=%+v", s.UniqueKey, handleMsgErr)
			break
		}
	}

	_ = s.conn.Close()
	nazalog.Debugf("[%s] < handleTCPConnect.", s.UniqueKey)

	return nil
}

func (s *ServerCommandSession) handleOptions(requestCtx nazahttp.HTTPReqMsgCtx) error {
	nazalog.Infof("[%s] < R OPTIONS", s.UniqueKey)
	resp := PackResponseOptions(requestCtx.Headers[HeaderFieldCSeq])
	_, err := s.conn.Write([]byte(resp))
	return err
}

func (s *ServerCommandSession) handleAnnounce(requestCtx nazahttp.HTTPReqMsgCtx) error {
	nazalog.Infof("[%s] < R ANNOUNCE", s.UniqueKey)

	presentation, err := parsePresentation(requestCtx.URI)
	if err != nil {
		nazalog.Errorf("[%s] getPresentation failed. uri=%s", s.UniqueKey, requestCtx.URI)
		return err
	}

	sdpLogicCtx, err := sdp.ParseSDP2LogicContext(requestCtx.Body)
	if err != nil {
		nazalog.Errorf("[%s] parse sdp failed. err=%v", s.UniqueKey, err)
		return err
	}

	s.pubSession = NewPubSession(presentation, s)
	nazalog.Infof("[%s] link new PubSession. [%s]", s.UniqueKey, s.pubSession.UniqueKey)
	s.pubSession.InitWithSDP(requestCtx.Body, sdpLogicCtx)

	if ok := s.observer.OnNewRTSPPubSession(s.pubSession); !ok {
		nazalog.Warnf("[%s] force close pubsession.", s.pubSession.UniqueKey)
		return ErrRTSP
	}

	resp := PackResponseAnnounce(requestCtx.Headers[HeaderFieldCSeq])
	_, err = s.conn.Write([]byte(resp))
	return err
}

// 一次SETUP对应一路流（音频或视频）
func (s *ServerCommandSession) handleSetup(requestCtx nazahttp.HTTPReqMsgCtx) error {
	nazalog.Infof("[%s] < R SETUP", s.UniqueKey)

	remoteAddr := s.conn.RemoteAddr().String()
	host, _, _ := net.SplitHostPort(remoteAddr)

	// 是否为interleaved模式
	ts := requestCtx.Headers[HeaderFieldTransport]
	if strings.Contains(ts, TransportFieldInterleaved) {
		rtpChannel, rtcpChannel, err := parseRTPRTCPChannel(ts)
		if err != nil {
			nazalog.Errorf("[%s] parse rtp rtcp channel error. err=%+v", s.UniqueKey, err)
			return err
		}
		if s.pubSession != nil {
			if err := s.pubSession.SetupWithChannel(requestCtx.URI, int(rtpChannel), int(rtcpChannel), remoteAddr); err != nil {
				nazalog.Errorf("[%s] setup channel error. err=%+v", s.UniqueKey, err)
				return err
			}
		} else if s.subSession != nil {
			if err := s.subSession.SetupWithChannel(requestCtx.URI, int(rtpChannel), int(rtcpChannel), remoteAddr); err != nil {
				nazalog.Errorf("[%s] setup channel error. err=%+v", s.UniqueKey, err)
				return err
			}
		} else {
			nazalog.Errorf("[%s] setup but session not exist.", s.UniqueKey)
			return ErrRTSP
		}

		resp := PackResponseSetupTCP(requestCtx.Headers[HeaderFieldCSeq], ts)
		_, err = s.conn.Write([]byte(resp))
		return err
	}

	rRTPPort, rRTCPPort, err := parseClientPort(requestCtx.Headers[HeaderFieldTransport])
	if err != nil {
		nazalog.Errorf("[%s] parseClientPort failed. err=%+v", s.UniqueKey, err)
		return err
	}
	rtpConn, rtcpConn, lRTPPort, lRTCPPort, err := initConnWithClientPort(host, rRTPPort, rRTCPPort)
	if err != nil {
		nazalog.Errorf("[%s] initConnWithClientPort failed. err=%+v", s.UniqueKey, err)
		return err
	}

	if s.pubSession != nil {
		if err = s.pubSession.SetupWithConn(requestCtx.URI, rtpConn, rtcpConn); err != nil {
			nazalog.Errorf("[%s] setup conn error. err=%+v", s.UniqueKey, err)
			return err
		}
	} else if s.subSession != nil {
		if err = s.subSession.SetupWithConn(requestCtx.URI, rtpConn, rtcpConn); err != nil {
			nazalog.Errorf("[%s] setup conn error. err=%+v", s.UniqueKey, err)
			return err
		}
	} else {
		nazalog.Errorf("[%s] setup but session not exist.", s.UniqueKey)
		return ErrRTSP
	}

	resp := PackResponseSetup(requestCtx.Headers[HeaderFieldCSeq], rRTPPort, rRTCPPort, lRTPPort, lRTCPPort)
	_, err = s.conn.Write([]byte(resp))
	return err
}

func (s *ServerCommandSession) handleDescribe(requestCtx nazahttp.HTTPReqMsgCtx) error {
	nazalog.Infof("[%s] < R DESCRIBE", s.UniqueKey)

	presentation, err := parsePresentation(requestCtx.URI)
	if err != nil {
		nazalog.Errorf("[%s] parsePresentation failed. uri=%s", s.UniqueKey, requestCtx.URI)
		return err
	}

	s.subSession = NewSubSession(presentation, s)
	nazalog.Infof("[%s] link new SubSession. [%s]", s.UniqueKey, s.subSession.UniqueKey)
	ok, rawSDP := s.observer.OnNewRTSPSubSessionDescribe(s.subSession)
	if !ok {
		nazalog.Warnf("[%s] force close subSession.", s.UniqueKey)
		return ErrRTSP
	}

	ctx, _ := sdp.ParseSDP2LogicContext(rawSDP)
	s.subSession.InitWithSDP(rawSDP, ctx)

	resp := PackResponseDescribe(requestCtx.Headers[HeaderFieldCSeq], string(rawSDP))
	_, err = s.conn.Write([]byte(resp))
	return err
}

func (s *ServerCommandSession) handleRecord(requestCtx nazahttp.HTTPReqMsgCtx) error {
	nazalog.Infof("[%s] < R RECORD", s.UniqueKey)
	resp := PackResponseRecord(requestCtx.Headers[HeaderFieldCSeq])
	_, err := s.conn.Write([]byte(resp))
	return err
}

func (s *ServerCommandSession) handlePlay(requestCtx nazahttp.HTTPReqMsgCtx) error {
	nazalog.Infof("[%s] < R PLAY", s.UniqueKey)
	if ok := s.observer.OnNewRTSPSubSessionPlay(s.subSession); !ok {
		return ErrRTSP
	}
	resp := PackResponsePlay(requestCtx.Headers[HeaderFieldCSeq])
	_, err := s.conn.Write([]byte(resp))
	return err
}

func (s *ServerCommandSession) handleTeardown(requestCtx nazahttp.HTTPReqMsgCtx) error {
	nazalog.Infof("[%s] < R TEARDOWN", s.UniqueKey)
	resp := PackResponseTeardown(requestCtx.Headers[HeaderFieldCSeq])
	_, err := s.conn.Write([]byte(resp))
	return err
}

// 从uri中解析stream name
func parsePresentation(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", err
	}
	if len(u.Path) == 0 {
		return "", ErrRTSP
	}
	items := strings.Split(u.Path[1:], "/")
	switch len(items) {
	case 0:
		return "", ErrRTSP
	case 1:
		return items[0], nil
	default:
		// TODO chef: 是否应该根据SDP的内容来决定过滤的内容
		if strings.Contains(items[len(items)-1], "streamid=") {
			return items[len(items)-2], nil
		} else {
			return items[len(items)-1], nil
		}
	}
}

// 从setup消息的header中解析rtp rtcp channel
func parseRTPRTCPChannel(setupTransport string) (rtp, rtcp uint16, err error) {
	return parseTransport(setupTransport, TransportFieldInterleaved)
}

// 从setup消息的header中解析rtp rtcp 端口
func parseClientPort(setupTransport string) (rtp, rtcp uint16, err error) {
	return parseTransport(setupTransport, TransportFieldClientPort)
}

func parseServerPort(setupTransport string) (rtp, rtcp uint16, err error) {
	return parseTransport(setupTransport, TransportFieldServerPort)
}

func parseTransport(setupTransport string, key string) (first, second uint16, err error) {
	var clientPort string
	items := strings.Split(setupTransport, ";")
	for _, item := range items {
		if strings.HasPrefix(item, key) {
			kv := strings.Split(item, "=")
			if len(kv) != 2 {
				continue
			}
			clientPort = kv[1]
		}
	}
	items = strings.Split(clientPort, "-")
	if len(items) != 2 {
		return 0, 0, ErrRTSP
	}
	iFirst, err := strconv.Atoi(items[0])
	if err != nil {
		return 0, 0, err
	}
	iSecond, err := strconv.Atoi(items[1])
	if err != nil {
		return 0, 0, err
	}
	return uint16(iFirst), uint16(iSecond), err
}
