// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import (
	"encoding/hex"
	"fmt"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/nazalog"
)

const initMsgLen = 4096

// TODO chef: 将这个buffer实现和bytes.Buffer做比较，考虑将它放入naza package中
type StreamMsg struct {
	buf []byte
	b   uint32 // 读取起始位置
	e   uint32 // 读取结束位置，写入起始位置
}

type Stream struct {
	header base.RtmpHeader
	msg    StreamMsg

	timestamp uint32 // 注意，是rtmp chunk协议header中的时间戳，可能是绝对的，也可能是相对的。上层不应该使用这个字段，而应该使用Header.TimestampAbs
}

func NewStream() *Stream {
	return &Stream{
		msg: StreamMsg{
			buf: make([]byte, initMsgLen),
		},
	}
}

// 序列化成可读字符串，一般用于发生错误时打印日志
func (stream *Stream) toDebugString() string {
	// 注意，这里打印的二进制数据的其实位置是从 0 开始，而不是 msg.b 位置
	return fmt.Sprintf("header=%+v, b=%d, hex=%s",
		stream.header, stream.msg.b, hex.Dump(stream.msg.buf[:stream.msg.e]))
}

func (stream *Stream) toAvMsg() base.RtmpMsg {
	// TODO chef: 考虑可能出现header中的len和buf的大小不一致的情况
	if stream.header.MsgLen != uint32(len(stream.msg.buf[stream.msg.b:stream.msg.e])) {
		nazalog.Errorf("toAvMsg. headerMsgLen=%d, bufLen=%d", stream.header.MsgLen, len(stream.msg.buf[stream.msg.b:stream.msg.e]))
	}
	return base.RtmpMsg{
		Header:  stream.header,
		Payload: stream.msg.buf[stream.msg.b:stream.msg.e],
	}
}

// 确保可写空间，如果不够会扩容
func (msg *StreamMsg) reserve(n uint32) {
	bufCap := uint32(cap(msg.buf))
	nn := bufCap - msg.e // 剩余空闲空间
	if nn > n {          // 足够
		return
	}
	for nn < n { // 不够，空闲空间翻倍，直到大于需求空间
		nn <<= 1
	}
	nb := make([]byte, bufCap+nn)  // 当前容量加扩充容量
	copy(nb, msg.buf[msg.b:msg.e]) // 老数据拷贝
	msg.buf = nb                   // 替换
	nazalog.Debugf("reserve. newLen=%d(%d, %d), need=(%d -> %d), cap=(%d -> %d)", len(msg.buf), msg.b, msg.e, n, nn, bufCap, cap(msg.buf))
}

// 可读长度
func (msg *StreamMsg) len() uint32 {
	return msg.e - msg.b
}

// 写入数据后调用
func (msg *StreamMsg) produced(n uint32) {
	msg.e += n
}

// 读取数据后调用
func (msg *StreamMsg) consumed(n uint32) {
	msg.b += n
}

// 清空，空闲内存空间保留不释放
func (msg *StreamMsg) clear() {
	msg.b = 0
	msg.e = 0
}

//func (msg *StreamMsg) bytes() []byte {
//	return msg.buf[msg.b: msg.e]
//}

func (msg *StreamMsg) peekStringWithType() (string, error) {
	str, _, err := Amf0.ReadString(msg.buf[msg.b:msg.e])
	return str, err
}

func (msg *StreamMsg) readStringWithType() (string, error) {
	str, l, err := Amf0.ReadString(msg.buf[msg.b:msg.e])
	if err == nil {
		msg.consumed(uint32(l))
	}
	return str, err
}

func (msg *StreamMsg) readNumberWithType() (int, error) {
	val, l, err := Amf0.ReadNumber(msg.buf[msg.b:msg.e])
	if err == nil {
		msg.consumed(uint32(l))
	}
	return int(val), err
}

func (msg *StreamMsg) readObjectWithType() (ObjectPairArray, error) {
	opa, l, err := Amf0.ReadObject(msg.buf[msg.b:msg.e])
	if err == nil {
		msg.consumed(uint32(l))
	}
	return opa, err
}

func (msg *StreamMsg) readNull() error {
	l, err := Amf0.ReadNull(msg.buf[msg.b:msg.e])
	if err == nil {
		msg.consumed(uint32(l))
	}
	return err
}
