// Copyright 2019, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import (
	"github.com/cfeeling/lal/pkg/base"
)

type PushSession struct {
	IsFresh bool

	core *ClientSession
}

type PushSessionOption struct {
	// 从调用Push函数，到可以发送音视频数据的前一步，也即收到服务端返回的rtmp publish对应结果的信令的超时时间
	// 如果为0，则没有超时时间
	PushTimeoutMS int

	WriteAVTimeoutMS int
}

var defaultPushSessionOption = PushSessionOption{
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
		IsFresh: true,
		core: NewClientSession(CSTPushSession, func(option *ClientSessionOption) {
			option.DoTimeoutMS = opt.PushTimeoutMS
			option.WriteAVTimeoutMS = opt.WriteAVTimeoutMS
		}),
	}
}

// 如果没有错误发生，阻塞到接收音视频数据的前一步，也即收到服务端返回的rtmp publish对应结果的信令
func (s *PushSession) Push(rawURL string) error {
	return s.core.Do(rawURL)
}

func (s *PushSession) AsyncWrite(msg []byte) error {
	return s.core.AsyncWrite(msg)
}

func (s *PushSession) Flush() error {
	return s.core.Flush()
}

func (s *PushSession) Dispose() {
	s.core.Dispose()
}

func (s *PushSession) GetStat() base.StatSession {
	return s.core.GetStat()
}

func (s *PushSession) UpdateStat(interval uint32) {
	s.core.UpdateStat(interval)
}

func (s *PushSession) IsAlive() (readAlive, writeAlive bool) {
	return s.core.IsAlive()
}

func (s *PushSession) AppName() string {
	return s.core.AppName()
}

func (s *PushSession) StreamName() string {
	return s.core.StreamName()
}

func (s *PushSession) RawQuery() string {
	return s.core.RawQuery()
}

func (s *PushSession) Wait() <-chan error {
	return s.core.Wait()
}

func (s *PushSession) UniqueKey() string {
	return s.core.UniqueKey
}
