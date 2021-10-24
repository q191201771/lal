// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"github.com/q191201771/naza/pkg/assert"
	"github.com/q191201771/naza/pkg/nazalog"
	"testing"
)

func TestBuffer(t *testing.T) {
	golden := []byte("1234567890")

	b := NewBuffer(8)
	assert.Equal(t, nil, b.Bytes())
	assert.Equal(t, 8, len(b.WritableBytes()))
	assert.Equal(t, 0, b.Len())
	assert.Equal(t, 8, b.Cap())

	// 简单写读
	b.Grow(5)
	buf := b.WritableBytes()[:5]
	assert.Equal(t, nil, b.Bytes())
	copy(buf, golden[:5])
	b.Flush(5)
	buf = b.Bytes()
	assert.Equal(t, golden[:5], buf)
	assert.Equal(t, 5, b.Len())
	b.Skip(5)
	assert.Equal(t, nil, b.Bytes())
	assert.Equal(t, 8, len(b.WritableBytes()))
	assert.Equal(t, 0, b.Len())
	assert.Equal(t, 8, b.Cap())

	// 发生扩容
	buf = b.ReserveBytes(10)
	copy(buf, golden)
	b.Flush(10)
	buf = b.Bytes()
	assert.Equal(t, golden, buf)
	b.Skip(10)
	assert.Equal(t, nil, b.Bytes())
	assert.Equal(t, 16, len(b.WritableBytes()))
	assert.Equal(t, 0, b.Len())
	assert.Equal(t, 16, b.Cap())

	// 利用头部空闲空间扩容
	buf = b.ReserveBytes(10)
	copy(buf, golden)
	b.Flush(10)
	b.Skip(2)
	buf = b.ReserveBytes(7)
	copy(buf, golden[:7])
	b.Flush(7)
	nazalog.Debugf("%s", string(b.Bytes()))
	assert.Equal(t, golden[2:], b.Bytes()[:8])
	assert.Equal(t, golden[:7], b.Bytes()[8:])
	assert.Equal(t, 15, b.Len())
	assert.Equal(t, 16, b.Cap())

	// Truncate
	b.Reset()
	buf = b.ReserveBytes(10)
	copy(buf, golden)
	b.Flush(10)
	b.Truncate(4)
	assert.Equal(t, golden[:6], b.Bytes())

	// 特殊值
	b.Reset()
	b.Flush(b.Cap())
	assert.Equal(t, nil, b.WritableBytes())

	// 一些错误
	b.Reset()
	b.Skip(1)
	assert.Equal(t, nil, b.Bytes())
	b.Truncate(1)
	assert.Equal(t, nil, b.Bytes())
	b.Flush(b.Cap() + 1)
	assert.Equal(t, b.Cap(), b.Len())

	// 特殊值，极小的扩容
	b = NewBuffer(1)
	buf = b.ReserveBytes(2)
}
