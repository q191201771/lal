// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"fmt"
	"github.com/q191201771/naza/pkg/nazalog"
	"io"
)

// TODO(chef): refactor 移入naza中
// TODO(chef): 增加options: growRoundThreshold; 是否做检查
// TODO(chef): 扩容策略函数可由外部传入

const growRoundThreshold = 1048576 // 1MB

// Buffer 先进先出可扩容流式buffer，可直接读写内部切片避免拷贝
//
// 示例
//   读取方式1
//     buf := Bytes()
//     ... // 读取buf的内容
//     Skip(n)
//
//   读取方式2
//     buf := Peek(n)
//     ...
//
//   读取方式3
//     buf := make([]byte, n)
//     nn, err := Read(buf)
//
//   写入方式1
//     Grow(n)
//     buf := WritableBytes()[:n]
//     ... // 向buf中写入内容
//     Flush(n)
//
//   写入方式2
//     buf := ReserveBytes(n)
//     ... // 向buf中写入内容
//     Flush(n)
//
//   写入方式3
//     n, err := Write(buf)
//
type Buffer struct {
	core []byte
	rpos int
	wpos int
}

func NewBuffer(initCap int) *Buffer {
	return &Buffer{
		core: make([]byte, initCap, initCap),
	}
}

// NewBufferRefBytes
//
// 注意，不拷贝参数`b`的内存块，仅持有
//
func NewBufferRefBytes(b []byte) *Buffer {
	return &Buffer{
		core: b,
	}
}

// ---------------------------------------------------------------------------------------------------------------------

// Bytes Buffer中所有未读数据，类似于PeekAll，不拷贝
//
func (b *Buffer) Bytes() []byte {
	if b.rpos == b.wpos {
		return nil
	}
	return b.core[b.rpos:b.wpos]
}

// Peek 查看指定长度的未读数据，不拷贝，类似于Next，但是不会修改读取偏移位置
//
func (b *Buffer) Peek(n int) []byte {
	if b.rpos == b.wpos {
		return nil
	}
	if b.Len() < n {
		return b.Bytes()
	}
	return b.core[b.rpos : b.rpos+n]
}

// Skip 将前`n`未读数据标记为已读（也即消费完成）
//
func (b *Buffer) Skip(n int) {
	if n > b.wpos-b.rpos {
		nazalog.Warnf("[%p] Buffer::Skip too large. n=%d, %s", b, n, b.DebugString())
		b.Reset()
		return
	}
	b.rpos += n
	b.resetIfEmpty()
}

// ---------------------------------------------------------------------------------------------------------------------

// Grow 确保Buffer中至少有`n`大小的空间可写，类似于Reserve
//
func (b *Buffer) Grow(n int) {
	//nazalog.Debugf("[%p] > Buffer::Grow. n=%d, %s", b, n, b.DebugString())
	tail := len(b.core) - b.wpos
	if tail >= n {
		// 尾部空闲空间足够
		return
	}

	if b.rpos+tail >= n {
		// 头部加上尾部空闲空间足够，将可读数据移动到头部，回收头部空闲空间
		nazalog.Debugf("[%p] Buffer::Grow. move, n=%d, copy=%d", b, n, b.Len())
		copy(b.core, b.core[b.rpos:b.wpos])
		b.wpos -= b.rpos
		b.rpos = 0
		return
	}

	// 扩容后总共需要的大小
	needed := b.Len() + n

	// 扩容大小在阈值范围内时，向上取值到2的倍数
	if needed < growRoundThreshold {
		needed = roundUpPowerOfTwo(needed)
	}

	nazalog.Debugf("[%p] Buffer::Grow. realloc, n=%d, copy=%d, cap=(%d, %d)", b, n, b.Len(), b.Cap(), needed)
	core := make([]byte, needed, needed)
	copy(core, b.core[b.rpos:b.wpos])
	b.core = core
	b.rpos = 0
	b.wpos -= b.rpos
}

// WritableBytes 返回当前可写入的字节切片
//
func (b *Buffer) WritableBytes() []byte {
	if len(b.core) == b.wpos {
		return nil
	}
	return b.core[b.wpos:]
}

// ReserveBytes 返回可写入`n`大小的字节切片，如果空闲空间不够，内部会进行扩容
//
// 注意，返回值空间大小只会为`n`，
//
func (b *Buffer) ReserveBytes(n int) []byte {
	b.Grow(n)
	return b.WritableBytes()[:n]
}

// Flush 写入完成，更新写入位置
//
func (b *Buffer) Flush(n int) {
	if len(b.core)-b.wpos < n {
		nazalog.Warnf("[%p] Buffer::Flush too large. n=%d, %s", b, n, b.DebugString())
		b.wpos = len(b.core)
		return
	}
	b.wpos += n
}

// ----- implement io.Reader interface ---------------------------------------------------------------------------------

// Read 拷贝，`p`空间由外部申请
//
func (b *Buffer) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if b.Len() == 0 {
		return 0, io.EOF
	}
	n = copy(p, b.core[b.rpos:b.wpos])
	b.Skip(n)
	return n, nil
}

// ----- implement io.Writer interface ---------------------------------------------------------------------------------

// Write 拷贝
//
func (b *Buffer) Write(p []byte) (n int, err error) {
	b.Grow(len(p))
	copy(b.core[b.wpos:], p)
	b.wpos += n
	return len(p), nil
}

// ---------------------------------------------------------------------------------------------------------------------

// Truncate 丢弃可读数据的末尾`n`大小的数据，或者理解为取消写
//
func (b *Buffer) Truncate(n int) {
	if b.Len() < n {
		nazalog.Warnf("[%p] Buffer::Truncate too large. n=%d, %s", b, n, b.DebugString())
		b.Reset()
		return
	}
	b.wpos -= n
	b.resetIfEmpty()
}

// Reset 重置
//
// 注意，并不会释放内存块
//
func (b *Buffer) Reset() {
	b.rpos = 0
	b.wpos = 0
}

// ---------------------------------------------------------------------------------------------------------------------

// Len Buffer中还没有读的数据的长度
//
func (b *Buffer) Len() int {
	return b.wpos - b.rpos
}

// Cap 整个Buffer占用的空间
//
func (b *Buffer) Cap() int {
	return cap(b.core)
}

// ---------------------------------------------------------------------------------------------------------------------

func (b *Buffer) DebugString() string {
	return fmt.Sprintf("len(core)=%d, rpos=%d, wpos=%d", len(b.core), b.rpos, b.wpos)
}

// ---------------------------------------------------------------------------------------------------------------------

func (b *Buffer) resetIfEmpty() {
	if b.rpos == b.wpos {
		b.Reset()
	}
}

// TODO(chef): refactor 移入naza中
func roundUpPowerOfTwo(n int) int {
	if n <= 2 {
		return 2
	}

	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n |= n >> 32
	n++
	return n
}
