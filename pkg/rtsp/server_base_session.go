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
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
)

// TODO chef: 没使用到可以删了
const (
	ServerBaseSessionStageInitial uint32 = iota + 1
	ServerBaseSessionStageAnnounce
	ServerBaseSessionStageDescribe
	ServerBaseSessionStageRecord
	ServerBaseSessionStagePlay
)

type ServerBaseSessionObserver interface {
	// @brief  Announce阶段回调
	// @return 如果返回false，则表示上层要强制关闭这个推流请求
	OnNewRTSPPubSession(session *PubSession) bool

	OnDelRTSPPubSession(session *PubSession)

	// @brief Describe阶段回调
	// @return ok  如果返回false，则表示上层要强制关闭这个拉流请求
	// @return sdp
	OnNewRTSPSubSessionDescribe(session *SubSession) (ok bool, sdp []byte)

	// @brief Describe阶段回调
	// @return ok  如果返回false，则表示上层要强制关闭这个拉流请求
	OnNewRTSPSubSessionPlay(session *SubSession) bool

	OnDelRTSPSubSession(session *SubSession)
}

type ServerBaseSession struct {
	UniqueKey string                    // const after ctor
	observer  ServerBaseSessionObserver // const after ctor
	conn      net.Conn

	pubSession *PubSession
	subSession *SubSession

	stage uint32 // atomic
}

func NewServerBaseSession(observer ServerBaseSessionObserver, conn net.Conn) *ServerBaseSession {
	uk := base.GenUniqueKey(base.UKPRTSPServerBaseSession)
	s := &ServerBaseSession{
		UniqueKey: uk,
		observer:  observer,
		conn:      conn,
		stage:     ServerBaseSessionStageInitial,
	}

	nazalog.Infof("[%s] lifecycle new rtmp ServerSession. session=%p, remote addr=%s", uk, s, conn.RemoteAddr().String())
	return s
}

func (s *ServerBaseSession) RunLoop() error {
	err := s.runCmdLoop()

	if s.pubSession != nil {
		s.observer.OnDelRTSPPubSession(s.pubSession)
	} else if s.subSession != nil {
		s.observer.OnDelRTSPSubSession(s.subSession)
	}

	return err
}

func (s *ServerBaseSession) runCmdLoop() error {
	var (
		r         = bufio.NewReader(s.conn)
		rtpLenBuf = make([]byte, 2)
	)

Loop:
	for {
		// rfc2326 10.12 Embedded (Interleaved) Binary Data
		// 判断是否interleaved
		flag, err := r.ReadByte()
		if err != nil {
			nazalog.Errorf("[%s] read error. err=%+v", s.UniqueKey, err)
			break Loop
		}
		if flag == Interleaved {
			if s.pubSession == nil {
				nazalog.Errorf("[%s] read interleaved packet but pubSession not exist.", s.UniqueKey)
				break Loop
			}
			// channel
			channel, err := r.ReadByte()
			if err != nil {
				nazalog.Errorf("[%s] read error. err=%+v", s.UniqueKey, err)
				break Loop
			}
			_, err = io.ReadFull(r, rtpLenBuf)
			if err != nil {
				nazalog.Errorf("[%s] read error. err=%+v", s.UniqueKey, err)
				break Loop
			}
			rtpLen := int(bele.BEUint16(rtpLenBuf))
			// TODO chef: 这里为了安全性，应该检查大小
			rtpBuf := make([]byte, rtpLen)
			_, err = io.ReadFull(r, rtpBuf[:rtpLen])
			if err != nil {
				nazalog.Errorf("[%s] read error. err=%+v", s.UniqueKey, err)
				break Loop
			}
			s.pubSession.HandleInterleavedRTPPacket(rtpBuf[:rtpLen], int(channel))
			continue
		}

		// 不是interleaved，将flag这个字节返还
		err = r.UnreadByte()
		if err != nil {
			nazalog.Errorf("[%s] read error. err=%+v", s.UniqueKey, err)
			break Loop
		}

		// 读取一个message
		requestCtx, err := nazahttp.ReadHTTPRequest(r)
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

func (s *ServerBaseSession) handleOptions(requestCtx nazahttp.HTTPRequestCtx) error {
	nazalog.Infof("[%s] < R OPTIONS", s.UniqueKey)
	resp := PackResponseOptions(requestCtx.Headers[HeaderFieldCSeq])
	_, err := s.conn.Write([]byte(resp))
	return err
}

func (s *ServerBaseSession) handleAnnounce(requestCtx nazahttp.HTTPRequestCtx) error {
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

	s.pubSession = NewPubSession(presentation)
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
func (s *ServerBaseSession) handleSetup(requestCtx nazahttp.HTTPRequestCtx) error {
	nazalog.Infof("[%s] < R SETUP", s.UniqueKey)

	remoteAddr := s.conn.RemoteAddr().String()
	host, _, _ := net.SplitHostPort(remoteAddr)

	// 是否为interleaved模式
	ts := requestCtx.Headers[HeaderFieldTransport]
	if strings.Contains(ts, TransportFieldInterleaved) {
		if s.pubSession == nil {
			nazalog.Errorf("[%s] read interleaved setup but pubSession not exist.", s.UniqueKey)
			return ErrRTSP
		}
		rtpChannel, rtcpChannel, err := parseRTPRTCPChannel(ts)
		if err != nil {
			nazalog.Errorf("[%s] parse rtp rtcp channel error. err=%+v", s.UniqueKey, err)
			return err
		}
		if err := s.pubSession.SetupWithChannel(requestCtx.URI, int(rtpChannel), int(rtcpChannel), remoteAddr); err != nil {
			nazalog.Errorf("[%s] setup channel error. err=%+v", s.UniqueKey, err)
			return err
		}

		resp := PackResponseSetupTCP(requestCtx.Headers[HeaderFieldCSeq], ts)
		_, err = s.conn.Write([]byte(resp))
		return err
	}

	rRTPPort, rRTCPPort, err := parseRTPRTCPPort(requestCtx.Headers[HeaderFieldTransport])
	if err != nil {
		nazalog.Errorf("[%s] parseRTPRTCPPort failed. err=%+v", s.UniqueKey, err)
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
		if err = s.subSession.Setup(requestCtx.URI, rtpConn, rtcpConn); err != nil {
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

func (s *ServerBaseSession) handleDescribe(requestCtx nazahttp.HTTPRequestCtx) error {
	nazalog.Infof("[%s] < R DESCRIBE", s.UniqueKey)

	presentation, err := parsePresentation(requestCtx.URI)
	if err != nil {
		nazalog.Errorf("[%s] parsePresentation failed. uri=%s", s.UniqueKey, requestCtx.URI)
		return err
	}

	s.subSession = NewSubSession(presentation)
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

func (s *ServerBaseSession) handleRecord(requestCtx nazahttp.HTTPRequestCtx) error {
	nazalog.Infof("[%s] < R RECORD", s.UniqueKey)
	resp := PackResponseRecord(requestCtx.Headers[HeaderFieldCSeq])
	_, err := s.conn.Write([]byte(resp))
	return err
}

func (s *ServerBaseSession) handlePlay(requestCtx nazahttp.HTTPRequestCtx) error {
	nazalog.Infof("[%s] < R PLAY", s.UniqueKey)
	if ok := s.observer.OnNewRTSPSubSessionPlay(s.subSession); !ok {
		return ErrRTSP
	}
	resp := PackResponsePlay(requestCtx.Headers[HeaderFieldCSeq])
	_, err := s.conn.Write([]byte(resp))
	return err
}

func (s *ServerBaseSession) handleTeardown(requestCtx nazahttp.HTTPRequestCtx) error {
	nazalog.Infof("[%s] < R TEARDOWN", s.UniqueKey)
	resp := PackResponseTeardown(requestCtx.Headers[HeaderFieldCSeq])
	_, err := s.conn.Write([]byte(resp))
	return err
}

func (s *ServerBaseSession) setStage(stage uint32) {
	atomic.StoreUint32(&s.stage, stage)
}

func (s *ServerBaseSession) getStage() uint32 {
	return atomic.LoadUint32(&s.stage)
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
func parseRTPRTCPPort(setupTransport string) (rtp, rtcp uint16, err error) {
	return parseTransport(setupTransport, TransportFieldClientPort)
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

// 传入远端IP，RTPPort，RTCPPort，创建两个对应的RTP和RTCP的UDP连接对象，以及对应的本端端口
func initConnWithClientPort(rHost string, rRTPPort, rRTCPPort uint16) (rtpConn, rtcpConn *nazanet.UDPConnection, lRTPPort, lRTCPPort uint16, err error) {
	// NOTICE
	// 处理Pub时，
	// 一路流的rtp端口和rtcp端口必须不同。
	// 我尝试给ffmpeg返回rtp和rtcp同一个端口，结果ffmpeg依然使用rtp+1作为rtcp的端口。
	// 又尝试给ffmpeg返回rtp:a和rtcp:a+2的端口，结果ffmpeg依然使用a和a+1端口。
	// 也即是说，ffmpeg默认认为rtcp的端口是rtp的端口+1。而不管SETUP RESPONSE的rtcp端口是多少。
	// 我目前在Acquire2这个函数里做了保证，绑定两个可用且连续的端口。

	var rtpc, rtcpc *net.UDPConn
	rtpc, lRTPPort, rtcpc, lRTCPPort, err = availUDPConnPool.Acquire2()
	if err != nil {
		return
	}
	nazalog.Debugf("acquire udp conn. rtp port=%d, rtcp port=%d", lRTPPort, lRTCPPort)

	rtpConn, err = nazanet.NewUDPConnection(func(option *nazanet.UDPConnectionOption) {
		option.Conn = rtpc
		option.RAddr = net.JoinHostPort(rHost, fmt.Sprintf("%d", rRTPPort))
		option.MaxReadPacketSize = rtprtcp.MaxRTPRTCPPacketSize
	})
	if err != nil {
		return
	}
	rtcpConn, err = nazanet.NewUDPConnection(func(option *nazanet.UDPConnectionOption) {
		option.Conn = rtcpc
		option.RAddr = net.JoinHostPort(rHost, fmt.Sprintf("%d", rRTCPPort))
		option.MaxReadPacketSize = rtprtcp.MaxRTPRTCPPacketSize
	})
	return
}
