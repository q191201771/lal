// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"encoding/hex"
	"net"
	"time"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazanet"
)

// to be continued
// 注意，音频和视频是不同的UDP连接
// pub和sub挂载转发时，需要对应上

type SubSession struct {
	UniqueKey  string
	StreamName string

	rtpConn  *nazanet.UDPConnection
	rtcpConn *nazanet.UDPConnection
	stat     base.StatPub
}

func NewSubSession(streamName string) *SubSession {
	uk := base.GenUniqueKey(base.UKPRTSPSubSession)
	ss := &SubSession{
		UniqueKey:  uk,
		StreamName: streamName,
		stat: base.StatPub{
			StatSession: base.StatSession{
				Protocol:  base.ProtocolRTSP,
				StartTime: time.Now().Format("2006-01-02 15:04:05.999"),
			},
		},
	}
	nazalog.Infof("[%s] lifecycle new rtsp PubSession. session=%p, streamName=%s", uk, ss, streamName)
	return ss
}

func (s *SubSession) SetRTPConn(conn *nazanet.UDPConnection) {
	s.rtpConn = conn
	go s.rtpConn.RunLoop(s.onReadUDPPacket)
}

func (s *SubSession) SetRTCPConn(conn *nazanet.UDPConnection) {
	s.rtcpConn = conn
	go s.rtcpConn.RunLoop(s.onReadUDPPacket)
}

func (s *SubSession) onReadUDPPacket(b []byte, rAddr *net.UDPAddr, err error) bool {
	nazalog.Debugf("SubSession::onReadUDPPacket. %s", hex.Dump(b))
	return true
}

func (s *SubSession) WriteRawRTPPacket(b []byte) {
	if err := s.rtpConn.Write(b); err != nil {
		nazalog.Errorf("err=%+v", err)
	}
}
