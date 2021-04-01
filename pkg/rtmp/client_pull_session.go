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
	PullTimeoutMS:   10000,
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

// 阻塞直到和对端完成拉流前，握手部分的工作（也即收到RTMP Play response），或者发生错误
//
// @param onReadRTMPAVMsg: 注意，回调结束后，内存块会被PullSession重复使用
func (s *PullSession) Pull(rawURL string, onReadRTMPAVMsg OnReadRTMPAVMsg) error {
	s.core.onReadRTMPAVMsg = onReadRTMPAVMsg
	return s.core.Do(rawURL)
}

// 文档请参考： interface IClientSessionLifecycle
func (s *PullSession) Dispose() error {
	return s.core.Dispose()
}

// 文档请参考： interface IClientSessionLifecycle
func (s *PullSession) WaitChan() <-chan error {
	return s.core.WaitChan()
}

// 文档请参考： interface ISessionURLContext
func (s *PullSession) URL() string {
	return s.core.URL()
}

// 文档请参考： interface ISessionURLContext
func (s *PullSession) AppName() string {
	return s.core.AppName()
}

// 文档请参考： interface ISessionURLContext
func (s *PullSession) StreamName() string {
	return s.core.StreamName()
}

// 文档请参考： interface ISessionURLContext
func (s *PullSession) RawQuery() string {
	return s.core.RawQuery()
}

// 文档请参考： interface IObject
func (s *PullSession) UniqueKey() string {
	return s.core.uniqueKey
}

// 文档请参考： interface ISessionStat
func (s *PullSession) GetStat() base.StatSession {
	return s.core.GetStat()
}

// 文档请参考： interface ISessionStat
func (s *PullSession) UpdateStat(intervalSec uint32) {
	s.core.UpdateStat(intervalSec)
}

// 文档请参考： interface ISessionStat
func (s *PullSession) IsAlive() (readAlive, writeAlive bool) {
	return s.core.IsAlive()
}
