// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
)

// 考虑以下两种场景：
// - 只有上行，没有下行，没有必要做rtmp chunk切片的操作
// - 有多个下行，只需要做一次rtmp chunk切片
// 所以这一步做了懒处理
type LazyChunkDivider struct {
	message []byte
	header  *base.RtmpHeader
	chunks  []byte
}

func (lcd *LazyChunkDivider) Init(message []byte, header *base.RtmpHeader) {
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
type LazyRtmpMsg2FlvTag struct {
	msg base.RtmpMsg
	tag []byte
}

func (l *LazyRtmpMsg2FlvTag) Init(msg base.RtmpMsg) {
	l.msg = msg
}

func (l *LazyRtmpMsg2FlvTag) Get() []byte {
	if l.tag == nil {
		l.tag = remux.RtmpMsg2FlvTag(l.msg).Raw
	}
	return l.tag
}

// ---------------------------------------------------------------------------------------------------------------------

// 提供两个功能:
//   1. 缓存Metadata, VideoSeqHeader, AacSeqHeader
//   2. 缓存音视频GOP数据
//
// 以下，只讨论GopCache的第2点功能
//
// 音频和视频都会缓存
//
// GopCache也可能不缓存GOP数据，见NewGopCache函数的gopNum参数说明
//
// 以下，我们只讨论gopNum > 0(也即gopSize > 1)的情况
//
// GopCache为空时，只有输入了关键帧，才能开启GOP缓存，非关键帧以及音频数据不会被缓存
// 因此，单音频的流是ok的，相当于不缓存任何数据
//
// GopCache不为空时，输入关键帧触发生成新的GOP元素，其他情况则往最后一个GOP元素一直追加
//
// first用于读取第一个GOP（可能不完整），last的前一个用于写入当前GOP
//
// 最近不完整的GOP也会被缓存，见NewGopCache函数的gopNum参数说明
//
// -----
// gopNum  = 1
// gopSize = 2
//
//              first     |   first       |       first   | 在后面两个状态间转换，就不画了
//                |       |     |         |        |      |
//                0   1   |     0   1	  |    0   1      |
//                *   *   |     *   *	  |    *   *      |
//                |       |         |	  |    |          |
//              last      |        last   |   last        |
//                        |               |               |
//              (empty)   |   (full)      |   (full)      |
// GetGopCount: 0         |   1           |   1           |
// -----
//
//
type GopCache struct {
	t         string
	uniqueKey string

	Metadata       []byte
	VideoSeqHeader []byte
	AacSeqHeader   []byte

	gopRing      []Gop
	gopRingFirst int
	gopRingLast  int
	gopSize      int
}

// @param gopNum: gop缓存大小
//                如果为0，则不缓存音频数据，也即GOP缓存功能不生效
//                如果>0，则缓存<gopNum>个完整GOP，另外还可能有半个最近不完整的GOP
//
func NewGopCache(t string, uniqueKey string, gopNum int) *GopCache {
	return &GopCache{
		t:            t,
		uniqueKey:    uniqueKey,
		gopSize:      gopNum + 1,
		gopRing:      make([]Gop, gopNum+1, gopNum+1),
		gopRingFirst: 0,
		gopRingLast:  0,
	}
}

type LazyGet func() []byte

func (gc *GopCache) Feed(msg base.RtmpMsg, lg LazyGet) {
	switch msg.Header.MsgTypeId {
	case base.RtmpTypeIdMetadata:
		gc.Metadata = lg()
		nazalog.Debugf("[%s] cache %s metadata. size:%d", gc.uniqueKey, gc.t, len(gc.Metadata))
		return
	case base.RtmpTypeIdAudio:
		if msg.IsAacSeqHeader() {
			gc.AacSeqHeader = lg()
			nazalog.Debugf("[%s] cache %s aac seq header. size:%d", gc.uniqueKey, gc.t, len(gc.AacSeqHeader))
			return
		}
	case base.RtmpTypeIdVideo:
		if msg.IsVideoKeySeqHeader() {
			gc.VideoSeqHeader = lg()
			nazalog.Debugf("[%s] cache %s video seq header. size:%d", gc.uniqueKey, gc.t, len(gc.VideoSeqHeader))
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

// 获取GOP数量，注意，最后一个可能是不完整的
func (gc *GopCache) GetGopCount() int {
	return (gc.gopRingLast + gc.gopSize - gc.gopRingFirst) % gc.gopSize
}

func (gc *GopCache) GetGopDataAt(pos int) [][]byte {
	if pos >= gc.GetGopCount() || pos < 0 {
		return nil
	}
	return gc.gopRing[(pos+gc.gopRingFirst)%gc.gopSize].data
}

func (gc *GopCache) Clear() {
	gc.Metadata = nil
	gc.VideoSeqHeader = nil
	gc.AacSeqHeader = nil
	gc.gopRingLast = 0
	gc.gopRingFirst = 0
}

// 往最后一个GOP元素追加一个msg
// 注意，如果GopCache为空，则不缓存msg
func (gc *GopCache) feedLastGop(msg base.RtmpMsg, b []byte) {
	if !gc.isGopRingEmpty() {
		gc.gopRing[(gc.gopRingLast-1+gc.gopSize)%gc.gopSize].Feed(msg, b)
	}
}

// 生成一个最新的GOP元素，并往里追加一个msg
func (gc *GopCache) feedNewGop(msg base.RtmpMsg, b []byte) {
	if gc.isGopRingFull() {
		gc.gopRingFirst = (gc.gopRingFirst + 1) % gc.gopSize
	}
	gc.gopRing[gc.gopRingLast].Clear()
	gc.gopRing[gc.gopRingLast].Feed(msg, b)
	gc.gopRingLast = (gc.gopRingLast + 1) % gc.gopSize
}

func (gc *GopCache) isGopRingFull() bool {
	return (gc.gopRingLast+1)%gc.gopSize == gc.gopRingFirst
}

func (gc *GopCache) isGopRingEmpty() bool {
	return gc.gopRingFirst == gc.gopRingLast
}

type Gop struct {
	data [][]byte
}

func (g *Gop) Feed(msg base.RtmpMsg, b []byte) {
	g.data = append(g.data, b)
}

func (g *Gop) Clear() {
	g.data = g.data[:0]
}
