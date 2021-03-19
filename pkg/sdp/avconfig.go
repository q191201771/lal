// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package sdp

import (
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/cfeeling/lal/pkg/base"
)

func ParseASC(a *AFmtPBase) ([]byte, error) {
	if a.Format != base.RTPPacketTypeAAC {
		return nil, ErrSDP
	}

	v, ok := a.Parameters["config"]
	if !ok {
		return nil, ErrSDP
	}
	if len(v) < 4 || (len(v)%2) != 0 {
		return nil, ErrSDP
	}
	l := len(v) / 2
	r := make([]byte, l)
	for i := 0; i < l; i++ {
		b, err := strconv.ParseInt(v[i*2:i*2+2], 16, 0)
		if err != nil {
			return nil, ErrSDP
		}
		r[i] = uint8(b)
	}

	return r, nil
}

func ParseVPSSPSPPS(a *AFmtPBase) (vps, sps, pps []byte, err error) {
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
func ParseSPSPPS(a *AFmtPBase) (sps, pps []byte, err error) {
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
