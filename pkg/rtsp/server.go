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
	"sync"

	"github.com/q191201771/naza/pkg/nazanet"

	"github.com/q191201771/lal/pkg/sdp"

	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
)

type ServerObserver interface {
	// @return 如果返回false，则表示上层要强制关闭这个推流请求
	OnNewRTSPPubSession(session *PubSession) bool

	OnDelRTSPPubSession(session *PubSession)

	// @return 如果返回false，则表示上层要强制关闭这个拉流请求
	OnNewRTSPSubSession(session *SubSession) bool

	OnDelRTSPSubSession(session *SubSession)
}

type Server struct {
	addr     string
	observer ServerObserver

	ln net.Listener

	m                       sync.Mutex
	presentation2PubSession map[string]*PubSession
}

func NewServer(addr string, observer ServerObserver) *Server {
	return &Server{
		addr:                    addr,
		observer:                observer,
		presentation2PubSession: make(map[string]*PubSession),
	}
}

func (s *Server) Listen() (err error) {
	s.ln, err = net.Listen("tcp", s.addr)
	if err != nil {
		return
	}
	nazalog.Infof("start rtsp server listen. addr=%s", s.addr)
	return
}

func (s *Server) RunLoop() error {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return err
		}
		go s.handleTCPConnect(conn)
	}
}

func (s *Server) Dispose() {
	if s.ln == nil {
		return
	}
	if err := s.ln.Close(); err != nil {
		nazalog.Error(err)
	}
}

func (s *Server) handleTCPConnect(conn net.Conn) {
	// Pub ffmpeg OPTIONS -> ANNOUNCE -> SETUP -> RECORD
	// Sub vlc OPTIONS -> DESCRIBE -> SETUP

	host, port, _ := net.SplitHostPort(conn.RemoteAddr().String())
	nazalog.Debugf("> handleTCPConnect. conn=%p, host=%s, port=%s", conn, host, port)

	//announced := false // 收到过MethodAnnounce的标志。如果收到，表示是推流
	var (
		pubSession *PubSession
		subSession *SubSession
	)

	r := bufio.NewReader(conn)
Loop:
	for {
		method, uri, headers, body, err := handleOneHTTPRequest(r)
		if err != nil {
			nazalog.Error(err)
			break Loop
		}
		nazalog.Debugf("read http request. method=%s, uri=%s, headers=%+v, body=%s", method, uri, headers, string(body))

		switch method {
		case MethodOptions:
			// pub, sub
			nazalog.Info("< R OPTIONS")
			resp := PackResponseOptions(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
		case MethodAnnounce:
			// pub
			nazalog.Info("< R ANNOUNCE")

			presentation, err := parsePresentation(uri)
			if err != nil {
				nazalog.Errorf("getPresentation failed. uri=%s", uri)
				break Loop
			}

			sdpCtx, err := sdp.ParseSDP(body)
			if err != nil {
				nazalog.Errorf("parse sdp failed. err=%v", err)
				break Loop
			}

			pubSession = NewPubSession(presentation)
			pubSession.InitWithSDP(body, sdpCtx)

			s.m.Lock()
			s.presentation2PubSession[presentation] = pubSession
			s.m.Unlock()

			// TODO chef: 缺少统一释放pubsession的逻辑

			// TODO chef: 我用ffmpeg向lal推rtsp流，发现lal直接关闭rtsp的连接，ffmpeg并不会退出，是否应先发送什么命令？
			if ok := s.observer.OnNewRTSPPubSession(pubSession); !ok {
				nazalog.Warnf("[%s] force close pubsession.", pubSession.UniqueKey)
				break Loop
			}

			//announced = true

			resp := PackResponseAnnounce(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
		case MethodSetup:
			// pub, sub
			nazalog.Info("< R SETUP")

			//presentation, err := parsePresentation(uri)
			//if err != nil {
			//	nazalog.Errorf("parsePresentation failed. uri=%s", uri)
			//	break Loop
			//}
			//
			//s.m.Lock()
			//session, ok := s.presentation2PubSession[presentation]
			//s.m.Unlock()
			//if !ok {
			//	nazalog.Errorf("presentation invalid. presentation=%s", presentation)
			//	break Loop
			//}

			// 一次SETUP对应一路流（音频或视频）

			rRTPPort, rRTCPPort, err := parseRTPRTCPPort(headers[HeaderFieldTransport])
			if err != nil {
				nazalog.Errorf("parseRTPRTCPPort failed. err=%+v", err)
				break Loop
			}
			rtpConn, rtcpConn, lRTPPort, lRTCPPort, err := initConnWithClientPort(host, rRTPPort, rRTCPPort)
			if err != nil {
				nazalog.Errorf("initConnWithClientPort failed. err=%+v", err)
				break Loop
			}

			if pubSession != nil {
				if err = pubSession.Setup(uri, rtpConn, rtcpConn); err != nil {
					nazalog.Errorf("SETUP failed. err=%+v", err)
					break Loop
				}
			} else if subSession != nil {
				subSession.SetRTPConn(rtpConn)
				subSession.SetRTCPConn(rtcpConn)
			} else {
				nazalog.Error("SETUP while session not exist.")
				break Loop
			}

			resp := PackResponseSetup(headers[HeaderFieldCSeq], rRTPPort, rRTCPPort, lRTPPort, lRTCPPort)
			_, _ = conn.Write([]byte(resp))
		case MethodRecord:
			// pub
			nazalog.Info("< R RECORD")
			resp := PackResponseRecord(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
		case MethodTeardown:
			// pub
			nazalog.Info("< R TEARDOWN")

			presentation, err := parsePresentation(uri)
			if err != nil {
				nazalog.Errorf("parsePresentation failed. uri=%s", uri)
				break Loop
			}
			s.m.Lock()
			session, ok := s.presentation2PubSession[presentation]
			delete(s.presentation2PubSession, presentation)
			s.m.Unlock()
			if ok {
				session.Dispose()
				s.observer.OnDelRTSPPubSession(session)
			}

			resp := PackResponseTeardown(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
		case MethodDescribe:
			// sub
			nazalog.Info("< R DESCRIBE")

			// TODO chef: sdp的转发放入上层group中
			presentation, err := parsePresentation(uri)
			if err != nil {
				nazalog.Errorf("parsePresentation failed. uri=%s", uri)
				break Loop
			}
			s.m.Lock()
			pubSession, ok := s.presentation2PubSession[presentation]
			s.m.Unlock()
			if ok {
				resp := PackResponseDescribe(headers[HeaderFieldCSeq], string(pubSession.rawSDP))
				_, _ = conn.Write([]byte(resp))
			} else {
				nazalog.Errorf("rtsp sub but pub not exist. presentation=%s", presentation)
				break Loop
			}

			subSession = NewSubSession(presentation)
			if !s.observer.OnNewRTSPSubSession(subSession) {
				break Loop
			}

		case MethodPlay:
			nazalog.Info("< R PLAY")
			resp := PackResponsePlay(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
		default:
			nazalog.Error(method)
		}
	}
	_ = conn.Close()
	nazalog.Debugf("< handleTCPConnect. conn=%p", conn)
}

func handleOneHTTPRequest(r *bufio.Reader) (method string, uri string, headers map[string]string, body []byte, err error) {
	var requestLine string

	requestLine, headers, err = nazahttp.ReadHTTPHeader(r)
	if err != nil {
		return
	}

	method, uri, _, err = nazahttp.ParseHTTPRequestLine(requestLine)
	if err != nil {
		return
	}

	if contentLength, ok := headers["Content-Length"]; ok {
		var cl, l int
		if cl, err = strconv.Atoi(contentLength); err == nil {
			body = make([]byte, cl)
			l, err = io.ReadAtLeast(r, body, cl)
			if l != cl {
				err = ErrRTSP
			}
			if err != nil {
				return
			}
		}
	}

	return
}

func parsePresentation(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		nazalog.Errorf("parse uri failed. uri=%s", uri)
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
		if strings.Contains(items[len(items)-1], "streamid=") {
			return items[len(items)-2], nil
		} else {
			return items[len(items)-1], nil
		}
	}
}

func parseRTPRTCPPort(setupTransport string) (rtp, rtcp uint16, err error) {
	var clientPort string
	items := strings.Split(setupTransport, ";")
	for _, item := range items {
		if strings.HasPrefix(item, "client_port") {
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
	irtp, err := strconv.Atoi(items[0])
	if err != nil {
		return 0, 0, err
	}
	irtcp, err := strconv.Atoi(items[1])
	if err != nil {
		return 0, 0, err
	}
	return uint16(irtp), uint16(irtcp), err
}

func initConnWithClientPort(rHost string, rRTPPort, rRTCPPort uint16) (rtpConn, rtcpConn *nazanet.UDPConnection, lRTPPort, lRTCPPort uint16, err error) {
	// TODO chef:
	// 以下还需要进一步确认：
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
	})
	if err != nil {
		return
	}
	rtcpConn, err = nazanet.NewUDPConnection(func(option *nazanet.UDPConnectionOption) {
		option.Conn = rtcpc
		option.RAddr = net.JoinHostPort(rHost, fmt.Sprintf("%d", rRTCPPort))
	})
	return
}
