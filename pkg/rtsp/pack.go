// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"fmt"
	"time"

	"github.com/q191201771/lal/pkg/base"
)

// rfc2326 10.1 OPTIONS
// uri CSeq
var RequestOptionsTmpl = "OPTIONS %s RTSP/1.0\r\n" +
	"CSeq: %d\r\n" +
	"User-Agent: " + base.LALRTSPPullSessionUA + "\r\n" +
	"\r\n"

// CSeq
var ResponseOptionsTmpl = "RTSP/1.0 200 OK\r\n" +
	"Server: " + base.LALRTSPOptionsResponseServer + "\r\n" +
	"CSeq: %s\r\n" +
	"Public:DESCRIBE, ANNOUNCE, SETUP, PLAY, PAUSE, RECORD, TEARDOWN\r\n" +
	"\r\n"

// rfc2326 10.3 ANNOUNCE
// CSeq
var ResponseAnnounceTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"\r\n"

// rfc2326 10.2 DESCRIBE
// uri CSeq
var RequestDescribeTmpl = "DESCRIBE %s RTSP/1.0\r\n" +
	"Accept: application/sdp\r\n" +
	"CSeq: %d\r\n" +
	"User-Agent: " + base.LALRTSPPullSessionUA + "\r\n" +
	"\r\n"

// CSeq, Date, Content-Length,
var ResponseDescribeTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"Date: %s\r\n" +
	"Content-Type: application/sdp\r\n" +
	"Content-Length: %d\r\n" +
	"\r\n" +
	"%s"

// rfc2326 10.4 SETUP
// uri CSeq RTPPort RTCPPort
var RequestSetupTmpl = "SETUP %s RTSP/1.0\r\n" +
	"CSeq: %d\r\n" +
	"Transport: RTP/AVP/UDP;unicast;client_port=%d-%d\r\n" +
	"User-Agent: " + base.LALRTSPPullSessionUA + "\r\n" +
	"\r\n"

// uri CSeq Session RTPPort RTCPPort
var RequestSetupWithSessionTmpl = "SETUP %s RTSP/1.0\r\n" +
	"CSeq: %d\r\n" +
	"Session: %s\r\n" +
	"Transport: RTP/AVP/UDP;unicast;client_port=%d-%d\r\n" +
	"User-Agent: " + base.LALRTSPPullSessionUA + "\r\n" +
	"\r\n"

// CSeq, Date, Session, Transport(client_port, server_rtp_port, server_rtcp_port)
var ResponseSetupTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"Date: %s\r\n" +
	"Session: %s\r\n" +
	"Transport:RTP/AVP/UDP;unicast;client_port=%d-%d;server_port=%d-%d\r\n" +
	"\r\n"

// CSeq, Date, Session, Transport
var ResponseSetupTCPTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"Date: %s\r\n" +
	"Session: %s\r\n" +
	"Transport:%s\r\n" +
	"\r\n"

// rfc2326 10.11 RECORD
// CSeq, Session
var ResponseRecordTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"Session: %s\r\n" +
	"\r\n"

// rfc2326 10.5 PLAY
// uri CSeq Session
var RequestPlayTmpl = "PLAY %s RTSP/1.0\r\n" +
	"CSeq: %d\r\n" +
	"Range: npt=0.000-\r\n" +
	"Session: %s\r\n" +
	"User-Agent: " + base.LALRTSPPullSessionUA + "\r\n" +
	"\r\n"

// CSeq Date
var ResponsePlayTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"Date: %s\r\n" +
	"\r\n"

// rfc2326 10.7 TEARDOWN
// CSeq
var ResponseTeardownTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"\r\n"

func PackRequestOptions(uri string, cseq int) string {
	return fmt.Sprintf(RequestOptionsTmpl, uri, cseq)
}

func PackRequestDescribe(uri string, cseq int) string {
	return fmt.Sprintf(RequestDescribeTmpl, uri, cseq)
}

// @param sessionID 可以为空，如果为空，则请求中不包含`Session`字段
func PackRequestSetup(uri string, cseq int, sessionID string, rtpClientPort int, rtcpClientPort int) string {
	if sessionID == "" {
		return fmt.Sprintf(RequestSetupTmpl, uri, cseq, rtpClientPort, rtcpClientPort)
	}
	return fmt.Sprintf(RequestSetupWithSessionTmpl, uri, cseq, sessionID, rtpClientPort, rtcpClientPort)
}

func PackRequestPlay(uri string, cseq int, sessionID string) string {
	return fmt.Sprintf(RequestPlayTmpl, uri, cseq, sessionID)
}

func PackResponseOptions(cseq string) string {
	return fmt.Sprintf(ResponseOptionsTmpl, cseq)
}

func PackResponseAnnounce(cseq string) string {
	return fmt.Sprintf(ResponseAnnounceTmpl, cseq)
}

func PackResponseDescribe(cseq, sdp string) string {
	date := time.Now().Format(time.RFC1123)
	return fmt.Sprintf(ResponseDescribeTmpl, cseq, date, len(sdp), sdp)
}

// @param transportC:
//   pub example:
//   RTP/AVP/UDP;unicast;client_port=24254-24255;mode=record
//   RTP/AVP/UDP;unicast;client_port=24256-24257;mode=record
//   sub example:
//   RTP/AVP/UDP;unicast;client_port=9420-9421
func PackResponseSetup(cseq string, rRTPPort, rRTCPPort, lRTPPort, lRTCPPort uint16) string {
	date := time.Now().Format(time.RFC1123)

	return fmt.Sprintf(ResponseSetupTmpl, cseq, date, sessionID, rRTPPort, rRTCPPort, lRTPPort, lRTCPPort)
}

func PackResponseSetupTCP(cseq string, ts string) string {
	date := time.Now().Format(time.RFC1123)

	return fmt.Sprintf(ResponseSetupTCPTmpl, cseq, date, sessionID, ts)
}

func PackResponseRecord(cseq string) string {
	return fmt.Sprintf(ResponseRecordTmpl, cseq, sessionID)
}

func PackResponsePlay(cseq string) string {
	date := time.Now().Format(time.RFC1123)
	return fmt.Sprintf(ResponsePlayTmpl, cseq, date)
}

func PackResponseTeardown(cseq string) string {
	return fmt.Sprintf(ResponseTeardownTmpl, cseq)
}
