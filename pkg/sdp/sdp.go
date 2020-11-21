// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package sdp

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"

	"github.com/q191201771/lal/pkg/base"
)

var ErrSDP = errors.New("lal.sdp: fxxk")

const (
	ARTPMapEncodingName = "H265"
)

type LogicContext struct {
	AudioClockRate   int
	VideoClockRate   int
	AudioPayloadType base.AVPacketPT
	VideoPayloadType base.AVPacketPT
	AudioAControl    string
	VideoAControl    string
	ASC              []byte
	VPS              []byte
	SPS              []byte
	PPS              []byte
}

type RawContext struct {
	ARTPMapList   []ARTPMap
	AFmtPBaseList []AFmtPBase
	AControlList  []AControl
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

type AControl struct {
	Value string
}

func ParseSDP2LogicContext(b []byte) (LogicContext, error) {
	var ret LogicContext

	c, err := ParseSDP2RawContext(b)
	if err != nil {
		return ret, err
	}

	for i, item := range c.ARTPMapList {
		switch item.PayloadType {
		case base.RTPPacketTypeAVCOrHEVC:
			ret.VideoClockRate = item.ClockRate
			if item.EncodingName == ARTPMapEncodingName {
				ret.VideoPayloadType = base.AVPacketPTHEVC
			} else {
				ret.VideoPayloadType = base.AVPacketPTAVC
			}
			if i < len(c.AControlList) {
				ret.VideoAControl = c.AControlList[i].Value
			}
		case base.RTPPacketTypeAAC:
			ret.AudioClockRate = item.ClockRate
			ret.AudioPayloadType = base.AVPacketPTAAC
			if i < len(c.AControlList) {
				ret.AudioAControl = c.AControlList[i].Value
			}
		default:
			return ret, ErrSDP
		}
	}

	for _, item := range c.AFmtPBaseList {
		switch item.Format {
		case base.RTPPacketTypeAVCOrHEVC:
			if ret.VideoPayloadType == base.AVPacketPTHEVC {
				ret.VPS, ret.SPS, ret.PPS, err = ParseVPSSPSPPS(item)
			} else {
				ret.SPS, ret.PPS, err = ParseSPSPPS(item)
			}
			if err != nil {
				return ret, err
			}
		case base.RTPPacketTypeAAC:
			ret.ASC, err = ParseASC(item)
			if err != nil {
				return ret, err
			}
		default:
			return ret, ErrSDP
		}
	}

	return ret, nil
}

// 例子见单元测试
func ParseSDP2RawContext(b []byte) (RawContext, error) {
	var sdpCtx RawContext

	s := string(b)
	lines := strings.Split(s, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "a=rtpmap") {
			aRTPMap, err := ParseARTPMap(line)
			if err != nil {
				return sdpCtx, err
			}
			sdpCtx.ARTPMapList = append(sdpCtx.ARTPMapList, aRTPMap)
		}
		if strings.HasPrefix(line, "a=fmtp") {
			aFmtPBase, err := ParseAFmtPBase(line)
			if err != nil {
				return sdpCtx, err
			}
			sdpCtx.AFmtPBaseList = append(sdpCtx.AFmtPBaseList, aFmtPBase)
		}
		if strings.HasPrefix(line, "a=control") {
			aControl, err := ParseAControl(line)
			if err != nil {
				return sdpCtx, err
			}
			sdpCtx.AControlList = append(sdpCtx.AControlList, aControl)
		}
	}

	return sdpCtx, nil
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

func ParseAControl(s string) (ret AControl, err error) {
	if !strings.HasPrefix(s, "a=control:") {
		err = ErrSDP
		return
	}
	ret.Value = strings.TrimPrefix(s, "a=control:")
	return
}

func ParseASC(a AFmtPBase) ([]byte, error) {
	if a.Format != base.RTPPacketTypeAAC {
		return nil, ErrSDP
	}

	v, ok := a.Parameters["config"]
	if !ok {
		return nil, ErrSDP
	}
	if len(v) < 4 {
		return nil, ErrSDP
	}

	f, err := strconv.ParseInt(v[0:2], 16, 0)
	if err != nil {
		return nil, ErrSDP
	}
	s, err := strconv.ParseInt(v[2:4], 16, 0)
	if err != nil {
		return nil, ErrSDP
	}
	r := make([]byte, 2)
	r[0] = uint8(f)
	r[1] = uint8(s)
	return r, nil
}

func ParseVPSSPSPPS(a AFmtPBase) (vps, sps, pps []byte, err error) {
	if a.Format != base.RTPPacketTypeAVCOrHEVC {
		return nil, nil, nil, ErrSDP
	}

	v, ok := a.Parameters["sprop-vps"]
	if !ok {
		return nil, nil, nil, ErrSDP
	}
	if vps, err = base64.StdEncoding.DecodeString(v); err != nil {
		return nil, nil, nil, err
	}

	v, ok = a.Parameters["sprop-sps"]
	if !ok {
		return nil, nil, nil, ErrSDP
	}
	if sps, err = base64.StdEncoding.DecodeString(v); err != nil {
		return nil, nil, nil, err
	}

	v, ok = a.Parameters["sprop-pps"]
	if !ok {
		return nil, nil, nil, ErrSDP
	}
	if pps, err = base64.StdEncoding.DecodeString(v); err != nil {
		return nil, nil, nil, err
	}

	return
}

// 解析AVC/H264的sps，pps
// 例子见单元测试
func ParseSPSPPS(a AFmtPBase) (sps, pps []byte, err error) {
	if a.Format != base.RTPPacketTypeAVCOrHEVC {
		return nil, nil, ErrSDP
	}

	v, ok := a.Parameters["sprop-parameter-sets"]
	if !ok {
		return nil, nil, ErrSDP
	}

	items := strings.SplitN(v, ",", 2)
	if len(items) != 2 {
		return nil, nil, ErrSDP
	}

	sps, err = base64.StdEncoding.DecodeString(items[0])
	if err != nil {
		return nil, nil, ErrSDP
	}

	pps, err = base64.StdEncoding.DecodeString(items[1])
	return
}
