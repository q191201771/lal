// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"bufio"
	"fmt"
	"net"
	"strings"

	"github.com/q191201771/naza/pkg/connection"

	"github.com/cfeeling/lal/pkg/base"
	"github.com/cfeeling/lal/pkg/sdp"
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
	UniqueKey    string                       // const after ctor
	observer     ServerCommandSessionObserver // const after ctor
	conn         connection.Connection
	prevConnStat connection.Stat
	staleStat    *connection.Stat
	stat         base.StatSession

	pubSession *PubSession
	subSession *SubSession
}

func NewServerCommandSession(observer ServerCommandSessionObserver, conn net.Conn) *ServerCommandSession {
	uk := base.GenUniqueKey(base.UKPRTSPServerCommandSession)
	s := &ServerCommandSession{
		UniqueKey: uk,
		observer:  observer,
		conn: connection.New(conn, func(option *connection.Option) {
			option.ReadBufSize = serverCommandSessionReadBufSize
		}),
	}

	nazalog.Infof("[%s] lifecycle new rtsp ServerSession. session=%p, laddr=%s, raddr=%s", uk, s, conn.LocalAddr().String(), conn.RemoteAddr().String())
	return s
}

func (session *ServerCommandSession) RunLoop() error {
	return session.runCmdLoop()
}

func (session *ServerCommandSession) Dispose() error {
	nazalog.Infof("[%s] lifecycle dispose rtsp ServerCommandSession. session=%p", session.UniqueKey, session)
	return session.conn.Close()
}

// 使用RTSP TCP命令连接，向对端发送RTP数据
func (session *ServerCommandSession) WriteInterleavedPacket(packet []byte, channel int) error {
	_, err := session.conn.Write(packInterleaved(channel, packet))
	return err
}

func (session *ServerCommandSession) RemoteAddr() string {
	return session.conn.RemoteAddr().String()
}

func (session *ServerCommandSession) UpdateStat(interval uint32) {
	currStat := session.conn.GetStat()
	rDiff := currStat.ReadBytesSum - session.prevConnStat.ReadBytesSum
	session.stat.Bitrate = int(rDiff * 8 / 1024 / uint64(interval))
	wDiff := currStat.WroteBytesSum - session.prevConnStat.WroteBytesSum
	session.stat.Bitrate = int(wDiff * 8 / 1024 / uint64(interval))
	session.prevConnStat = currStat
}

func (session *ServerCommandSession) GetStat() base.StatSession {
	connStat := session.conn.GetStat()
	session.stat.ReadBytesSum = connStat.ReadBytesSum
	session.stat.WroteBytesSum = connStat.WroteBytesSum
	return session.stat
}

func (session *ServerCommandSession) IsAlive() (readAlive, writeAlive bool) {
	currStat := session.conn.GetStat()
	if session.staleStat == nil {
		session.staleStat = new(connection.Stat)
		*session.staleStat = currStat
		return true, true
	}

	readAlive = !(currStat.ReadBytesSum-session.staleStat.ReadBytesSum == 0)
	writeAlive = !(currStat.WroteBytesSum-session.staleStat.WroteBytesSum == 0)
	*session.staleStat = currStat
	return
}

func (session *ServerCommandSession) runCmdLoop() error {
	var r = bufio.NewReader(session.conn)

Loop:
	for {
		isInterleaved, packet, channel, err := readInterleaved(r)
		if err != nil {
			nazalog.Errorf("[%s] read interleaved error. err=%+v", session.UniqueKey, err)
			break Loop
		}
		if isInterleaved {
			if session.pubSession != nil {
				session.pubSession.HandleInterleavedPacket(packet, int(channel))
			} else if session.subSession != nil {
				session.subSession.HandleInterleavedPacket(packet, int(channel))
			} else {
				nazalog.Errorf("[%s] read interleaved packet but pub or sub not exist.", session.UniqueKey)
				break Loop
			}
			continue
		}

		// 读取一个message
		requestCtx, err := nazahttp.ReadHTTPRequestMessage(r)
		if err != nil {
			nazalog.Errorf("[%s] read rtsp message error. err=%+v", session.UniqueKey, err)
			break Loop
		}

		nazalog.Debugf("[%s] read http request. method=%s, uri=%s, version=%s, headers=%+v, body=%s",
			session.UniqueKey, requestCtx.Method, requestCtx.URI, requestCtx.Version, requestCtx.Headers, string(requestCtx.Body))

		var handleMsgErr error
		switch requestCtx.Method {
		case MethodOptions:
			// pub, sub
			handleMsgErr = session.handleOptions(requestCtx)
		case MethodAnnounce:
			// pub
			handleMsgErr = session.handleAnnounce(requestCtx)
		case MethodDescribe:
			// sub
			handleMsgErr = session.handleDescribe(requestCtx)
		case MethodSetup:
			// pub, sub
			handleMsgErr = session.handleSetup(requestCtx)
		case MethodRecord:
			// pub
			handleMsgErr = session.handleRecord(requestCtx)
		case MethodPlay:
			// sub
			handleMsgErr = session.handlePlay(requestCtx)
		case MethodTeardown:
			// pub
			handleMsgErr = session.handleTeardown(requestCtx)
			break Loop
		default:
			nazalog.Errorf("[%s] unknown rtsp message. method=%s", session.UniqueKey, requestCtx.Method)
		}
		if handleMsgErr != nil {
			nazalog.Errorf("[%s] handle rtsp message error. err=%+v", session.UniqueKey, handleMsgErr)
			break
		}
	}

	_ = session.conn.Close()
	nazalog.Debugf("[%s] < handleTCPConnect.", session.UniqueKey)

	return nil
}

func (session *ServerCommandSession) handleOptions(requestCtx nazahttp.HTTPReqMsgCtx) error {
	nazalog.Infof("[%s] < R OPTIONS", session.UniqueKey)
	resp := PackResponseOptions(requestCtx.Headers[HeaderCSeq])
	_, err := session.conn.Write([]byte(resp))
	return err
}

func (session *ServerCommandSession) handleAnnounce(requestCtx nazahttp.HTTPReqMsgCtx) error {
	nazalog.Infof("[%s] < R ANNOUNCE", session.UniqueKey)

	urlCtx, err := base.ParseRTSPURL(requestCtx.URI)
	if err != nil {
		nazalog.Errorf("[%s] parse presentation failed. uri=%s", session.UniqueKey, requestCtx.URI)
		return err
	}

	sdpLogicCtx, err := sdp.ParseSDP2LogicContext(requestCtx.Body)
	if err != nil {
		nazalog.Errorf("[%s] parse sdp failed. err=%v", session.UniqueKey, err)
		return err
	}

	session.pubSession = NewPubSession(urlCtx, session)
	nazalog.Infof("[%s] link new PubSession. [%s]", session.UniqueKey, session.pubSession.UniqueKey)
	session.pubSession.InitWithSDP(requestCtx.Body, sdpLogicCtx)

	if ok := session.observer.OnNewRTSPPubSession(session.pubSession); !ok {
		nazalog.Warnf("[%s] force close pubsession.", session.pubSession.UniqueKey)
		return ErrRTSP
	}

	resp := PackResponseAnnounce(requestCtx.Headers[HeaderCSeq])
	_, err = session.conn.Write([]byte(resp))
	return err
}

func (session *ServerCommandSession) handleDescribe(requestCtx nazahttp.HTTPReqMsgCtx) error {
	nazalog.Infof("[%s] < R DESCRIBE", session.UniqueKey)

	urlCtx, err := base.ParseRTSPURL(requestCtx.URI)
	if err != nil {
		nazalog.Errorf("[%s] parse presentation failed. uri=%s", session.UniqueKey, requestCtx.URI)
		return err
	}

	session.subSession = NewSubSession(urlCtx, session)
	nazalog.Infof("[%s] link new SubSession. [%s]", session.UniqueKey, session.subSession.UniqueKey)
	ok, rawSDP := session.observer.OnNewRTSPSubSessionDescribe(session.subSession)
	if !ok {
		nazalog.Warnf("[%s] force close subSession.", session.UniqueKey)
		return ErrRTSP
	}

	sdpLogicCtx, _ := sdp.ParseSDP2LogicContext(rawSDP)
	session.subSession.InitWithSDP(rawSDP, sdpLogicCtx)

	resp := PackResponseDescribe(requestCtx.Headers[HeaderCSeq], string(rawSDP))
	_, err = session.conn.Write([]byte(resp))
	return err
}

// 一次SETUP对应一路流（音频或视频）
func (session *ServerCommandSession) handleSetup(requestCtx nazahttp.HTTPReqMsgCtx) error {
	nazalog.Infof("[%s] < R SETUP", session.UniqueKey)

	remoteAddr := session.conn.RemoteAddr().String()
	host, _, _ := net.SplitHostPort(remoteAddr)

	// 是否为interleaved模式
	htv := requestCtx.Headers[HeaderTransport]
	if strings.Contains(htv, TransportFieldInterleaved) {
		rtpChannel, rtcpChannel, err := parseRTPRTCPChannel(htv)
		if err != nil {
			nazalog.Errorf("[%s] parse rtp rtcp channel error. err=%+v", session.UniqueKey, err)
			return err
		}
		if session.pubSession != nil {
			if err := session.pubSession.SetupWithChannel(requestCtx.URI, int(rtpChannel), int(rtcpChannel)); err != nil {
				nazalog.Errorf("[%s] setup channel error. err=%+v", session.UniqueKey, err)
				return err
			}
		} else if session.subSession != nil {
			if err := session.subSession.SetupWithChannel(requestCtx.URI, int(rtpChannel), int(rtcpChannel)); err != nil {
				nazalog.Errorf("[%s] setup channel error. err=%+v", session.UniqueKey, err)
				return err
			}
		} else {
			nazalog.Errorf("[%s] setup but session not exist.", session.UniqueKey)
			return ErrRTSP
		}

		resp := PackResponseSetup(requestCtx.Headers[HeaderCSeq], htv)
		_, err = session.conn.Write([]byte(resp))
		return err
	}

	rRTPPort, rRTCPPort, err := parseClientPort(requestCtx.Headers[HeaderTransport])
	if err != nil {
		nazalog.Errorf("[%s] parseClientPort failed. err=%+v", session.UniqueKey, err)
		return err
	}
	rtpConn, rtcpConn, lRTPPort, lRTCPPort, err := initConnWithClientPort(host, rRTPPort, rRTCPPort)
	if err != nil {
		nazalog.Errorf("[%s] initConnWithClientPort failed. err=%+v", session.UniqueKey, err)
		return err
	}
	nazalog.Debugf("[%s] init conn. lRTPPort=%d, lRTCPPort=%d, rRTPPort=%d, rRTCPPort=%d",
		session.UniqueKey, lRTPPort, lRTCPPort, rRTPPort, rRTCPPort)

	if session.pubSession != nil {
		if err = session.pubSession.SetupWithConn(requestCtx.URI, rtpConn, rtcpConn); err != nil {
			nazalog.Errorf("[%s] setup conn error. err=%+v", session.UniqueKey, err)
			return err
		}
		htv = fmt.Sprintf(HeaderTransportServerRecordTmpl, rRTPPort, rRTCPPort, lRTPPort, lRTCPPort)
	} else if session.subSession != nil {
		if err = session.subSession.SetupWithConn(requestCtx.URI, rtpConn, rtcpConn); err != nil {
			nazalog.Errorf("[%s] setup conn error. err=%+v", session.UniqueKey, err)
			return err
		}
		htv = fmt.Sprintf(HeaderTransportServerPlayTmpl, rRTPPort, rRTCPPort, lRTPPort, lRTCPPort)
	} else {
		nazalog.Errorf("[%s] setup but session not exist.", session.UniqueKey)
		return ErrRTSP
	}

	resp := PackResponseSetup(requestCtx.Headers[HeaderCSeq], htv)
	_, err = session.conn.Write([]byte(resp))
	return err
}

func (session *ServerCommandSession) handleRecord(requestCtx nazahttp.HTTPReqMsgCtx) error {
	nazalog.Infof("[%s] < R RECORD", session.UniqueKey)
	resp := PackResponseRecord(requestCtx.Headers[HeaderCSeq])
	_, err := session.conn.Write([]byte(resp))
	return err
}

func (session *ServerCommandSession) handlePlay(requestCtx nazahttp.HTTPReqMsgCtx) error {
	nazalog.Infof("[%s] < R PLAY", session.UniqueKey)
	if ok := session.observer.OnNewRTSPSubSessionPlay(session.subSession); !ok {
		return ErrRTSP
	}
	resp := PackResponsePlay(requestCtx.Headers[HeaderCSeq])
	_, err := session.conn.Write([]byte(resp))
	return err
}

func (session *ServerCommandSession) handleTeardown(requestCtx nazahttp.HTTPReqMsgCtx) error {
	nazalog.Infof("[%s] < R TEARDOWN", session.UniqueKey)
	resp := PackResponseTeardown(requestCtx.Headers[HeaderCSeq])
	_, err := session.conn.Write([]byte(resp))
	return err
}
