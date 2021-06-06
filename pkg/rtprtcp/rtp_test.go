// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtprtcp_test

import (
	"testing"

	"github.com/q191201771/lal/pkg/rtprtcp"

	"github.com/q191201771/naza/pkg/assert"
)

func TestCompareSeq(t *testing.T) {
	assert.Equal(t, 0, rtprtcp.CompareSeq(0, 0))
	assert.Equal(t, 0, rtprtcp.CompareSeq(1024, 1024))
	assert.Equal(t, 0, rtprtcp.CompareSeq(65535, 65535))

	assert.Equal(t, 1, rtprtcp.CompareSeq(1, 0))
	assert.Equal(t, 1, rtprtcp.CompareSeq(16383, 0))

	assert.Equal(t, -1, rtprtcp.CompareSeq(16384, 0))
	assert.Equal(t, -1, rtprtcp.CompareSeq(65534, 0))
	assert.Equal(t, -1, rtprtcp.CompareSeq(65535, 0))
	assert.Equal(t, -1, rtprtcp.CompareSeq(65534, 1))
	assert.Equal(t, -1, rtprtcp.CompareSeq(65535, 1))

	assert.Equal(t, -1, rtprtcp.CompareSeq(0, 1))
	assert.Equal(t, -1, rtprtcp.CompareSeq(0, 16383))

	assert.Equal(t, 1, rtprtcp.CompareSeq(0, 16384))
	assert.Equal(t, 1, rtprtcp.CompareSeq(0, 65534))
	assert.Equal(t, 1, rtprtcp.CompareSeq(0, 65535))
	assert.Equal(t, 1, rtprtcp.CompareSeq(1, 65534))
	assert.Equal(t, 1, rtprtcp.CompareSeq(1, 65535))
}

func TestSubSeq(t *testing.T) {
	assert.Equal(t, 0, rtprtcp.SubSeq(0, 0))
	assert.Equal(t, 0, rtprtcp.SubSeq(1024, 1024))
	assert.Equal(t, 0, rtprtcp.SubSeq(65535, 65535))

	assert.Equal(t, 1, rtprtcp.SubSeq(1, 0))
	assert.Equal(t, 16383, rtprtcp.SubSeq(16383, 0))

	assert.Equal(t, -49152, rtprtcp.SubSeq(16384, 0))
	assert.Equal(t, -2, rtprtcp.SubSeq(65534, 0))
	assert.Equal(t, -1, rtprtcp.SubSeq(65535, 0))
	assert.Equal(t, -3, rtprtcp.SubSeq(65534, 1))
	assert.Equal(t, -2, rtprtcp.SubSeq(65535, 1))

	assert.Equal(t, -1, rtprtcp.SubSeq(0, 1))
	assert.Equal(t, -16383, rtprtcp.SubSeq(0, 16383))

	assert.Equal(t, 49152, rtprtcp.SubSeq(0, 16384))
	assert.Equal(t, 2, rtprtcp.SubSeq(0, 65534))
	assert.Equal(t, 1, rtprtcp.SubSeq(0, 65535))
	assert.Equal(t, 3, rtprtcp.SubSeq(1, 65534))
	assert.Equal(t, 2, rtprtcp.SubSeq(1, 65535))
}

func TestParseRtpHeader(t *testing.T) {
}
