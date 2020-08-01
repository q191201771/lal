// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import "github.com/q191201771/lal/pkg/base"

type OnReadRTMPAVMsg func(msg base.RTMPMsg)

type PullSession struct {
	core *ClientSession
}

type PullSessionOption struct {
	ConnectTimeoutMS int
	PullTimeoutMS    int
	ReadAVTimeoutMS  int
}

var defaultPullSessionOption = PullSessionOption{
	ConnectTimeoutMS: 0,
	PullTimeoutMS:    0,
	ReadAVTimeoutMS:  0,
}

type ModPullSessionOption func(option *PullSessionOption)

func NewPullSession(modOptions ...ModPullSessionOption) *PullSession {
	opt := defaultPullSessionOption
	for _, fn := range modOptions {
		fn(&opt)
	}

	return &PullSession{
		core: NewClientSession(CSTPullSession, func(option *ClientSessionOption) {
			option.ConnectTimeoutMS = opt.ConnectTimeoutMS
			option.DoTimeoutMS = opt.PullTimeoutMS
			option.ReadAVTimeoutMS = opt.ReadAVTimeoutMS
		}),
	}
}

// 建立rtmp play连接
// 阻塞直到收到服务端返回的rtmp publish对应结果的信令，或发生错误
//
// @param onReadRTMPAVMsg: 注意，回调结束后，内存块会被PullSession重复使用
func (s *PullSession) Pull(rawURL string, onReadRTMPAVMsg OnReadRTMPAVMsg) error {
	s.core.onReadRTMPAVMsg = onReadRTMPAVMsg
	return s.core.doWithTimeout(rawURL)
}

func (s *PullSession) Done() <-chan error {
	return s.core.Done()
}

func (s *PullSession) Dispose() {
	s.core.Dispose()
}

func (s *PullSession) UniqueKey() string {
	return s.core.UniqueKey
}
