// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package stun

import (
	"fmt"
	"strings"

	"github.com/q191201771/naza/pkg/nazanet"
)

// TODO chef:
// - 重试

type Client struct {
}

// @param addr 填入server地址，如果不包含端口，则使用默认端口3478
func (c *Client) Query(addr string, timeoutMS int) (ip string, port int, err error) {
	if !strings.Contains(addr, ":") {
		addr = fmt.Sprintf("%s:%d", addr, DefaultPort)
	}

	uc, err := nazanet.NewUDPConnection(func(option *nazanet.UDPConnectionOption) {
		option.LAddr = addr
	})
	if err != nil {
		return "", 0, err
	}
	req, err := PackBindingRequest()
	if err != nil {
		return "", 0, err
	}
	if err := uc.Write(req); err != nil {
		return "", 0, err
	}

	b, _, err := uc.ReadWithTimeout(timeoutMS)
	if err != nil {
		return "", 0, err
	}

	_ = uc.Dispose()

	return UnpackResponseMessage(b)
}
