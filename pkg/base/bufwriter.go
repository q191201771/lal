// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

// 调用方场景/约定描述：
// - 内部实际写总是全量写成功
// - 外部写调用的内存块，永久有效，调用结束后，外部并不使用
//   内部实际写的内存块，调用结束后，实际写的实现可能还继续持有该内存块
//
// 与bufio.Writer表现不同的地方：
// 数据超过缓存容量时，bufio.Writer一般会切分成多个缓存容量大小的块实际写，BufWriter则可能实际写大数据

type IBufWriter interface {
	Write(p []byte)
	Flush()
}

type WriterFunc func(p []byte)

func NewWriterFuncSize(wr WriterFunc, size int) IBufWriter {
	if size <= 0 {
		return &directWriter{
			wr: wr,
		}
	}

	b := &bufWriter{
		wr:          wr,
		defaultSize: size,
	}
	b.mallocOnePiece(size)
	return b
}

type bufWriter struct {
	wr          WriterFunc
	defaultSize int // 缓存最大容量
	buf         []byte
	n           int // 当前已缓存大小
}

type directWriter struct {
	wr WriterFunc
}

func (b *bufWriter) Write(p []byte) {
	avail := b.available()
	if len(p) > avail {
		// 缓存剩余空间不够

		if b.n == 0 {
			// 缓存完全没有使用，依然空间不够
			// 直接发送
			b.wr(p)
			return
		}

		// 填满当前缓存块，并发送
		b.append(p[:avail])
		b.Flush()

		// 剩余数据
		remain := len(p) - avail
		if remain < b.defaultSize {
			// 剩余数据较小
			// 只追加进入缓存
			b.append(p[avail:])
		} else {
			// 剩余数据较大
			// 直接发送
			b.wr(p[avail:])
		}
		return
	}

	// 缓存剩余空间足够
	// 追加进入缓存
	b.append(p)
}

func (b *bufWriter) Flush() {
	if b.n == 0 {
		return
	}
	b.wr(b.buf[:b.n])
	b.mallocOnePiece(b.defaultSize)
}

// 缓存剩余空间
func (b *bufWriter) available() int {
	return len(b.buf) - b.n
}

func (b *bufWriter) mallocOnePiece(size int) {
	b.buf = make([]byte, size)
	b.n = 0
}

func (b *bufWriter) append(p []byte) {
	copy(b.buf[b.n:], p)
	b.n += len(p)
}

func (dw *directWriter) Write(p []byte) {
	dw.wr(p)
}

func (dw *directWriter) Flush() {
	// noop
}
