// Copyright 2021, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"testing"

	"github.com/cfeeling/naza/pkg/nazalog"
)

func TestStreamExist(t *testing.T) {
	err := StreamExist("rtmp://127.0.0.1/live/test110")
	nazalog.Errorf("%+v", err)
}
