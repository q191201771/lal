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
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
)

var Trans trans

type trans struct {
}

func (t trans) FLVTagHeader2RTMPHeader(in httpflv.TagHeader) (out base.RTMPHeader) {
	out.MsgLen = in.DataSize
	out.MsgTypeID = in.Type
	out.MsgStreamID = rtmp.MSID1
	switch in.Type {
	case httpflv.TagTypeMetadata:
		out.CSID = rtmp.CSIDAMF
	case httpflv.TagTypeAudio:
		out.CSID = rtmp.CSIDAudio
	case httpflv.TagTypeVideo:
		out.CSID = rtmp.CSIDVideo
	}
	out.TimestampAbs = in.Timestamp
	return
}

func (t trans) MakeDefaultRTMPHeader(in base.RTMPHeader) (out base.RTMPHeader) {
	out.MsgLen = in.MsgLen
	out.TimestampAbs = in.TimestampAbs
	out.MsgTypeID = in.MsgTypeID
	out.MsgStreamID = rtmp.MSID1
	switch in.MsgTypeID {
	case base.RTMPTypeIDMetadata:
		out.CSID = rtmp.CSIDAMF
	case base.RTMPTypeIDAudio:
		out.CSID = rtmp.CSIDAudio
	case base.RTMPTypeIDVideo:
		out.CSID = rtmp.CSIDVideo
	}
	return
}

// 音视频内存块不发生拷贝
func (t trans) FLVTag2RTMPMsg(tag httpflv.Tag) (msg base.RTMPMsg) {
	msg.Header = t.FLVTagHeader2RTMPHeader(tag.Header)
	msg.Payload = tag.Raw[11 : 11+msg.Header.MsgLen]
	return
}

// 音视频内存块发生拷贝
func (t trans) RTMPMsg2FLVTag(msg base.RTMPMsg) *httpflv.Tag {
	var tag httpflv.Tag
	tag.Header.Type = msg.Header.MsgTypeID
	tag.Header.DataSize = msg.Header.MsgLen
	tag.Header.Timestamp = msg.Header.TimestampAbs
	tag.Raw = httpflv.PackHTTPFLVTag(msg.Header.MsgTypeID, msg.Header.TimestampAbs, msg.Payload)
	return &tag
}
