// Copyright 2023, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import (
	"encoding/hex"
	"github.com/q191201771/naza/pkg/assert"
	"github.com/q191201771/naza/pkg/nazabytes"
	"net"
	"testing"
	"time"
)

type testServerSessionObserver struct {
}

func (o *testServerSessionObserver) OnRtmpConnect(session *ServerSession, opa ObjectPairArray) {

}

func (o *testServerSessionObserver) OnNewRtmpPubSession(session *ServerSession) error {
	return nil
}

func (o *testServerSessionObserver) OnNewRtmpSubSession(session *ServerSession) error {
	return nil
}

// 考虑加入naza中
type mConn struct{}

func (mConn) Read(b []byte) (n int, err error) {
	//TODO implement me
	panic("implement me")
}

func (mConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (mConn) Close() error {
	//TODO implement me
	panic("implement me")
}

func (mConn) LocalAddr() net.Addr {
	//TODO implement me
	panic("implement me")
}

func (mConn) RemoteAddr() net.Addr {
	addrs, _ := net.InterfaceAddrs()
	return addrs[0]
}

func (mConn) SetDeadline(t time.Time) error {
	//TODO implement me
	panic("implement me")
}

func (mConn) SetReadDeadline(t time.Time) error {
	//TODO implement me
	panic("implement me")
}

func (mConn) SetWriteDeadline(t time.Time) error {
	//TODO implement me
	panic("implement me")
}

func TestServerSession_doMsg(t *testing.T) {
	var o testServerSessionObserver
	var c mConn
	s := NewServerSession(&o, &c)

	var stream Stream
	stream.msg.buff = nazabytes.NewBuffer(1024)

	// publish信令中没有pub type
	// {Csid:5 MsgLen:39 MsgTypeId:20 MsgStreamId:1 TimestampAbs:0}
	stream.header.MsgTypeId = 20
	//b, _ := hex.DecodeString("0200077075626c69736800400800000000000005020009696e6e6572746573740200046c697665")
	b, _ := hex.DecodeString("0200077075626c69736800400800000000000005020009696e6e657274657374")
	stream.msg.buff.Write(b)

	err := s.doMsg(&stream)
	assert.Equal(t, nil, err)
}
