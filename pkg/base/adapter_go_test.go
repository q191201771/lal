// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"github.com/q191201771/naza/pkg/assert"
	"testing"
	"time"
)

func TestUnixMilli(t *testing.T) {
	n := time.Now()
	assert.Equal(t, n.UnixMilli(), UnixMilli(n))
}
