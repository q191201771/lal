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

	"github.com/q191201771/naza/pkg/nazalog"
)

// VLC rtsp://127.0.0.1:5544/vlc
// handleTCPConnect. conn=0xc000010030 - server.go:50
// requestLine=OPTIONS rtsp://127.0.0.1:5544/vlc RTSP/1.0, headers=map[CSeq:2 User-Agent:LibVLC/3.0.8 (LIVE555 Streaming Media v2016.11.28)] - server.go:58
// requestLine=DESCRIBE rtsp://127.0.0.1:5544/vlc RTSP/1.0, headers=map[Accept:application/sdp CSeq:3 User-Agent:LibVLC/3.0.8 (LIVE555 Streaming Media v2016.11.28)] - server.go:58
// EOF - server.go:55
// handleTCPConnect. conn=0xc000010038 - server.go:50
// requestLine=SETUP rtsp://127.0.0.1:5544/vlc RTSP/1.0, headers=map[CSeq:0 Transport:RTP/AVP;unicast;client_port=9300-9301] - server.go:58
// requestLine=PLAY rtsp://127.0.0.1:5544/stream=0 RTSP/1.0, headers=map[CSeq:1 Session:47112344] - server.go:58

// ffmpeg -re -stream_loop -1 -i /Volumes/Data/tmp/test.flv -c copy -f rtsp rtsp://localhost:5544/test110
//  INFO start hls server listen. addr=:5544 - server.go:37
// DEBUG handleTCPConnect. conn=0xc000010030 - server.go:52
// DEBUG requestLine=OPTIONS rtsp://localhost:5544/test110 RTSP/1.0, headers=map[CSeq:1 User-Agent:Lavf57.83.100] - server.go:60
//  INFO < R OPTIONS - server.go:85
// DEBUG requestLine=ANNOUNCE rtsp://localhost:5544/test110 RTSP/1.0, headers=map[CSeq:2 Content-Length:481 Content-Type:application/sdp User-Agent:Lavf57.83.100] - server.go:60
// DEBUG body=v=0
// o=- 0 0 IN IP6 ::1
// s=No Name
// c=IN IP6 ::1
// t=0 0
// a=tool:libavformat 57.83.100
// m=video 0 RTP/AVP 96
// b=AS:212
// a=rtpmap:96 H264/90000
// a=fmtp:96 packetization-mode=1; sprop-parameter-sets=Z2QAIKzZQMApsBEAAAMAAQAAAwAyDxgxlg==,aOvssiw=; profile-level-id=640020
// a=control:streamid=0
// m=audio 0 RTP/AVP 97
// b=AS:30
// a=rtpmap:97 MPEG4-GENERIC/44100/2
// a=fmtp:97 profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3; config=1210
// a=control:streamid=1
// - server.go:76
//  INFO < R ANNOUNCE - server.go:89
// DEBUG requestLine=SETUP rtsp://localhost:5544/test110/streamid=0 RTSP/1.0, headers=map[CSeq:3 Transport:RTP/AVP/UDP;unicast;client_port=5560-5561;mode=record User-Agent:Lavf57.83.100] - server.go:60
//  INFO < R SETUP - server.go:97
// DEBUG requestLine=SETUP rtsp://localhost:5544/test110/streamid=1 RTSP/1.0, headers=map[CSeq:4 Session:47112344 Transport:RTP/AVP/UDP;unicast;client_port=5562-5563;mode=record User-Agent:Lavf57.83.100] - server.go:60
//  INFO < R SETUP - server.go:97
// DEBUG requestLine=RECORD rtsp://localhost:5544/test110 RTSP/1.0, headers=map[CSeq:5 Range:npt=0.000- Session:47112344 User-Agent:Lavf57.83.100] - server.go:60
// ERROR RECORD - server.go:105

// [DONE]
// rfc2326 10.1 OPTIONS
// CSeq
var ResponseOptionsTmpl = "RTSP/1.0 200 OK\r\n" +
	"Server: " + serverName + "\r\n" +
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

func PackResponseDescribe(cseq string) string {
	date := time.Now().Format(time.RFC1123)
	return fmt.Sprintf(ResponseDescribeTmpl, cseq, date, 376)
}

func PackResponsePlay(cseq string) string {
	date := time.Now().Format(time.RFC1123)
	return fmt.Sprintf(PlayTmpl, cseq, date)
}
