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

// ---------------------------------------------------------------------------------------------------------------------

// LazyRtmpChunkDivider 在必要时，有且仅有一次做切分成chunk的操作
//
type LazyRtmpChunkDivider struct {
	message []byte
	header  *base.RtmpHeader
	chunks  []byte
}

func (lcd *LazyRtmpChunkDivider) Init(message []byte, header *base.RtmpHeader) {
	lcd.message = message
	lcd.header = header
}

func (lcd *LazyRtmpChunkDivider) Get() []byte {
	if lcd.chunks == nil {
		lcd.chunks = rtmp.Message2Chunks(lcd.message, lcd.header)
	}
	return lcd.chunks
}
