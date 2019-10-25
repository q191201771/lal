// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
)

var Trans trans

type trans struct {
}

// 注意，tag -> message [nocopy]
func (t trans) FLVTag2RTMPMsg(tag httpflv.Tag) (header rtmp.Header, timestampAbs uint32, message []byte) {
	header.MsgLen = tag.Header.DataSize
	header.MsgTypeID = tag.Header.T
	header.MsgStreamID = rtmp.MSID1
	switch tag.Header.T {
	case httpflv.TagTypeMetadata:
		header.CSID = rtmp.CSIDAMF
	case httpflv.TagTypeAudio:
		header.CSID = rtmp.CSIDAudio
	case httpflv.TagTypeVideo:
		header.CSID = rtmp.CSIDVideo
	}
	header.Timestamp = tag.Header.Timestamp
	timestampAbs = tag.Header.Timestamp
	message = tag.Raw[11 : 11+header.MsgLen]
	return
}

// 注意，message -> tag [copy]
func (t trans) RTMPMsg2FLVTag(header rtmp.Header, timestampAbs uint32, message []byte) *httpflv.Tag {
	var tag httpflv.Tag
	tag.Header.T = header.MsgTypeID
	tag.Header.DataSize = header.MsgLen
	tag.Header.Timestamp = timestampAbs
	tag.Raw = httpflv.PackHTTPFLVTag(header.MsgTypeID, timestampAbs, message)
	return &tag
}
