// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"encoding/base64"
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
	ClockRate          int
	EncodingParameters string
}

type FmtPBase struct {
	Format     int               // same as PayloadType
	Parameters map[string]string // name -> value
}

func ParseSDP(b []byte) SDP {
	s := string(b)
	lines := strings.Split(s, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "a=rtpmap") {
			aRTPMap, err := ParseARTPMap(line)
			nazalog.Debugf("%+v, %v", aRTPMap, err)
		}
		if strings.HasPrefix(line, "a=fmtp") {
			fmtPBase, err := ParseFmtPBase(line)
			nazalog.Debugf("%+v, %v", fmtPBase, err)
		}
	}

	return SDP{}
}

func ParseARTPMap(s string) (ret ARTPMap, err error) {
	// rfc 3640 3.3.1.  General
	// rfc 3640 3.3.6.  High Bit-rate AAC
	//
	// a=rtpmap:<payload type> <encoding name>/<clock rate>[/<encoding parameters>]
	//
	// example see unit test

	items := strings.SplitN(s, ":", 2)
	if len(items) != 2 {
		err = ErrSDP
		return
	}
	items = strings.SplitN(items[1], " ", 2)
	if len(items) != 2 {
		err = ErrSDP
		return
	}
	ret.PayloadType, err = strconv.Atoi(items[0])
	if err != nil {
		return
	}
	items = strings.SplitN(items[1], "/", 3)
	switch len(items) {
	case 3:
		ret.EncodingParameters = items[2]
		fallthrough
	case 2:
		ret.EncodingName = items[0]
		ret.ClockRate, err = strconv.Atoi(items[1])
		if err != nil {
			return
		}
	default:
		err = ErrSDP
	}
	return
}

func ParseFmtPBase(s string) (ret FmtPBase, err error) {
	// rfc 3640 4.4.1.  The a=fmtp Keyword
	//
	// a=fmtp:<format> <parameter name>=<value>[; <parameter name>=<value>]
	//
	// example see unit test

	ret.Parameters = make(map[string]string)

	items := strings.SplitN(s, ":", 2)
	if len(items) != 2 {
		err = ErrSDP
		return
	}

	items = strings.SplitN(items[1], " ", 2)
	if len(items) != 2 {
		err = ErrSDP
		return
	}

	ret.Format, err = strconv.Atoi(items[0])
	if err != nil {
		return
	}

	items = strings.Split(items[1], ";")
	for _, pp := range items {
		pp = strings.TrimSpace(pp)
		kv := strings.SplitN(pp, "=", 2)
		if len(kv) != 2 {
			err = ErrSDP
			return
		}
		ret.Parameters[kv[0]] = kv[1]
	}

	return
}

func ParseSPSPPS(f FmtPBase) (sps, pps []byte, err error) {
	if f.Format != RTPPacketTypeAVC {
		err = ErrSDP
		return
	}

	v, ok := f.Parameters["sprop-parameter-sets"]
	if !ok {
		err = ErrSDP
		return
	}

	items := strings.SplitN(v, ",", 2)
	if len(items) != 2 {
		err = ErrSDP
		return
	}

	sps, err = base64.StdEncoding.DecodeString(items[0])
	if err != nil {
		return
	}

	pps, err = base64.StdEncoding.DecodeString(items[1])

	return
}
