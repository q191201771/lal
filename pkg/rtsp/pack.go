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
	"strings"
	"time"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/nazalog"
)

// [DONE]
// rfc2326 10.1 OPTIONS
// CSeq
var ResponseOptionsTmpl = "RTSP/1.0 200 OK\r\n" +
	"Server: " + base.LALFullInfo + "\r\n" +
	"CSeq: %s\r\n" +
	//"Public:DESCRIBE, SETUP, TEARDOWN, PLAY, PAUSE\r\n\r\n"
	"Public:DESCRIBE, ANNOUNCE, SETUP, PLAY, PAUSE, RECORD, TEARDOWN\r\n" +
	"\r\n"

// [DONE]
// rfc2326 10.3 ANNOUNCE
// CSeq
var AnnounceTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"\r\n"

// rfc2326 10.4 SETUP
// CSeq, Date, Session, Transport(client_port, server_rtp_port, server_rtcp_port)
var ResponseSetupTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"Date: %s\r\n" +
	"Session: %s\r\n" +
	"Transport:RTP/AVP/UDP;unicast;client_port=%s;server_port=%d-%d\r\n" +
	"\r\n"

// rfc2326 10.11 RECORD
// CSeq, Session
var ResponseRecordTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"Session: %s\r\n" +
	"\r\n"

var ResponseTeardownTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"\r\n"

// rfc2326 10.2 DESCRIBE
// CSeq, Date, Content-Length,
var ResponseDescribeTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"Date: %s\r\n" +
	"Content-Type: application/sdp\r\n" +
	"Content-Length: %d\r\n" +
	"\r\n" +
	SDPTmpl

// rfc4566
// * v=0
//   Session Description Protocol Version (v)
// * o=<username> <sess-id> <sess-version> <nettype> <addrtype> <unicast-address>
//     <unicast-address>
//   Owner/Creator, Session Id (o)
// * s=
//   Session Name (s)
// * c=
//   Connection Information (c)
// * t=<start-time> <stop-time>
//   Time Description, active time (t)
// * a=
//   Session Attribute | Media Attribute (a)
// * m=
//   Media Description, name and address (m)
// * b=
//   Bandwidth Information (b)
var SDPTmpl = "v=0\r\n" +
	"o=mhandley 2890844526 2890842807 IN IP4 126.16.64.4\r\n" +
	//"s=SDP Seminar\r\n" +
	//"i=A Seminar on the session description protocol\r\n" +
	//"u=http://www.cs.ucl.ac.uk/staff/M.Handley/sdp.03.ps\r\n" +
	//"e=mjh@isi.edu (Mark Handley)\r\n" +
	//"c=IN IP4 224.2.17.12/127\r\n" +
	//"t=2873397496 2873404696\r\n" +
	"t=0 0\r\n" +
	"a=recvonly\r\n"
	//"m=audio 3456 RTP/AVP 0\r\n" +
	//"m=video 2232 RTP/AVP 31\r\n" +
	//"m=whiteboard 32416 UDP WB\r\n" +
	//"a=orient:portrait\r\n"

// rfc2326 10.5 PLAY
var PlayTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"Date: %s\r\n" +
	"\r\n"

func PackResponseOptions(cseq string) string {
	return fmt.Sprintf(ResponseOptionsTmpl, cseq)
}

func PackResponseAnnounce(cseq string) string {
	return fmt.Sprintf(AnnounceTmpl, cseq)
}

// param transportC:
//   server pub example:
//   RTP/AVP/UDP;unicast;client_port=24254-24255;mode=record
//   RTP/AVP/UDP;unicast;client_port=24256-24257;mode=record
func PackResponseSetup(cseq string, transportC string, serverRTPPort uint16, serverRTCPPort uint16) string {
	date := time.Now().Format(time.RFC1123)
	nazalog.Debug(transportC)

	var clientPort string
	items := strings.Split(transportC, ";")
	for _, item := range items {
		if strings.HasPrefix(item, "client_port") {
			kv := strings.Split(item, "=")
			if len(kv) != 2 {
				continue
			}
			clientPort = kv[1]
		}
	}

	return fmt.Sprintf(ResponseSetupTmpl, cseq, date, sessionID, clientPort, serverRTPPort, serverRTCPPort)
}

func PackResponseRecord(cseq string) string {
	return fmt.Sprintf(ResponseRecordTmpl, cseq, sessionID)
}

func PackResponseTeardown(cseq string) string {
	return fmt.Sprintf(ResponseTeardownTmpl, cseq)
}

func PackResponseDescribe(cseq string) string {
	date := time.Now().Format(time.RFC1123)
	return fmt.Sprintf(ResponseDescribeTmpl, cseq, date, 376)
}

func PackResponsePlay(cseq string) string {
	date := time.Now().Format(time.RFC1123)
	return fmt.Sprintf(PlayTmpl, cseq, date)
}
