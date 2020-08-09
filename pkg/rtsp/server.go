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
	OnNewRTSPPubSession(session *PubSession)
	OnDelRTSPPubSession(session *PubSession)
}

type Server struct {
	addr string
	obs  ServerObserver

	ln               net.Listener
	availUDPConnPool *nazanet.AvailUDPConnPool

	m                       sync.Mutex
	presentation2PubSession map[string]*PubSession
}

func NewServer(addr string, obs ServerObserver) *Server {
	return &Server{
		addr:                    addr,
		obs:                     obs,
		availUDPConnPool:        nazanet.NewAvailUDPConnPool(minServerPort, maxServerPort),
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
	nazalog.Debugf("> handleTCPConnect. conn=%p", conn)
	r := bufio.NewReader(conn)
	for {
		method, uri, headers, body, err := handleOneHTTPRequest(r)
		if err != nil {
			nazalog.Error(err)
			break
		}
		nazalog.Debugf("read http request. method=%s, uri=%s, headers=%+v, body=%s", method, uri, headers, string(body))

		switch method {
		case MethodOptions:
			// pub
			nazalog.Info("< R OPTIONS")
			resp := PackResponseOptions(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
		case MethodAnnounce:
			// pub
			nazalog.Info("< R ANNOUNCE")

			presentation, err := parsePresentation(uri)
			if err != nil {
				nazalog.Errorf("getPresentation failed. uri=%s", uri)
				break
			}

			sdp, err := sdp.ParseSDP(body)
			if err != nil {
				nazalog.Errorf("parse sdp failed. err=%v", err)
				break
			}

			pubSession := NewPubSession(presentation)
			pubSession.InitWithSDP(sdp)

			s.m.Lock()
			s.presentation2PubSession[presentation] = pubSession
			s.m.Unlock()

			s.obs.OnNewRTSPPubSession(pubSession)

			resp := PackResponseAnnounce(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
		case MethodSetup:
			// pub
			nazalog.Info("< R SETUP")

			presentation, err := parsePresentation(uri)
			if err != nil {
				nazalog.Errorf("getPresentation failed. uri=%s", uri)
			}

			s.m.Lock()
			session, ok := s.presentation2PubSession[presentation]
			s.m.Unlock()
			if !ok {
				nazalog.Errorf("presentation invalid. presentation=%s", presentation)
				break
			}

			// 一次SETUP对应一路流（音频或视频）

			// TODO chef:
			// 以下还需要进一步确认：
			// 一路流的rtp端口和rtcp端口必须不同。
			// 我尝试给ffmpeg返回rtp和rtcp同一个端口，结果ffmpeg依然使用rtp+1作为rtcp的端口。
			// 又尝试给ffmpeg返回rtp:a和rtcp:a+2的端口，结果ffmpeg依然使用a和a+1端口。
			// 也即是说，ffmpeg默认认为rtcp的端口是rtp的端口+1。而不管SETUP RESPONSE的rtcp端口是多少。
			// 我目前在Acquire2这个函数里做了保证，绑定两个可用且连续的端口。

			rtpConn, rtpPort, rtcpConn, rtcpPort, err := s.availUDPConnPool.Acquire2()
			if err != nil {
				nazalog.Errorf("acquire udp server failed. err=%+v", err)
				break
			}
			nazalog.Debugf("acquire udp conn. rtp port=%d, rtcp port=%d", rtpPort, rtcpPort)
			session.SetRTPConn(rtpConn)
			session.SetRTCPConn(rtcpConn)

			resp := PackResponseSetup(headers[HeaderFieldCSeq], headers[HeaderFieldTransport], rtpPort, rtcpPort)
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
				nazalog.Errorf("getPresentation failed. uri=%s", uri)
			}
			s.m.Lock()
			session, ok := s.presentation2PubSession[presentation]
			delete(s.presentation2PubSession, presentation)
			s.m.Unlock()
			if ok {
				session.Dispose()
				s.obs.OnDelRTSPPubSession(session)
			}

			resp := PackResponseTeardown(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
		case MethodDescribe:
			nazalog.Info("< R DESCRIBE")
			resp := PackResponseDescribe(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
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
	items := strings.Split(u.Path, "/")
	if len(items) < 3 {
		return "", ErrRTSP
	}

	return items[2], nil
}
