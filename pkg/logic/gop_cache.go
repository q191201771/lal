// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import "github.com/q191201771/lal/pkg/rtmp"

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

type GOPCache struct {
	num int

	metadata           []byte
	gopBuf             []byte
	hasAACSeqHeader    bool
	hasAVCKeySeqHeader bool
	hasAVCKeyNalu      bool
}

func NewGopCache(num int) *GOPCache {
	return &GOPCache{
		num: num,
	}
}

func (gc *GOPCache) Feed(msg rtmp.AVMsg, lcd LazyChunkDivider) {
	// CHEFTODO to be continued.
	//switch msg.Header.MsgTypeID {
	//case rtmp.TypeidDataMessageAMF0:
	//	gc.metadata = lcd.Get()
	//case rtmp.TypeidAudio:
	//	if gc.hasAACSeqHeader {
	//		gc.gopBuf = append(gc.gopBuf, lcd.Get()...)
	//	} else {
	//		if msg.IsAACSeqHeader() {
	//			gc.gopBuf = append(gc.gopBuf, lcd.Get()...)
	//			gc.hasAACSeqHeader = true
	//		}
	//	}
	//case rtmp.TypeidVideo:
	//	if gc.hasAVCKeySeqHeader && gc.hasAVCKeyNalu {
	//		gc.gopBuf = append(gc.gopBuf, lcd.Get()...)
	//	}
	//}
}
