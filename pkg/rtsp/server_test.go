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
)

func Test_parsePresentation(t *testing.T) {
	ret, err := parsePresentation("rtsp://localhost:5544/test110")
	assert.Equal(t, "test110", ret)
	assert.Equal(t, nil, err)
	ret, err = parsePresentation("rtsp://localhost:5544/live/test110")
	assert.Equal(t, "test110", ret)
	assert.Equal(t, nil, err)
	ret, err = parsePresentation("rtsp://localhost:5544/live/test110/streamid=0")
	assert.Equal(t, "test110", ret)
	assert.Equal(t, nil, err)
}
