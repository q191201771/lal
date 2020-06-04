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
	gopRing        []GOP
	gopRingFirst   int
	gopRingLast    int
	gopSize        int
}

func NewGopCache(t string, uniqueKey string, gopNum int) *GOPCache {
	return &GOPCache{
		t:         t,
		uniqueKey: uniqueKey,
		gopSize: gopNum + 1,
		gopRing: make([]GOP, gopNum + 1, gopNum + 1),
		gopRingFirst: 0,
		gopRingLast: 0,
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

	if gc.gopSize > 1 {
		if msg.IsVideoKeyNalu() {
			gc.feedNewGop(msg, lg())
		} else {
			gc.feedLastGop(msg, lg())
		}
	}
}

func (gc *GOPCache) feedLastGop(msg rtmp.AVMsg, b []byte) {
	if gc.gopRingLast != gc.gopRingFirst {
		gc.gopRing[(gc.gopRingLast - 1 + gc.gopSize) % gc.gopSize].Feed(msg, b)
	}
}

func (gc *GOPCache) feedNewGop(msg rtmp.AVMsg, b []byte) {
	gc.gopRing[gc.gopRingLast].Clear()
	gc.gopRing[gc.gopRingLast].Feed(msg, b)
	gc.gopRingLast = (gc.gopRingLast + 1) % gc.gopSize
	if gc.gopRingLast == gc.gopRingFirst {
		gc.gopRingFirst = (gc.gopRingFirst + 1) % gc.gopSize
	}
}

func (gc *GOPCache) isGopEmpty() bool {
	return gc.gopRingFirst == gc.gopRingLast
}

func (gc *GOPCache) GetGopLen() int{
	return (gc.gopRingLast + gc.gopSize - gc.gopRingFirst) % gc.gopSize
}

func (gc *GOPCache) GetGopDataAt(pos int) [][]byte {
	if pos >= gc.GetGopLen() || pos < 0 {
		return nil
	}
	return gc.gopRing[(pos + gc.gopRingFirst) % gc.gopSize].data
}

func (gc *GOPCache) Clear() {
	gc.Metadata = nil
	gc.VideoSeqHeader = nil
	gc.AACSeqHeader = nil
	gc.gopRingLast = 0
	gc.gopRingFirst = 0
}

type GOP struct {
	data [][]byte
}

func (g *GOP) Feed(msg rtmp.AVMsg, b []byte) {
	g.data = append(g.data, b)
}

func (g *GOP) Clear() {
	g.data = g.data[:0]
}
