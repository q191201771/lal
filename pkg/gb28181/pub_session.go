// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package gb28181

import (
	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/naza/pkg/nazanet"
	"net"
)

type PubSession struct {
	conn *nazanet.UdpConnection
	rtprtcp.IRtpUnpacker
}

func NewPubSession() *PubSession {
	return &PubSession{}
}

func (session *PubSession) RunLoop(addr string) error {
	var err error
	session.conn, err = nazanet.NewUdpConnection(func(option *nazanet.UdpConnectionOption) {
		option.LAddr = addr
	})
	if err != nil {
		return err
	}
	err = session.conn.RunLoop(func(b []byte, raddr *net.UDPAddr, err error) bool {
		return true
	})
	return err
}
