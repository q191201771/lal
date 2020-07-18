// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"sync"

	"github.com/q191201771/naza/pkg/nazalog"
)

type RTPServer struct {
	udpServer *UDPServer

	m            sync.Mutex
	ssrc2Session map[uint32]*Session
}

func NewRTPServer(addr string) *RTPServer {
	var s RTPServer
	s.udpServer = NewUDPServer(addr, s.OnReadUDPPacket)
	s.ssrc2Session = make(map[uint32]*Session)
	return &s
}

func (r *RTPServer) OnReadUDPPacket(b []byte, addr string, err error) {
	//nazalog.Debugf("< R length=%d, remote=%s, err=%v", len(b), addr, err)
	h, err := parseRTPPacket(b)
	if err != nil {
		nazalog.Errorf("read invalid rtp packet. err=%+v", err)
	}
	var rtpPacket RTPPacket
	rtpPacket.header = h
	rtpPacket.raw = b
	switch h.packetType {
	case RTPPacketTypeAAC:
		s := r.getOrCreateSession(h)
		s.FeedAACPacket(rtpPacket)
	case RTPPacketTypeAVC:
		nazalog.Debugf("header=%+v, length=%d", h, len(b))
		s := r.getOrCreateSession(h)
		s.FeedAVCPacket(rtpPacket)
	}
}

func (r *RTPServer) Listen() (err error) {
	nazalog.Infof("start rtp server listen. addr=%s", r.udpServer.addr)
	return r.udpServer.Listen()
}

func (r *RTPServer) RunLoop() error {
	return r.udpServer.RunLoop()
}

func (r *RTPServer) getOrCreateSession(h RTPHeader) *Session {
	r.m.Lock()
	defer r.m.Unlock()
	s, ok := r.ssrc2Session[h.ssrc]
	if ok {
		return s
	}
	return NewSession(h.ssrc, isAudio(h.packetType))
}
