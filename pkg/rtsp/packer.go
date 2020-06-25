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
)

// VLC rtsp://127.0.0.1:5544/vlc
// handleTCPConnect. conn=0xc000010030 - server.go:50
// requestLine=OPTIONS rtsp://127.0.0.1:5544/vlc RTSP/1.0, headers=map[CSeq:2 User-Agent:LibVLC/3.0.8 (LIVE555 Streaming Media v2016.11.28)] - server.go:58
// requestLine=DESCRIBE rtsp://127.0.0.1:5544/vlc RTSP/1.0, headers=map[Accept:application/sdp CSeq:3 User-Agent:LibVLC/3.0.8 (LIVE555 Streaming Media v2016.11.28)] - server.go:58
// EOF - server.go:55
// handleTCPConnect. conn=0xc000010038 - server.go:50
// requestLine=SETUP rtsp://127.0.0.1:5544/vlc RTSP/1.0, headers=map[CSeq:0 Transport:RTP/AVP;unicast;client_port=9300-9301] - server.go:58
// requestLine=PLAY rtsp://127.0.0.1:5544/stream=0 RTSP/1.0, headers=map[CSeq:1 Session:47112344] - server.go:58

// ffmpeg
// handleTCPConnect. conn=0xc000010030 - server.go:52
// requestLine=OPTIONS rtsp://localhost:5544/test110 RTSP/1.0, headers=map[CSeq:1 User-Agent:Lavf57.83.100] - server.go:60
// requestLine=ANNOUNCE rtsp://localhost:5544/test110 RTSP/1.0, headers=map[CSeq:2 Content-Length:280 Content-Type:application/sdp User-Agent:Lavf57.83.100] - server.go:60
// body=v=0
// o=- 0 0 IN IP6 ::1
// s=No Name
// c=IN IP6 ::1
// t=0 0
// a=tool:libavformat 57.83.100
// m=video 0 RTP/AVP 96
// a=rtpmap:96 H264/90000
// a=fmtp:96 packetization-mode=1; sprop-parameter-sets=Z2QAKKzZQHgCJ+XARAAAAwAEAAADACg8YMZY,aOvjyyLA; profile-level-id=640028
// a=control:streamid=0
// - server.go:76

// [DONE]
// rfc2326 10.1 OPTIONS
// CSeq
var ResponseOptionsTmpl = "RTSP/1.0 200 OK\r\n" +
	"Server: " + serverName + "\r\n" +
	"CSeq: %s\r\n" +
	"Public:DESCRIBE, SETUP, TEARDOWN, PLAY, PAUSE\r\n\r\n"

// rfc2326 10.2 DESCRIBE
// CSeq, Date, Content-Length,
var ResponseDescribeTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"Date: %s\r\n" +
	"Content-Type: application/sdp\r\n" +
	"Content-Length: %d\r\n\r\n" +
	SDPTmpl

// rfc4566
// * v=0 版本号，固定
// * o=<username> <sess-id> <sess-version> <nettype> <addrtype> <unicast-address>
//     <unicast-address>
// * t=<start-time> <stop-time>
// * a=
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

// rfc2326 10.4 SETUP
// CSeq, Date, Transport
// TODO chef: server_port
var ResponseSetupTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"Date: %s\r\n" +
	"Session: 47112344\r\n" +
	"Transport:%s;server_port=6256-6257\r\n\r\n"

// rfc2326 10.5 PLAY
var PlayTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n" +
	"Date: %s\r\n"

var AnnounceTmpl = "RTSP/1.0 200 OK\r\n" +
	"CSeq: %s\r\n"

func PackResponseOptions(cseq string) string {
	return fmt.Sprintf(ResponseOptionsTmpl, cseq)
}

func PackResponseDescribe(cseq string) string {
	date := time.Now().Format(time.RFC1123)
	return fmt.Sprintf(ResponseDescribeTmpl, cseq, date, 376)
}

func PackResponseSetup(cseq string, transportC string) string {
	date := time.Now().Format(time.RFC1123)
	return fmt.Sprintf(ResponseSetupTmpl, cseq, date, transportC)
}

func PackResponsePlay(cseq string) string {
	date := time.Now().Format(time.RFC1123)
	return fmt.Sprintf(PlayTmpl, cseq, date)
}

func PackResponseAnnounce(cseq string) string {
	return fmt.Sprintf(AnnounceTmpl, cseq)
}
