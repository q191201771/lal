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
	"fmt"
	"net"
	"sync"

	"github.com/q191201771/naza/pkg/nazalog"
)

var ErrUDP = errors.New("lal.udp: fxxk")

// 从一段本地UDP端口范围内，取出一个还没被绑定监听使用的UDP端口，建立监听，作为UDP连接返回

type UDPServerPool struct {
	minPort uint16
	maxPort uint16

	m        sync.Mutex
	lastPort uint16
}

// 调用方保证maxPort大于等于minPort
// TODO chef: 监听ip参数
func NewUDPServerPool(minPort, maxPort uint16) *UDPServerPool {
	return &UDPServerPool{
		minPort:  minPort,
		maxPort:  maxPort,
		lastPort: minPort,
	}
}

func (u *UDPServerPool) Acquire() (*net.UDPConn, uint16, error) {
	u.m.Lock()
	defer u.m.Unlock()

	var acquired bool
	p := u.lastPort
	for {
		// 一轮试下来，也没有成功的
		if acquired && p == u.lastPort {
			return nil, 0, ErrUDP
		}
		acquired = true

		addr := fmt.Sprintf(":%d", p)

		udpAddr, err := net.ResolveUDPAddr("udp", addr)
		nazalog.Assert(nil, err)

		conn, err := net.ListenUDP("udp", udpAddr)
		if err != nil {
			if p == u.maxPort {
				p = u.minPort
			} else {
				p++
			}
			continue
		}
		return conn, p, nil
	}
}
