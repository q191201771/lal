// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
)

// 考虑以下两种场景：
// - 只有上行，没有下行，没有必要做rtmp chunk切片的操作
// - 有多个下行，只需要做一次rtmp chunk切片
// 所以这一步做了懒处理
type LazyChunkDivider struct {
	message []byte
	header  *rtmp.Header
	chunks []byte
}

func (lcd *LazyChunkDivider) Init(message []byte, header *rtmp.Header) {
	lcd.message = message
	lcd.header = header
}

func (lcd *LazyChunkDivider) Get() []byte {
	if lcd.chunks == nil {
		lcd.chunks = rtmp.Message2Chunks(lcd.message, lcd.header)
	}
	return lcd.chunks
}

// 懒转换
type LazyRTMPMsg2FLVTag struct {
	msg rtmp.AVMsg
	tag []byte
}

func (l *LazyRTMPMsg2FLVTag) Init(msg rtmp.AVMsg) {
	l.msg = msg
}

func (l *LazyRTMPMsg2FLVTag) Get() []byte {
	if l.tag == nil {
		l.tag = Trans.RTMPMsg2FLVTag(l.msg).Raw
	}
	return l.tag
}

type GOPCache struct {
	t         string
	uniqueKey string
	Metadata       []byte
	VideoSeqHeader []byte
	AACSeqHeader   []byte
	gopCircularQue *gopCircularQueue
}

func NewGopCache(t string, uniqueKey string, gopNum int) *GOPCache {
	return &GOPCache{
		t:         t,
		uniqueKey: uniqueKey,
		gopCircularQue: NewGopCircularQueue(gopNum),
	}
}

type LazyGet func() []byte

func (gc *GOPCache) Feed(msg rtmp.AVMsg, lg LazyGet) {
	switch msg.Header.MsgTypeID {
	case rtmp.TypeidDataMessageAMF0:
		gc.Metadata = lg()
		nazalog.Debugf("cache %s metadata. [%s] size:%d", gc.t, gc.uniqueKey, len(gc.Metadata))
		return
	case rtmp.TypeidAudio:
		if msg.IsAACSeqHeader() {
			gc.AACSeqHeader = lg()
			nazalog.Debugf("cache %s aac seq header. [%s] size:%d", gc.t, gc.uniqueKey, len(gc.AACSeqHeader))
			return
		}
	case rtmp.TypeidVideo:
		if msg.IsVideoKeySeqHeader() {
			gc.VideoSeqHeader = lg()
			nazalog.Debugf("cache %s video seq header. [%s] size:%d", gc.t, gc.uniqueKey, len(gc.VideoSeqHeader))
			return
		}
	}
	if gc.gopCircularQue.Cap() != 0 {
		if msg.IsVideoKeyNalu() {
			var gop GOP
			gop.Feed(msg, lg())
			gc.gopCircularQue.Enqueue(&gop)
		} else {
			gop := gc.gopCircularQue.Back()
			if gop != nil {
				gop.Feed(msg, lg())
			}
		}
	}
}

func (gc *GOPCache) GetGopLen() int{
	return gc.gopCircularQue.Len()
}

func (gc *GOPCache) GetGopAt(pos int) *GOP {
	return gc.gopCircularQue.At(pos)
}

func (gc *GOPCache) LastGOP() *GOP {
	return gc.gopCircularQue.Back()
}

func (gc *GOPCache) Clear() {
	gc.Metadata = nil
	gc.VideoSeqHeader = nil
	gc.AACSeqHeader = nil
	gc.gopCircularQue.Clear()
}

type GOP struct {
	data [][]byte
}

func (g *GOP) Feed(msg rtmp.AVMsg, b []byte) {
	g.data = append(g.data, b)
}

type gopCircularQueue struct {
	buf []*GOP
	size int
	first int
	last int
}

func NewGopCircularQueue(cap int) *gopCircularQueue {
	
	if cap < 0 {
		panic("negative cap argument in NewGopCircularQueue")
	}
	
	size := cap + 1
	return &gopCircularQueue{
		buf: make([]*GOP, size, size),
		size: size,
		first: 0,
		last: 0,
	}
}

//Enqueue 入队
func (gcq *gopCircularQueue) Enqueue(gop *GOP) {
	if gcq.Full() {
		//队列满了，抛弃队首元素
		gcq.firstInc()
	}
	gcq.buf[gcq.last] = gop
	gcq.lastInc()
}

//Dequeue 队首元素出队
func (gcq *gopCircularQueue) Dequeue() *GOP{
	if gcq.Empty() {
		return nil
	}
	
	//把first return
	gop := gcq.buf[gcq.first]
	gcq.firstInc()
	
	return gop
}

func (gcq *gopCircularQueue) Full() bool {
	return (gcq.last + 1) % gcq.size == gcq.first
}

func (gcq *gopCircularQueue) Empty() bool{
	return gcq.first == gcq.last
}

//Len 获取元素的个数
func (gcq *gopCircularQueue) Len() int{
	return (gcq.last + gcq.size -  gcq.first) % gcq.size
}

//Cap 获取队列的容量
func (gcq *gopCircularQueue) Cap() int {
	return gcq.size - 1
}

//Front 获取队首元素，不出队
func (gcq *gopCircularQueue) Front() *GOP {
	if gcq.Empty() {
		return nil
	}
	return gcq.buf[gcq.first]
}

//Back 获取队尾元素，不出队
func (gcq *gopCircularQueue) Back() *GOP {
	if gcq.Empty() {
		return nil
	}
	return gcq.buf[(gcq.last + gcq.size - 1) % gcq.size]
}

//At 获取第pos个元素
func (gcq *gopCircularQueue) At(pos int) *GOP {
	if pos >= gcq.Len() || pos < 0 {
		return nil
	}
	return gcq.buf[(pos + gcq.first) % gcq.size]
}

func (gcq *gopCircularQueue) Clear() {
	gcq.first = 0
	gcq.last = 0

}

func (gcq *gopCircularQueue) lastInc() {
	gcq.last = (gcq.last + 1) % gcq.size
}

func (gcq *gopCircularQueue) firstInc() {
	gcq.first = (gcq.first + 1) % gcq.size
}
