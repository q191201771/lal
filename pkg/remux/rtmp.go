// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package remux

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtmp"
)

// MakeDefaultRtmpHeader
//
// 使用场景：一般是输入流转换为输出流时。
// 目的：使得流格式更标准。
// 做法：设置 MsgStreamId 和 Csid，其他字段保持`in`的值。
//
func MakeDefaultRtmpHeader(in base.RtmpHeader) (out base.RtmpHeader) {
	out.MsgLen = in.MsgLen
	out.TimestampAbs = in.TimestampAbs
	out.MsgTypeId = in.MsgTypeId
	out.MsgStreamId = rtmp.Msid1
	switch in.MsgTypeId {
	case base.RtmpTypeIdMetadata:
		out.Csid = rtmp.CsidAmf
	case base.RtmpTypeIdAudio:
		out.Csid = rtmp.CsidAudio
	case base.RtmpTypeIdVideo:
		out.Csid = rtmp.CsidVideo
	}
	return
}

//// ---------------------------------------------------------------------------------------------------------------------
//
//// LazyRtmpChunkDivider 在必要时，有且仅有一次做切分成chunk的操作
////
//type LazyRtmpChunkDivider struct {
//	message []byte
//	header  *base.RtmpHeader
//	chunks  []byte
//}
//
//func (lcd *LazyRtmpChunkDivider) Init(message []byte, header *base.RtmpHeader) {
//	lcd.message = message
//	lcd.header = header
//}
//
//func (lcd *LazyRtmpChunkDivider) GetOriginal() []byte {
//	if lcd.chunks == nil {
//		lcd.chunks = rtmp.Message2Chunks(lcd.message, lcd.header)
//	}
//	return lcd.chunks
//}
//
//func (lcd *LazyRtmpChunkDivider) GetEnsureWithSetDataFrame() []byte {
//	if lcd.chunks == nil {
//		var msg []byte
//		var err error
//		if lcd.header.MsgTypeId == base.RtmpTypeIdMetadata {
//			msg, err = rtmp.MetadataEnsureWithSetDataFrame(lcd.message)
//			if err != nil {
//				nazalog.Errorf("[%p] rtmp.MetadataEnsureWithSetDataFrame failed. error=%+v", lcd, err)
//				msg = lcd.message
//			}
//		} else {
//			msg = lcd.message
//		}
//		lcd.chunks = rtmp.Message2Chunks(msg, lcd.header)
//	}
//	return lcd.chunks
//}
//
//func (lcd *LazyRtmpChunkDivider) GetEnsureWithoutSetDataFrame() []byte {
//	if lcd.chunks == nil {
//		var msg []byte
//		var err error
//		if lcd.header.MsgTypeId == base.RtmpTypeIdMetadata {
//			msg, err = rtmp.MetadataEnsureWithoutSetDataFrame(lcd.message)
//			if err != nil {
//				nazalog.Errorf("[%p] rtmp.MetadataEnsureWithoutSetDataFrame failed. error=%+v", lcd, err)
//				msg = lcd.message
//			}
//		} else {
//			msg = lcd.message
//		}
//		lcd.chunks = rtmp.Message2Chunks(msg, lcd.header)
//	}
//	return lcd.chunks
//}
