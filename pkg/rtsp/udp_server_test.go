// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"testing"

	"github.com/q191201771/naza/pkg/assert"
	"github.com/q191201771/naza/pkg/nazalog"
)

func TestUDPServer_Listen(t *testing.T) {
	s := NewUDPServerWithAddr(":8000", func(b []byte, addr string, err error) {

	})
	err := s.Listen()
	nazalog.Debugf("%+v", err)

	s2 := NewUDPServerWithAddr(":8000", func(b []byte, addr string, err error) {

	})
	err = s2.Listen()
	assert.IsNotNil(t, err)
	nazalog.Debugf("%+v", err)
}
