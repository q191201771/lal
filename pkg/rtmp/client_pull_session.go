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

type OnReadRtmpAvMsg func(msg base.RtmpMsg)

type PullSession struct {
	core *ClientSession
}

type PullSessionOption struct {
	// 从调用Pull函数，到接收音视频数据的前一步，也即收到服务端返回的rtmp play对应结果的信令的超时时间
	// 如果为0，则没有超时时间
	PullTimeoutMs int

	ReadAvTimeoutMs      int
	HandshakeComplexFlag bool
}

var defaultPullSessionOption = PullSessionOption{
	PullTimeoutMs:   10000,
	ReadAvTimeoutMs: 0,
}

type ModPullSessionOption func(option *PullSessionOption)

func NewPullSession(modOptions ...ModPullSessionOption) *PullSession {
	opt := defaultPullSessionOption
	for _, fn := range modOptions {
		fn(&opt)
	}

	return &PullSession{
		core: NewClientSession(CstPullSession, func(option *ClientSessionOption) {
			option.DoTimeoutMs = opt.PullTimeoutMs
			option.ReadAvTimeoutMs = opt.ReadAvTimeoutMs
			option.HandshakeComplexFlag = opt.HandshakeComplexFlag
		}),
	}
}

// 阻塞直到和对端完成拉流前，握手部分的工作（也即收到RTMP Play response），或者发生错误
//
// @param onReadRtmpAvMsg: 注意，回调结束后，内存块会被PullSession重复使用
func (s *PullSession) Pull(rawUrl string, onReadRtmpAvMsg OnReadRtmpAvMsg) error {
	s.core.onReadRtmpAvMsg = onReadRtmpAvMsg
	return s.core.Do(rawUrl)
}

// 文档请参考： interface IClientSessionLifecycle
func (s *PullSession) Dispose() error {
	return s.core.Dispose()
}

// 文档请参考： interface IClientSessionLifecycle
func (s *PullSession) WaitChan() <-chan error {
	return s.core.WaitChan()
}

// 文档请参考： interface ISessionUrlContext
func (s *PullSession) Url() string {
	return s.core.Url()
}

// 文档请参考： interface ISessionUrlContext
func (s *PullSession) AppName() string {
	return s.core.AppName()
}

// 文档请参考： interface ISessionUrlContext
func (s *PullSession) StreamName() string {
	return s.core.StreamName()
}

// 文档请参考： interface ISessionUrlContext
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
