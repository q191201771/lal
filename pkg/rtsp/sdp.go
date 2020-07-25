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
)

var ErrSDP = errors.New("lal.sdp: fxxk")

type SDP struct {
	ARTPMapList   []ARTPMap
	AFmtPBaseList []AFmtPBase
}

type ARTPMap struct {
	PayloadType        int
	EncodingName       string
	ClockRate          int
	EncodingParameters string
}

type AFmtPBase struct {
	Format     int               // same as PayloadType
	Parameters map[string]string // name -> value
}

// 例子见单元测试
func ParseSDP(b []byte) (SDP, error) {
	var sdp SDP

	s := string(b)
	lines := strings.Split(s, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "a=rtpmap") {
			aRTPMap, err := ParseARTPMap(line)
			if err != nil {
				return sdp, err
			}
			sdp.ARTPMapList = append(sdp.ARTPMapList, aRTPMap)
		}
		if strings.HasPrefix(line, "a=fmtp") {
			aFmtPBase, err := ParseAFmtPBase(line)
			if err != nil {
				return sdp, err
			}
			sdp.AFmtPBaseList = append(sdp.AFmtPBaseList, aFmtPBase)
		}
	}

	return sdp, nil
}

// 例子见单元测试
func ParseARTPMap(s string) (ret ARTPMap, err error) {
	// rfc 3640 3.3.1.  General
	// rfc 3640 3.3.6.  High Bit-rate AAC
	//
	// a=rtpmap:<payload type> <encoding name>/<clock rate>[/<encoding parameters>]
	//

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

// 例子见单元测试
func ParseAFmtPBase(s string) (ret AFmtPBase, err error) {
	// rfc 3640 4.4.1.  The a=fmtp Keyword
	//
	// a=fmtp:<format> <parameter name>=<value>[; <parameter name>=<value>]
	//

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

// 例子见单元测试
func ParseSPSPPS(a AFmtPBase) (sps, pps []byte, err error) {
	if a.Format != RTPPacketTypeAVC {
		err = ErrSDP
		return
	}

	v, ok := a.Parameters["sprop-parameter-sets"]
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
