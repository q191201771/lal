// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"bytes"
	"net"
	"testing"

	"github.com/q191201771/naza/pkg/assert"
)

func TestMergeWriter(t *testing.T) {
	goldenBuf1 := bytes.Repeat([]byte{'a'}, 8192)
	goldenBuf2 := bytes.Repeat([]byte{'b'}, 8192)

	var cbBuf net.Buffers
	w := NewMergeWriter(func(bs net.Buffers) {
		cbBuf = bs
	}, 4096)

	// 直接超过
	w.Write(goldenBuf1)
	assert.Equal(t, 1, len(cbBuf))
	assert.Equal(t, goldenBuf1, cbBuf[0])
	cbBuf = nil

	// 不超过
	w.Write(goldenBuf1[:1024])
	assert.Equal(t, nil, cbBuf)

	// 多次不超过
	w.Write(goldenBuf2[:2048])
	assert.Equal(t, nil, cbBuf)

	// 多次超过
	w.Write(goldenBuf1[:2048])
	assert.Equal(t, 3, len(cbBuf))
	assert.Equal(t, goldenBuf1[:1024], cbBuf[0])
	assert.Equal(t, goldenBuf2[:2048], cbBuf[1])
	assert.Equal(t, goldenBuf1[:2048], cbBuf[2])
	cbBuf = nil

	// 不超过，强制刷新
	w.Write(goldenBuf1[:1024])
	assert.Equal(t, nil, cbBuf)
	w.Flush()
	assert.Equal(t, 1, len(cbBuf))
	assert.Equal(t, goldenBuf1[:1024], cbBuf[0])
}
