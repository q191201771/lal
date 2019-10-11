// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

type PushSession struct {
	*ClientSession
}

type PushSessionTimeout struct {
	ConnectTimeoutMS int
	PushTimeoutMS    int
	WriteAVTimeoutMS int
}

func NewPushSession(timeout PushSessionTimeout) *PushSession {
	return &PushSession{
		ClientSession: NewClientSession(CSTPushSession, nil, ClientSessionTimeout{
			ConnectTimeoutMS: timeout.ConnectTimeoutMS,
			DoTimeoutMS:      timeout.PushTimeoutMS,
			WriteAVTimeoutMS: timeout.WriteAVTimeoutMS,
		}),
	}
}

func (s *PushSession) Push(rawURL string) error {
	return s.doWithTimeout(rawURL)
}
