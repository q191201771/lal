// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazalog"
)

type PullSession struct {
	UniqueKey string // const after ctor
}

func NewPullSession() *PullSession {
	uk := base.GenUniqueKey(base.UKPRTSPPullSession)
	s := &PullSession{
		UniqueKey: uk,
	}
	nazalog.Infof("[%s] lifecycle new rtsp PullSession. session=%p", uk, s)
	return s
}

func (session *PullSession) Pull(rawURL string) error {

	return nil
}

func (session *PullSession) Connect(rawURL string) error {

	return nil
}
