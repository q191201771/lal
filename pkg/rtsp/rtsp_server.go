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
	"strconv"

	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
)

// TODO chef:
// 重命名为CmdServer或者其他名字
type Server struct {
	addr string
	ln   net.Listener
}

func NewServer(addr string) *Server {
	return &Server{
		addr: addr,
	}
}

func (s *Server) Listen() (err error) {
	s.ln, err = net.Listen("tcp", s.addr)
	if err != nil {
		return
	}
	nazalog.Infof("start hls server listen. addr=%s", s.addr)
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
	nazalog.Debugf("handleTCPConnect. conn=%p", conn)
	r := bufio.NewReader(conn)
	for {
		requestLine, headers, err := nazahttp.ReadHTTPHeader(r)
		if err != nil {
			nazalog.Error(err)
			break
		}
		nazalog.Debugf("requestLine=%s, headers=%+v", requestLine, headers)

		method, _, _, err := nazahttp.ParseHTTPRequestLine(requestLine)
		if err != nil {
			nazalog.Error(err)
			break
		}

		var body []byte
		if contentLength, ok := headers["Content-Length"]; ok {
			if cl, err := strconv.Atoi(contentLength); err == nil {
				body = make([]byte, cl)
				l, err := io.ReadAtLeast(r, body, cl)
				if l != cl || err != nil {
					nazalog.Errorf("read rtsp cmd fail. content-length=%d, read length=%d, err=%+v", cl, l, err)
				} else {
					nazalog.Debugf("body=%s", string(body))
				}
			}
		}
		_ = body

		// TODO chef: header field not exist?
		switch method {
		case MethodOptions:
			nazalog.Info("< R OPTIONS")
			resp := PackResponseOptions(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
		case MethodAnnounce:
			nazalog.Info("< R ANNOUNCE")
			ParseSDP(body)
			resp := PackResponseAnnounce(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
		case MethodSetup:
			nazalog.Info("< R SETUP")
			resp := PackResponseSetup(headers[HeaderFieldCSeq], headers[HeaderFieldTransport])
			_, _ = conn.Write([]byte(resp))
		case MethodRecord:
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
}
