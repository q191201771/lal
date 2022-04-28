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
	"net"
	"strings"

	"github.com/q191201771/naza/pkg/nazaerrors"

	"github.com/q191201771/naza/pkg/connection"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/sdp"
	"github.com/q191201771/naza/pkg/nazahttp"
)

type IServerCommandSessionObserver interface {
	// OnNewRtspPubSession
	//
	// @brief  Announce阶段回调
	// @return 如果返回非nil，则表示上层要强制关闭这个推流请求
	//
	OnNewRtspPubSession(session *PubSession) error

	// OnNewRtspSubSessionDescribe
	//
	// @brief Describe阶段回调
	// @return ok  如果返回false，则表示上层要强制关闭这个拉流请求
	// @return sdp
	//
	OnNewRtspSubSessionDescribe(session *SubSession) (ok bool, sdp []byte)

	// OnNewRtspSubSessionPlay
	//
	// @brief Play阶段回调
	// @return ok  如果返回非nil，则表示上层要强制关闭这个拉流请求
	//
	OnNewRtspSubSessionPlay(session *SubSession) error
}

type ServerCommandSession struct {
	uniqueKey    string                        // const after ctor
	observer     IServerCommandSessionObserver // const after ctor
	conn         connection.Connection
	prevConnStat connection.Stat
	staleStat    *connection.Stat
	stat         base.StatSession

	pubSession *PubSession
	subSession *SubSession
}

func NewServerCommandSession(observer IServerCommandSessionObserver, conn net.Conn) *ServerCommandSession {
	uk := base.GenUkRtspServerCommandSession()
	s := &ServerCommandSession{
		uniqueKey: uk,
		observer:  observer,
		conn: connection.New(conn, func(option *connection.Option) {
			option.ReadBufSize = serverCommandSessionReadBufSize
			option.WriteChanSize = serverCommandSessionWriteChanSize
		}),
	}

	Log.Infof("[%s] lifecycle new rtsp ServerSession. session=%p, laddr=%s, raddr=%s", uk, s, conn.LocalAddr().String(), conn.RemoteAddr().String())
	return s
}

func (session *ServerCommandSession) RunLoop() error {
	return session.runCmdLoop()
}

func (session *ServerCommandSession) Dispose() error {
	Log.Infof("[%s] lifecycle dispose rtsp ServerCommandSession. session=%p", session.uniqueKey, session)
	return session.conn.Close()
}

func (session *ServerCommandSession) FeedSdp(b []byte) {

}

// WriteInterleavedPacket
//
// 使用RTSP TCP命令连接，向对端发送RTP数据
//
func (session *ServerCommandSession) WriteInterleavedPacket(packet []byte, channel int) error {
	_, err := session.conn.Write(packInterleaved(channel, packet))
	return err
}

func (session *ServerCommandSession) RemoteAddr() string {
	return session.conn.RemoteAddr().String()
}

// ----- ISessionStat --------------------------------------------------------------------------------------------------

func (session *ServerCommandSession) UpdateStat(intervalSec uint32) {
	currStat := session.conn.GetStat()
	rDiff := currStat.ReadBytesSum - session.prevConnStat.ReadBytesSum
	session.stat.Bitrate = int(rDiff * 8 / 1024 / uint64(intervalSec))
	wDiff := currStat.WroteBytesSum - session.prevConnStat.WroteBytesSum
	session.stat.Bitrate = int(wDiff * 8 / 1024 / uint64(intervalSec))
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

// ----- IObject -------------------------------------------------------------------------------------------------------

func (session *ServerCommandSession) UniqueKey() string {
	return session.uniqueKey
}

// ---------------------------------------------------------------------------------------------------------------------

func (session *ServerCommandSession) runCmdLoop() error {
	var r = bufio.NewReader(session.conn)

Loop:
	for {
		isInterleaved, packet, channel, err := readInterleaved(r)
		if err != nil {
			Log.Errorf("[%s] read interleaved error. err=%+v", session.uniqueKey, err)
			break Loop
		}
		if isInterleaved {
			if session.pubSession != nil {
				session.pubSession.HandleInterleavedPacket(packet, int(channel))
			} else if session.subSession != nil {
				session.subSession.HandleInterleavedPacket(packet, int(channel))
			} else {
				Log.Errorf("[%s] read interleaved packet but pub or sub not exist.", session.uniqueKey)
				break Loop
			}
			continue
		}

		// 读取一个message
		requestCtx, err := nazahttp.ReadHttpRequestMessage(r)
		if err != nil {
			Log.Errorf("[%s] read rtsp message error. err=%+v", session.uniqueKey, err)
			break Loop
		}

		Log.Debugf("[%s] read http request. method=%s, uri=%s, version=%s, headers=%+v, body=%s",
			session.uniqueKey, requestCtx.Method, requestCtx.Uri, requestCtx.Version, requestCtx.Headers, string(requestCtx.Body))

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
			Log.Errorf("[%s] unknown rtsp message. method=%s", session.uniqueKey, requestCtx.Method)
		}
		if handleMsgErr != nil {
			Log.Errorf("[%s] handle rtsp message error. err=%+v, ctx=%+v", session.uniqueKey, handleMsgErr, requestCtx)
			break
		}
	}

	_ = session.conn.Close()
	Log.Debugf("[%s] < handleTcpConnect.", session.uniqueKey)

	return nil
}

func (session *ServerCommandSession) handleOptions(requestCtx nazahttp.HttpReqMsgCtx) error {
	Log.Infof("[%s] < R OPTIONS", session.uniqueKey)
	resp := PackResponseOptions(requestCtx.Headers.Get(HeaderCSeq))
	_, err := session.conn.Write([]byte(resp))
	return err
}

func (session *ServerCommandSession) handleAnnounce(requestCtx nazahttp.HttpReqMsgCtx) error {
	Log.Infof("[%s] < R ANNOUNCE", session.uniqueKey)

	urlCtx, err := base.ParseRtspUrl(requestCtx.Uri)
	if err != nil {
		Log.Errorf("[%s] parse presentation failed. uri=%s", session.uniqueKey, requestCtx.Uri)
		return err
	}

	sdpCtx, err := sdp.ParseSdp2LogicContext(requestCtx.Body)
	if err != nil {
		Log.Errorf("[%s] parse sdp failed. err=%v", session.uniqueKey, err)
		return err
	}

	session.pubSession = NewPubSession(urlCtx, session)
	Log.Infof("[%s] link new PubSession. [%s]", session.uniqueKey, session.pubSession.uniqueKey)
	session.pubSession.InitWithSdp(sdpCtx)

	if err = session.observer.OnNewRtspPubSession(session.pubSession); err != nil {
		return err
	}

	resp := PackResponseAnnounce(requestCtx.Headers.Get(HeaderCSeq))
	_, err = session.conn.Write([]byte(resp))
	return err
}

func (session *ServerCommandSession) handleDescribe(requestCtx nazahttp.HttpReqMsgCtx) error {
	Log.Infof("[%s] < R DESCRIBE", session.uniqueKey)

	urlCtx, err := base.ParseRtspUrl(requestCtx.Uri)
	if err != nil {
		Log.Errorf("[%s] parse presentation failed. uri=%s", session.uniqueKey, requestCtx.Uri)
		return err
	}

	session.subSession = NewSubSession(urlCtx, session)
	Log.Infof("[%s] link new SubSession. [%s]", session.uniqueKey, session.subSession.uniqueKey)
	ok, rawSdp := session.observer.OnNewRtspSubSessionDescribe(session.subSession)
	if !ok {
		Log.Warnf("[%s] force close subSession.", session.uniqueKey)
		return base.ErrRtspClosedByObserver
	}

	sdpCtx, _ := sdp.ParseSdp2LogicContext(rawSdp)
	session.subSession.InitWithSdp(sdpCtx)

	resp := PackResponseDescribe(requestCtx.Headers.Get(HeaderCSeq), string(rawSdp))
	_, err = session.conn.Write([]byte(resp))
	return err
}

// 一次SETUP对应一路流（音频或视频）
func (session *ServerCommandSession) handleSetup(requestCtx nazahttp.HttpReqMsgCtx) error {
	Log.Infof("[%s] < R SETUP", session.uniqueKey)

	remoteAddr := session.conn.RemoteAddr().String()
	host, _, _ := net.SplitHostPort(remoteAddr)

	// 是否为interleaved模式
	htv := requestCtx.Headers.Get(HeaderTransport)
	if strings.Contains(htv, TransportFieldInterleaved) {
		rtpChannel, rtcpChannel, err := parseRtpRtcpChannel(htv)
		if err != nil {
			Log.Errorf("[%s] parse rtp rtcp channel error. err=%+v", session.uniqueKey, err)
			return err
		}
		if session.pubSession != nil {
			if err := session.pubSession.SetupWithChannel(requestCtx.Uri, int(rtpChannel), int(rtcpChannel)); err != nil {
				Log.Errorf("[%s] setup channel error. err=%+v", session.uniqueKey, err)
				return err
			}
		} else if session.subSession != nil {
			if err := session.subSession.SetupWithChannel(requestCtx.Uri, int(rtpChannel), int(rtcpChannel)); err != nil {
				Log.Errorf("[%s] setup channel error. err=%+v", session.uniqueKey, err)
				return err
			}
		} else {
			Log.Errorf("[%s] setup but session not exist.", session.uniqueKey)
			return nazaerrors.Wrap(base.ErrRtsp)
		}

		resp := PackResponseSetup(requestCtx.Headers.Get(HeaderCSeq), htv)
		_, err = session.conn.Write([]byte(resp))
		return err
	}

	rRtpPort, rRtcpPort, err := parseClientPort(requestCtx.Headers.Get(HeaderTransport))
	if err != nil {
		Log.Errorf("[%s] parseClientPort failed. err=%+v", session.uniqueKey, err)
		return err
	}
	rtpConn, rtcpConn, lRtpPort, lRtcpPort, err := initConnWithClientPort(host, rRtpPort, rRtcpPort)
	if err != nil {
		Log.Errorf("[%s] initConnWithClientPort failed. err=%+v", session.uniqueKey, err)
		return err
	}
	Log.Debugf("[%s] init conn. lRtpPort=%d, lRtcpPort=%d, rRtpPort=%d, rRtcpPort=%d",
		session.uniqueKey, lRtpPort, lRtcpPort, rRtpPort, rRtcpPort)

	if session.pubSession != nil {
		if err = session.pubSession.SetupWithConn(requestCtx.Uri, rtpConn, rtcpConn); err != nil {
			Log.Errorf("[%s] setup conn error. err=%+v", session.uniqueKey, err)
			return err
		}
		htv = fmt.Sprintf(HeaderTransportServerRecordTmpl, rRtpPort, rRtcpPort, lRtpPort, lRtcpPort)
	} else if session.subSession != nil {
		if err = session.subSession.SetupWithConn(requestCtx.Uri, rtpConn, rtcpConn); err != nil {
			Log.Errorf("[%s] setup conn error. err=%+v", session.uniqueKey, err)
			return err
		}
		htv = fmt.Sprintf(HeaderTransportServerPlayTmpl, rRtpPort, rRtcpPort, lRtpPort, lRtcpPort)
	} else {
		Log.Errorf("[%s] setup but session not exist.", session.uniqueKey)
		return nazaerrors.Wrap(base.ErrRtsp)
	}

	resp := PackResponseSetup(requestCtx.Headers.Get(HeaderCSeq), htv)
	_, err = session.conn.Write([]byte(resp))
	return err
}

func (session *ServerCommandSession) handleRecord(requestCtx nazahttp.HttpReqMsgCtx) error {
	Log.Infof("[%s] < R RECORD", session.uniqueKey)
	resp := PackResponseRecord(requestCtx.Headers.Get(HeaderCSeq))
	_, err := session.conn.Write([]byte(resp))
	return err
}

func (session *ServerCommandSession) handlePlay(requestCtx nazahttp.HttpReqMsgCtx) error {
	Log.Infof("[%s] < R PLAY", session.uniqueKey)
	// TODO(chef): [opt] 上层关闭，可以考虑回复非200状态码再关闭
	if err := session.observer.OnNewRtspSubSessionPlay(session.subSession); err != nil {
		return err
	}
	resp := PackResponsePlay(requestCtx.Headers.Get(HeaderCSeq))
	_, err := session.conn.Write([]byte(resp))
	return err
}

func (session *ServerCommandSession) handleTeardown(requestCtx nazahttp.HttpReqMsgCtx) error {
	Log.Infof("[%s] < R TEARDOWN", session.uniqueKey)
	resp := PackResponseTeardown(requestCtx.Headers.Get(HeaderCSeq))
	_, err := session.conn.Write([]byte(resp))
	return err
}
