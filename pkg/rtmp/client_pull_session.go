// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

type PullSessionObserver interface {
	AVMsgObserver
}

type PullSession struct {
	*ClientSession
}

type PullSessionTimeout struct {
	ConnectTimeoutMS int
	PullTimeoutMS    int
	ReadAVTimeoutMS  int
}

func NewPullSession(obs PullSessionObserver, timeout PullSessionTimeout) *PullSession {
	return &PullSession{
		ClientSession: NewClientSession(CSTPullSession, obs, ClientSessionTimeout{
			ConnectTimeoutMS: timeout.ConnectTimeoutMS,
			DoTimeoutMS:      timeout.PullTimeoutMS,
			ReadAVTimeoutMS:  timeout.ReadAVTimeoutMS,
		}),
	}
}

func (s *PullSession) Pull(rawURL string) error {
	return s.doWithTimeout(rawURL)
}
