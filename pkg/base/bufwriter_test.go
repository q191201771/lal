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
	"testing"

	"github.com/q191201771/naza/pkg/assert"
)

func TestBufWriter(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriterFuncSize(func(p []byte) {
		_, _ = buf.Write(p)
	}, 4096)
	wb, _ := w.(*bufWriter)

	w.Write(bytes.Repeat([]byte{0x1}, 5000))
	assert.Equal(t, 4096-0, wb.available())
	assert.Equal(t, bytes.Repeat([]byte{0x1}, 5000), buf.Bytes())
	buf.Reset()

	w.Write(bytes.Repeat([]byte{0x2}, 1024))
	assert.Equal(t, 4096-1024, wb.available())
	assert.Equal(t, 0, buf.Len())
	w.Write(bytes.Repeat([]byte{0x3}, 1024))
	assert.Equal(t, 4096-2048, wb.available())
	assert.Equal(t, 0, buf.Len())

	w.Write(bytes.Repeat([]byte{0x4}, 4096))
	assert.Equal(t, 4096-2048, wb.available())
	assert.Equal(t, 4096, buf.Len())
	assert.Equal(t, bytes.Repeat([]byte{0x2}, 1024), buf.Bytes()[:1024])
	assert.Equal(t, bytes.Repeat([]byte{0x3}, 1024), buf.Bytes()[1024:2048])
	assert.Equal(t, bytes.Repeat([]byte{0x4}, 2048), buf.Bytes()[2048:])
	buf.Reset()

	w.Write(bytes.Repeat([]byte{0x5}, 8192))
	assert.Equal(t, 4096-0, wb.available())
	assert.Equal(t, 2048+8192, buf.Len())
	assert.Equal(t, bytes.Repeat([]byte{0x4}, 2048), buf.Bytes()[:2048])
	assert.Equal(t, bytes.Repeat([]byte{0x5}, 8192), buf.Bytes()[2048:])
	buf.Reset()

	w.Flush()
	assert.Equal(t, 4096-0, wb.available())
	assert.Equal(t, 0, buf.Len())

	w.Write(bytes.Repeat([]byte{0x6}, 1024))
	assert.Equal(t, 4096-1024, wb.available())
	assert.Equal(t, 0, buf.Len())
	w.Flush()
	assert.Equal(t, 4096-0, wb.available())
	assert.Equal(t, 1024, buf.Len())
	assert.Equal(t, bytes.Repeat([]byte{0x6}, 1024), buf.Bytes()[:1024])
}
