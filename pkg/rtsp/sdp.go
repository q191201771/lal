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
	"strconv"
	"strings"

	"github.com/q191201771/naza/pkg/nazalog"
)

var ErrSDP = errors.New("lal.sdp: fxxk")

type SDP struct {
}

// rfc 4566 5.14.  Media Descriptions ("m=")
// m=<media> <port> <proto> <fmt> ..
//
// example:
// m=audio 0 RTP/AVP 97
//type MediaDesc struct {
//	Media string
//	Port  string
//	Proto string
//	Fmt   string
//}

type ARTPMap struct {
	PayloadType        int
	EncodingName       string
	ClockRate          string
	EncodingParameters string
}

type FmtP struct {
	Mode string
}

func parseSDP(b []byte) SDP {
	s := string(b)
	lines := strings.Split(s, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "a=rtpmap") {
			aRTPMap, err := parseARTPMap(line)
			nazalog.Debugf("%+v, %v", aRTPMap, err)
		}
	}

	return SDP{}
}

func parseARTPMap(s string) (ret ARTPMap, err error) {
	// rfc 3640 3.3.1.  General
	// rfc 3640 3.3.6.  High Bit-rate AAC
	//
	// a=rtpmap:<payload type> <encoding name>/<clock rate>[/<encoding parameters>]
	//
	// example：
	// a=rtpmap:96 H264/90000
	// a=rtpmap:97 MPEG4-GENERIC/44100/2

	items := strings.Split(s, ":")
	if len(items) != 2 {
		err = ErrSDP
		return
	}
	items = strings.Split(items[1], " ")
	if len(items) != 2 {
		err = ErrSDP
		return
	}
	ret.PayloadType, err = strconv.Atoi(items[0])
	if err != nil {
		return
	}
	items = strings.Split(items[1], "/")
	switch len(items) {
	case 3:
		ret.EncodingParameters = items[2]
		fallthrough
	case 2:
		ret.EncodingName = items[0]
		ret.ClockRate = items[1]
	default:
		err = ErrSDP
	}
	return
}

func parseFmtP(s string) (ret ARTPMap, err error) {
	// rfc 3640 4.4.1.  The a=fmtp Keyword
	//
	// a=fmtp:<format> <parameter name>=<value>[; <parameter name>=<value>]
	//
	// example:
	// a=fmtp:96 packetization-mode=1; sprop-parameter-sets=Z2QAIKzZQMApsBEAAAMAAQAAAwAyDxgxlg==,aOvssiw=; profile-level-id=640020
	// a=fmtp:97 profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3; config=1210

	return
}
