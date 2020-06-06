// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"github.com/q191201771/lal/pkg/rtsp"
	"github.com/q191201771/naza/pkg/nazalog"
)

func main() {
	s := rtsp.NewServer(":5544")
	err := s.Listen()
	nazalog.Assert(nil, err)
	err = s.RunLoop()
	nazalog.Error(err)
}
