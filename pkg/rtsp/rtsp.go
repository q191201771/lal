// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"errors"

	"github.com/q191201771/naza/pkg/nazanet"
)

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

	unpackerItemMaxSize = 1024
)

var availUDPConnPool *nazanet.AvailUDPConnPool

func init() {
	availUDPConnPool = nazanet.NewAvailUDPConnPool(minServerPort, maxServerPort)
}

// ---------------------------------------------------------------------------------------------------------------------
// ffmpeg -re -stream_loop -1 -i /Volumes/Data/tmp/wontcry.flv -acodec copy -vcodec copy -f rtsp rtsp://localhost:5544/live/test110

// read http request. method=OPTIONS, uri=rtsp://localhost:5544/live/test110, headers=map[CSeq:1 User-Agent:Lavf57.83.100], body= - server.go:95
//
// read http request. method=ANNOUNCE, uri=rtsp://localhost:5544/live/test110, headers=map[CSeq:2 Content-Length:490 Content-Type:application/sdp User-Agent:Lavf57.83.100], body=v=0
// o=- 0 0 IN IP4 127.0.0.1
// s=No Name
// c=IN IP4 127.0.0.1
// t=0 0
// a=tool:libavformat 57.83.100
// m=video 0 RTP/AVP 96
// a=rtpmap:96 H264/90000
// a=fmtp:96 packetization-mode=1; sprop-parameter-sets=Z2QAFqyyAUBf8uAiAAADAAIAAAMAPB4sXJA=,aOvDyyLA; profile-level-id=640016
// a=control:streamid=0
// m=audio 0 RTP/AVP 97
// b=AS:128
// a=rtpmap:97 MPEG4-GENERIC/44100/2
// a=fmtp:97 profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3; config=121056E500
// a=control:streamid=1
// - server.go:95
//
// read http request. method=SETUP, uri=rtsp://localhost:5544/live/test110/streamid=0, headers=map[CSeq:3 Transport:RTP/AVP/UDP;unicast;client_port=32182-32183;mode=record User-Agent:Lavf57.83.100], body= - server.go:95
//
// read http request. method=SETUP, uri=rtsp://localhost:5544/live/test110/streamid=1, headers=map[CSeq:4 Session:191201771 Transport:RTP/AVP/UDP;unicast;client_port=32184-32185;mode=record User-Agent:Lavf57.83.100], body= - server.go:95
//
// read http request. method=RECORD, uri=rtsp://localhost:5544/live/test110, headers=map[CSeq:5 Range:npt=0.000- Session:191201771 User-Agent:Lavf57.83.100], body= - server.go:95
//
// read http request. method=TEARDOWN, uri=rtsp://localhost:5544/live/test110, headers=map[CSeq:6 Session:191201771 User-Agent:Lavf57.83.100], body= - server.go:95

// read udp packet failed. err=read udp [::]:8002: use of closed network connection - server_pub_session.go:199
// read udp packet failed. err=read udp [::]:8003: use of closed network connection - server_pub_session.go:199
// ---------------------------------------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------------------------------------
// read http request. method=OPTIONS, uri=rtsp://localhost:5544/live/test110, headers=map[CSeq:1 User-Agent:Lavf57.83.100], body= - server.go:108
// read http request. method=DESCRIBE, uri=rtsp://localhost:5544/live/test110, headers=map[Accept:application/sdp CSeq:2 User-Agent:Lavf57.83.100], body= - server.go:108
// read http request. method=SETUP, uri=rtsp://localhost:5544/live/test110/streamid=0, headers=map[CSeq:3 Transport:RTP/AVP/UDP;unicast;client_port=15690-15691 User-Agent:Lavf57.83.100], body= - server.go:108
// read http request. method=SETUP, uri=rtsp://localhost:5544/live/test110/streamid=1, headers=map[CSeq:4 Session:191201771 Transport:RTP/AVP/UDP;unicast;client_port=15692-15693 User-Agent:Lavf57.83.100], body= - server.go:108
// read http request. method=PLAY, uri=rtsp://localhost:5544/live/test110, headers=map[CSeq:5 Range:npt=0.000- Session:191201771 User-Agent:Lavf57.83.100], body= - server.go:108
// ---------------------------------------------------------------------------------------------------------------------

// 8000 video rtp
// 8001 video rtcp
// 8002 audio rtp
// 8003 audio rtcp
