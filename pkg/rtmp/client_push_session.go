// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import "github.com/q191201771/lal/pkg/base"

type PushSession struct {
	IsFresh bool

	core *ClientSession
}

type PushSessionOption struct {
	// 从调用Push函数，到可以发送音视频数据的前一步，也即收到服务端返回的rtmp publish对应结果的信令的超时时间
	// 如果为0，则没有超时时间
	PushTimeoutMs int

	WriteAvTimeoutMs     int
	HandshakeComplexFlag bool
}

var defaultPushSessionOption = PushSessionOption{
	PushTimeoutMs:    10000,
	WriteAvTimeoutMs: 0,
}

type ModPushSessionOption func(option *PushSessionOption)

func NewPushSession(modOptions ...ModPushSessionOption) *PushSession {
	opt := defaultPushSessionOption
	for _, fn := range modOptions {
		fn(&opt)
	}
	return &PushSession{
		IsFresh: true,
		core: NewClientSession(CstPushSession, func(option *ClientSessionOption) {
			option.DoTimeoutMs = opt.PushTimeoutMs
			option.WriteAvTimeoutMs = opt.WriteAvTimeoutMs
			option.HandshakeComplexFlag = opt.HandshakeComplexFlag
		}),
	}
}

// 阻塞直到和对端完成推流前，握手部分的工作（也即收到RTMP Publish response），或者发生错误
func (s *PushSession) Push(rawUrl string) error {
	return s.core.Do(rawUrl)
}

// 发送数据
// 注意，业务方需将数据打包成rtmp chunk格式后，再调用该函数发送
func (s *PushSession) Write(msg []byte) error {
	return s.core.Write(msg)
}

// 将缓存的数据立即刷新发送
// 是否有缓存策略，请参见配置及内部实现
func (s *PushSession) Flush() error {
	return s.core.Flush()
}

// 文档请参考： interface IClientSessionLifecycle
func (s *PushSession) Dispose() error {
	return s.core.Dispose()
}

// 文档请参考： interface IClientSessionLifecycle
func (s *PushSession) WaitChan() <-chan error {
	return s.core.WaitChan()
}

// 文档请参考： interface ISessionUrlContext
func (s *PushSession) Url() string {
	return s.core.Url()
}

// 文档请参考： interface ISessionUrlContext
func (s *PushSession) AppName() string {
	return s.core.AppName()
}

// 文档请参考： interface ISessionUrlContext
func (s *PushSession) StreamName() string {
	return s.core.StreamName()
}

// 文档请参考： interface ISessionUrlContext
func (s *PushSession) RawQuery() string {
	return s.core.RawQuery()
}

// 文档请参考： interface IObject
func (s *PushSession) UniqueKey() string {
	return s.core.uniqueKey
}

// 文档请参考： interface ISessionStat
func (s *PushSession) GetStat() base.StatSession {
	return s.core.GetStat()
}

// 文档请参考： interface ISessionStat
func (s *PushSession) UpdateStat(intervalSec uint32) {
	s.core.UpdateStat(intervalSec)
}

// 文档请参考： interface ISessionStat
func (s *PushSession) IsAlive() (readAlive, writeAlive bool) {
	return s.core.IsAlive()
}
