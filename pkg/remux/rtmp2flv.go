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
)

// @return 返回的内存块为新申请的独立内存块
func RTMPMsg2FLVTag(msg base.RTMPMsg) *httpflv.Tag {
	var tag httpflv.Tag
	tag.Header.Type = msg.Header.MsgTypeID
	tag.Header.DataSize = msg.Header.MsgLen
	tag.Header.Timestamp = msg.Header.TimestampAbs
	tag.Raw = httpflv.PackHTTPFLVTag(msg.Header.MsgTypeID, msg.Header.TimestampAbs, msg.Payload)
	return &tag
}
