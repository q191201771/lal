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

	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
)

type ServerObserver interface {
	OnNewRTSPPubSession(session *PubSession)
	OnDelRTSPPubSession(session *PubSession)
	OnNewRTSPSubSession(session *SubSession)
	OnDelRTSPSubSession(session *SubSession)
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
	// 保存所有拉流的
	presentation2SubSession map[string]*SubSession
}

func NewServer(addr string, obs ServerObserver) *Server {
	return &Server{
		addr:                    addr,
		obs:                     obs,
		udpServerPool:           NewUDPServerPool(minServerPort, maxServerPort),
		presentation2PubSession: make(map[string]*PubSession),
		presentation2SubSession: make(map[string]*SubSession),
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
	// 上一个命令,推流和拉流会使用不同命令,setup时需要判断是推/拉
	// 推: option,announce,setup,record
	// 拉: option,describe,setup,play
	var lastMethod string
	// 在创建PubSession/SubSession(announce/describe)时的流名字,确保连接关闭时释放session
	var sessionKey string

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
			lastMethod = method
		case MethodAnnounce:
			// pub
			nazalog.Info("< R ANNOUNCE")
			presentation,err := getStreamName(uri,false)
			if err != nil {
				break;
			}
			sdp, err := ParseSDP(body)
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
			lastMethod = method
			sessionKey = presentation
		case MethodSetup:
			// pub
			nazalog.Info("< R SETUP")
			presentation,err := getStreamName(uri,true)
			if err != nil {
				break;
			}

			// 通过lastMethod来区分是推/拉的setup
			if lastMethod == MethodAnnounce {
				//是推流
				s.m.Lock()
				session, ok := s.presentation2PubSession[presentation]
				if !ok {
					nazalog.Errorf("presentation invalid. presentation=%s", presentation)
					s.m.Unlock()
					break
				}
				s.m.Unlock()

				udpConn, port, err := s.udpServerPool.Acquire()
				nazalog.Debugf("acquire udp conn. port=%d", port)
				if err != nil {
					nazalog.Errorf("acquire udp server failed. err=%v", err)
					break
				}
				session.AddConn(udpConn)

				resp := PackResponseSetup(headers[HeaderFieldCSeq], headers[HeaderFieldTransport], port, port)
				_, _ = conn.Write([]byte(resp))
			} else {
				//是拉流
				s.m.Lock()
				session, ok := s.presentation2SubSession[presentation]
				s.m.Unlock()
				if !ok {
					nazalog.Errorf("presentation invalid. presentation=%s", presentation)
					break
				}

				transport,_ := headers["Transport"]
				transport = strings.ToUpper(transport)
				// rtp_over_udp
				if !strings.Contains(transport,"TCP") {
					udpConn, port, err := s.udpServerPool.Acquire()
					nazalog.Debugf("acquire udp conn. port=%d", port)
					if err != nil {
						nazalog.Errorf("acquire udp server failed. err=%v", err)
						break
					}
					rtp,rtcp := clientPorts(headers["Transport"])
					session.AddConn(udpConn,rtp,rtcp)
					// 一个trackID对应一个本地端口,注意RTP/RTCP使用了同一个本地端口而不是一般做法分开的两个端口
					resp := PackResponseSetup(headers[HeaderFieldCSeq], headers[HeaderFieldTransport], port, port)
					_, _ = conn.Write([]byte(resp))
				} else {
					//暂时不支持over tcp
					nazalog.Info("rtp over tcp: %s",transport)
					break
				}
			}
		case MethodRecord:
			// pub
			nazalog.Info("< R RECORD")
			resp := PackResponseRecord(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
			lastMethod = method
		case MethodDescribe:
			nazalog.Info("< R DESCRIBE")

			// describe的时候创建拉流的SubSession
			presentation,err := getStreamName(uri,false)
			if err != nil {
				break;
			}
			subSession := NewSubSession(presentation,conn)
			s.m.Lock()
			s.presentation2SubSession[presentation] = subSession
			s.m.Unlock()
			// 暂时只支持播放TS流
			resp := PackResponseDescribe(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
			sessionKey = presentation
		case MethodPlay:
			nazalog.Info("< R PLAY")
			// 在客户端PLAY后才执行OnNewRTSPSubSession
			// 因为OnNewRTSPSubSession后可能会立即向客户端发送数据,要求必须准备好相关连接
			presentation,err := getStreamName(uri,false)
			if err != nil {
				break;
			}
			s.m.Lock()
			session, _ := s.presentation2SubSession[presentation]
			s.m.Unlock()
			s.obs.OnNewRTSPSubSession(session)

			resp := PackResponsePlay(headers[HeaderFieldCSeq])
			_, _ = conn.Write([]byte(resp))
			lastMethod = method
		case MethodTeardown:
			presentation,err := getStreamName(uri,false)
			if err != nil {
				break;
			}
			s.deleteSession(lastMethod,presentation)
		default:
			nazalog.Error(method)
		}
	}
	_ = conn.Close()
	if len(sessionKey) > 0 {
		//确保session被释放,因为可能存在创建session后其它命令失败的情况
		s.deleteSession(lastMethod,sessionKey)
	}
	nazalog.Debugf("< handleTCPConnect. conn=%p", conn)
}
// 从SETUP中取出rtp_rtcp端口,client_port=xxx-xxx
func clientPorts(str string) (int, int) {
	// Transport: RTP/AVP/UDP;unicast;client_port=xxx-xxx
	index := strings.Index(str,HeaderFieldClientPort)
	// client_port=xxx-xxx
	str = str[index:len(str)]
	// xxx-xxx
	str = str[len(HeaderFieldClientPort) + 1 : len(str)]

	ports := strings.Split(str, "-")
	if len(ports) != 2 {
		return 0, 0
	}
	port1, err := strconv.ParseInt(ports[0], 10, 64)
	if err != nil {
		return 0, 0
	}
	port2, err := strconv.ParseInt(ports[1], 10, 64)
	if err != nil {
		return 0, 0
	}
	return int(port1), int(port2)
}
// 从URL中取出流名字: xxxx/stream1  ->  stream1, xxxx/stream1/trackID=0  ->  stream1
func getStreamName(uri string,isSetup bool) (streamName string,err error) {
	var presentation string

	u, err := url.Parse(uri)
	if err != nil {
		nazalog.Errorf("parse uri failed. uri=%s", uri)
		return presentation,err
	}
	items := strings.Split(u.Path, "/")
	if len(items) < 2 {
		nazalog.Errorf("uri invalid. uri=%s", uri)
		return presentation,err
	}
	n := 1
	if isSetup {
		n = 2
	}
	presentation = items[len(items)-n]

	return presentation,nil
}
// 释放session
func (s *Server) deleteSession(lastMethod string,sessionKey string)  {
	s.m.Lock()
	if lastMethod == MethodPlay {
		//拉流
		session, ok := s.presentation2SubSession[sessionKey]
		if ok {
			session.Dispose()
			delete(s.presentation2SubSession,sessionKey)
		}
	} else {
		//推流
		session, ok := s.presentation2PubSession[sessionKey]
		if ok {
			session.Dispose()
			delete(s.presentation2PubSession, sessionKey)
		}
	}
	s.m.Unlock()
}