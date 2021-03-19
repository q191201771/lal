// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package remux

import (
	"github.com/cfeeling/lal/pkg/base"
	"github.com/cfeeling/lal/pkg/httpflv"
	"github.com/cfeeling/lal/pkg/rtmp"
)

func FLVTagHeader2RTMPHeader(in httpflv.TagHeader) (out base.RTMPHeader) {
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

// @return 返回的内存块引用参数输入的内存块
func FLVTag2RTMPMsg(tag httpflv.Tag) (msg base.RTMPMsg) {
	msg.Header = FLVTagHeader2RTMPHeader(tag.Header)
	msg.Payload = tag.Raw[11 : 11+msg.Header.MsgLen]
	return
}
