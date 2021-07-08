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
	"encoding/hex"
	"strings"

	"github.com/q191201771/lal/pkg/base"
)

func ParseAsc(a *AFmtPBase) ([]byte, error) {
	if a.Format != base.RtpPacketTypeAac {
		return nil, ErrSdp
	}

	v, ok := a.Parameters["config"]
	if !ok {
		return nil, ErrSdp
	}
	if len(v) < 4 || (len(v)%2) != 0 {
		return nil, ErrSdp
	}
	return hex.DecodeString(v)
}

func ParseVpsSpsPps(a *AFmtPBase) (vps, sps, pps []byte, err error) {
	v, ok := a.Parameters["sprop-vps"]
	if !ok {
		return nil, nil, nil, ErrSdp
	}
	if vps, err = base64.StdEncoding.DecodeString(v); err != nil {
		return nil, nil, nil, err
	}

	v, ok = a.Parameters["sprop-sps"]
	if !ok {
		return nil, nil, nil, ErrSdp
	}
	if sps, err = base64.StdEncoding.DecodeString(v); err != nil {
		return nil, nil, nil, err
	}

	v, ok = a.Parameters["sprop-pps"]
	if !ok {
		return nil, nil, nil, ErrSdp
	}
	if pps, err = base64.StdEncoding.DecodeString(v); err != nil {
		return nil, nil, nil, err
	}

	return
}

// 解析AVC/H264的sps，pps
// 例子见单元测试
func ParseSpsPps(a *AFmtPBase) (sps, pps []byte, err error) {
	v, ok := a.Parameters["sprop-parameter-sets"]
	if !ok {
		return nil, nil, ErrSdp
	}

	items := strings.SplitN(v, ",", 2)
	if len(items) != 2 {
		return nil, nil, ErrSdp
	}

	sps, err = base64.StdEncoding.DecodeString(items[0])
	if err != nil {
		return nil, nil, ErrSdp
	}

	pps, err = base64.StdEncoding.DecodeString(items[1])
	return
}
