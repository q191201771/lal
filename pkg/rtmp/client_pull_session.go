// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import (
	"github.com/q191201771/lal/pkg/base"
)

type OnReadRTMPAVMsg func(msg base.RTMPMsg)

type PullSession struct {
	core *ClientSession
}

type PullSessionOption struct {
	// 从调用Pull函数，到接收音视频数据的前一步，也即收到服务端返回的rtmp play对应结果的信令的超时时间
	// 如果为0，则没有超时时间
	PullTimeoutMS int

	ReadAVTimeoutMS int
}

var defaultPullSessionOption = PullSessionOption{
	PullTimeoutMS:   0,
	ReadAVTimeoutMS: 0,
}

type ModPullSessionOption func(option *PullSessionOption)

func NewPullSession(modOptions ...ModPullSessionOption) *PullSession {
	opt := defaultPullSessionOption
	for _, fn := range modOptions {
		fn(&opt)
	}

	return &PullSession{
		core: NewClientSession(CSTPullSession, func(option *ClientSessionOption) {
			option.DoTimeoutMS = opt.PullTimeoutMS
			option.ReadAVTimeoutMS = opt.ReadAVTimeoutMS
		}),
	}
}

// 如果没有发生错误，阻塞直到到接收音视频数据的前一步，也即收到服务端返回的rtmp play对应结果的信令
//
// @param onReadRTMPAVMsg: 注意，回调结束后，内存块会被PullSession重复使用
func (s *PullSession) Pull(rawURL string, onReadRTMPAVMsg OnReadRTMPAVMsg) error {
	s.core.onReadRTMPAVMsg = onReadRTMPAVMsg
	return s.core.Do(rawURL)
}

// Pull成功后，调用该函数，可阻塞直到拉流结束
func (s *PullSession) Wait() <-chan error {
	return s.core.Wait()
}

func (s *PullSession) Dispose() {
	s.core.Dispose()
}

func (s *PullSession) UniqueKey() string {
	return s.core.UniqueKey
}

func (s *PullSession) AppName() string {
	return s.core.AppName()
}

func (s *PullSession) StreamName() string {
	return s.core.StreamName()
}

func (s *PullSession) RawQuery() string {
	return s.core.RawQuery()
}

func (s *PullSession) GetStat() base.StatSession {
	return s.core.GetStat()
}

func (s *PullSession) UpdateStat(interval uint32) {
	s.core.UpdateStat(interval)
}

func (s *PullSession) IsAlive() (readAlive, writeAlive bool) {
	return s.core.IsAlive()
}
