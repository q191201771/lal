// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import "sync/atomic"

type PushSessionStatus uint32

const (
	PushSessionStatusInit PushSessionStatus = iota
	PushSessionStatusConnecting
	PushSessionStatusConnected
	PushSessionStatusError
)

type PushSession struct {
	core   *ClientSession
	status uint32
}

type PushSessionOption struct {
	ConnectTimeoutMS int
	PushTimeoutMS    int
	WriteAVTimeoutMS int
}

var defaultPushSessionOption = PushSessionOption{
	ConnectTimeoutMS: 0,
	PushTimeoutMS:    0,
	WriteAVTimeoutMS: 0,
}

type ModPushSessionOption func(option *PushSessionOption)

func NewPushSession(modOptions ...ModPushSessionOption) *PushSession {
	opt := defaultPushSessionOption
	for _, fn := range modOptions {
		fn(&opt)
	}
	return &PushSession{
		core: NewClientSession(CSTPushSession, func(option *ClientSessionOption) {
			option.ConnectTimeoutMS = opt.ConnectTimeoutMS
			option.DoTimeoutMS = opt.PushTimeoutMS
			option.WriteAVTimeoutMS = opt.WriteAVTimeoutMS
		}),
		status: 0,
	}
}

// 阻塞直到收到服务端返回的 rtmp publish 对应结果的信令或发生错误
func (s *PushSession) Push(rawURL string) error {
	s.setStatus(PushSessionStatusConnecting)
	err := s.core.doWithTimeout(rawURL)
	if err == nil {
		s.setStatus(PushSessionStatusConnected)
	} else {
		s.setStatus(PushSessionStatusError)
	}
	return err
}

func (s *PushSession) AsyncWrite(msg []byte) error {
	err := s.core.AsyncWrite(msg)
	if err != nil {
		s.setStatus(PushSessionStatusError)
	}
	return err
}

func (s *PushSession) Flush() error {
	err := s.core.Flush()
	if err != nil {
		s.setStatus(PushSessionStatusError)
	}
	return err
}

func (s *PushSession) Dispose() {
	s.setStatus(PushSessionStatusError)
	s.core.Dispose()
}

func (s *PushSession) Status() PushSessionStatus {
	v := atomic.LoadUint32(&s.status)
	return PushSessionStatus(v)
}

func (s *PushSession) Done() <-chan error {
	return s.core.Done()
}

func (s *PushSession) UniqueKey() string {
	return s.core.UniqueKey
}

func (s *PushSession) setStatus(status PushSessionStatus) {
	i := uint32(status)
	atomic.StoreUint32(&s.status, i)
}
