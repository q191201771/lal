// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import "errors"

// 注意，正在学习以及实现rtsp，请不要使用这个package

// TODO chef
// - 支持tcp协议
// - pub_session生命周期如何结束
// - pub_session超时

var ErrRTSP = errors.New("lal.rtsp: fxxk")

const (
	MethodOptions  = "OPTIONS"
	MethodAnnounce = "ANNOUNCE"
	MethodSetup    = "SETUP"
	MethodRecord   = "RECORD"
	MethodTeardown = "TEARDOWN"
	MethodDescribe = "DESCRIBE"
	MethodPlay     = "PLAY"
)

const (
	HeaderFieldCSeq      = "CSeq"
	HeaderFieldTransport = "Transport"
)

var (
	// TODO chef: 参考协议标准，不要使用固定值
	sessionID = "191201771"

	minServerPort = uint16(8000)
	maxServerPort = uint16(16000)

	composerItemMaxSize = 1024
)

// ffmpeg -re -stream_loop -1 -i /Volumes/Data/tmp/test.flv -acodec copy -vcodec copy -f rtsp rtsp://localhost:5544/live/test110
// read http request. method=ANNOUNCE, uri=rtsp://localhost:5544/live/test110, headers=map[CSeq:2 Content-Length:481 Content-Type:application/sdp User-Agent:Lavf57.83.100], body=v=0
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
// - server.go:90
// read http request. method=SETUP, uri=rtsp://localhost:5544/live/test110/streamid=0, headers=map[CSeq:3 Transport:RTP/AVP/UDP;unicast;client_port=16366-16367;mode=record User-Agent:Lavf57.83.100], body= - server.go:90
// read http request. method=SETUP, uri=rtsp://localhost:5544/live/test110/streamid=1, headers=map[CSeq:4 Session:191201771 Transport:RTP/AVP/UDP;unicast;client_port=16368-16369;mode=record User-Agent:Lavf57.83.100], body= - server.go:90
// read http request. method=RECORD, uri=rtsp://localhost:5544/live/test110, headers=map[CSeq:5 Range:npt=0.000- Session:191201771 User-Agent:Lavf57.83.100], body= - server.go:90
// read http request. method=TEARDOWN, uri=rtsp://localhost:5544/live/test110, headers=map[CSeq:6 Session:191201771 User-Agent:Lavf57.83.100], body= - server.go:90

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
