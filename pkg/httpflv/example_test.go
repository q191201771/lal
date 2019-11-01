// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv_test

import (
	"testing"

	"github.com/q191201771/lal/pkg/httpflv"
	log "github.com/q191201771/naza/pkg/nazalog"
)

// TODO chef: 后续加个 httpflv post 在做完整流程测试吧

var (
	serverAddr = ":10001"
	pullURL    = "http://127.0.0.1:10001/live/11111.flv"
)

type MockServerObserver struct {
}

func (so *MockServerObserver) NewHTTPFLVSubSessionCB(session *httpflv.SubSession) bool {
	return true
}

func (so *MockServerObserver) DelHTTPFLVSubSessionCB(session *httpflv.SubSession) {
}

func TestExample(t *testing.T) {
	var err error

	var so MockServerObserver
	s := httpflv.NewServer(&so, serverAddr)
	go s.RunLoop()

	pullSession := httpflv.NewPullSession(func(option *httpflv.PullSessionOption) {
		option.ConnectTimeoutMS = 1000
		option.ReadTimeoutMS = 1000
	})
	err = pullSession.Pull(pullURL, func(tag httpflv.Tag) {
	})
	log.Debugf("pull failed. err=%+v", err)
}
