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
	gopNum    int

	Metadata       []byte
	VideoSeqHeader []byte
	AACSeqHeader   []byte

	// TODO 考虑优化成环形队列
	gopList []*GOP
}

func NewGopCache(t string, uniqueKey string, gopNum int) *GOPCache {
	return &GOPCache{
		t:         t,
		uniqueKey: uniqueKey,
		gopNum:    gopNum,
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

	if gc.gopNum != 0 {
		if msg.IsVideoKeyNalu() {
			var gop GOP
			gop.Feed(msg, lg())
			gc.gopList = append(gc.gopList, &gop)
			if len(gc.gopList) > gc.gopNum {
				gc.gopList = gc.gopList[1:]
			}
		} else {
			if len(gc.gopList) > 0 {
				gc.gopList[len(gc.gopList)-1].Feed(msg, lg())
			}
		}
	}
}

func (gc *GOPCache) GetFullGOP() []*GOP {
	if len(gc.gopList) < 2 {
		return nil
	}

	return gc.gopList[:len(gc.gopList)-2]
}

func (gc *GOPCache) GetLastGOP() *GOP {
	if len(gc.gopList) < 1 {
		return nil
	}
	return gc.gopList[len(gc.gopList)-1]
}

func (gc *GOPCache) Clear() {
	gc.Metadata = nil
	gc.VideoSeqHeader = nil
	gc.AACSeqHeader = nil
	gc.gopList = nil
}

type GOP struct {
	data [][]byte
}

func (g *GOP) Feed(msg rtmp.AVMsg, b []byte) {
	g.data = append(g.data, b)
}
