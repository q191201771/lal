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

	"github.com/q191201771/lal/pkg/sdp"

	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
)

type ServerObserver interface {
	OnNewRTSPPubSession(session *PubSession)
	OnDelRTSPPubSession(session *PubSession)
}

// TODO chef:
// 重命名为CmdServer或者其他名字
type Server struct {
	addr string
	obs  ServerObserver

	udpServerPool *UDPServerPool
	ln            net.Listener

	m                       sync.Mutex
	presentation2PubSession map[string]*PubSession
}

func NewServer(addr string, obs ServerObserver) *Server {
	return &Server{
		addr:                    addr,
		obs:                     obs,
		udpServerPool:           NewUDPServerPool(minServerPort, maxServerPort),
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

func (s *Server) handleTCPConnect(conn net.Conn) {
	nazalog.Debugf("> handleTCPConnect. conn=%p", conn)
	r := bufio.NewReader(conn)
	for {
		requestLine, headers, err := nazahttp.ReadHTTPHeader(r)
		if err != nil {
			nazalog.Errorf("ReadHTTPHeader error. err=%v", err)
			break
		}
		nazalog.Debugf("requestLine=%s, headers=%+v", requestLine, headers)

		method, uri, _, err := nazahttp.ParseHTTPRequestLine(requestLine)
		if err != nil {
			nazalog.Errorf("ParseHTTPRequestLine error. err=%v", err)
			break
		}

		var body []byte
		if contentLength, ok := headers["Content-Length"]; ok {
			if cl, err := strconv.Atoi(contentLength); err == nil {
				body = make([]byte, cl)
				l, err := io.ReadAtLeast(r, body, cl)
				if l != cl || err != nil {
					nazalog.Errorf("read rtsp cmd fail. content-length=%d, read length=%d, err=%+v", cl, l, err)
					break
				} else {
					nazalog.Debugf("body=%s", string(body))
				}
			}
		}
		_ = body

		// TODO chef:
		// 1. header field not exist?
		//
		switch method {
		case MethodOptions:
			// pub
			nazalog.Info("< R OPTIONS")
			resp := PackResponseOptions(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
		case MethodAnnounce:
			// pub
			nazalog.Info("< R ANNOUNCE")

			u, err := url.Parse(uri)
			if err != nil {
				nazalog.Errorf("parse uri failed. uri=%s", uri)
				break
			}
			items := strings.Split(u.Path, "/")
			if len(items) < 2 {
				nazalog.Errorf("uri invalid. uri=%s", uri)
				break
			}
			presentation := items[len(items)-1]

			sdp, err := sdp.ParseSDP(body)
			if err != nil {
				nazalog.Errorf("parse sdp failed. err=%v", err)
				break
			}

			pubSession := NewPubSession(presentation)
			pubSession.InitWithSDP(sdp)

			s.obs.OnNewRTSPPubSession(pubSession)

			s.presentation2PubSession[presentation] = pubSession

			resp := PackResponseAnnounce(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
		case MethodSetup:
			// pub
			nazalog.Info("< R SETUP")

			u, err := url.Parse(uri)
			if err != nil {
				nazalog.Errorf("parse uri failed. uri=%s", uri)
				break
			}
			items := strings.Split(u.Path, "/")
			if len(items) < 3 {
				nazalog.Errorf("uri invalid. uri=%s", uri)
				break
			}
			presentation := items[len(items)-2]

			s.m.Lock()
			session, ok := s.presentation2PubSession[presentation]
			s.m.Unlock()
			if !ok {
				nazalog.Errorf("presentation invalid. presentation=%s", presentation)
				break
			}

			udpConn, port, err := s.udpServerPool.Acquire()
			nazalog.Debugf("acquire udp conn. port=%d", port)
			if err != nil {
				nazalog.Errorf("acquire udp server failed. err=%v", err)
				break
			}
			session.AddConn(udpConn)

			resp := PackResponseSetup(headers[HeaderFieldCSeq], headers[HeaderFieldTransport], port, port)
			_, _ = conn.Write([]byte(resp))
		case MethodRecord:
			// pub
			nazalog.Info("< R RECORD")
			resp := PackResponseRecord(headers[HeaderFieldCSeq])
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
