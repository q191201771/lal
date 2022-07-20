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
	"github.com/q191201771/lal/pkg/httpflv"
)

// RtmpMsg2FlvTag @return 返回的内存块为新申请的独立内存块
func RtmpMsg2FlvTag(msg base.RtmpMsg) *httpflv.Tag {
	var tag httpflv.Tag
	tag.Header.Type = msg.Header.MsgTypeId
	tag.Header.DataSize = msg.Header.MsgLen
	tag.Header.Timestamp = msg.Header.TimestampAbs
	tag.Raw = httpflv.PackHttpflvTag(msg.Header.MsgTypeId, msg.Header.TimestampAbs, msg.Payload)
	return &tag
}

//// ---------------------------------------------------------------------------------------------------------------------
//
//// LazyRtmpMsg2FlvTag 在必要时，有且仅有一次做转换操作
////
//type LazyRtmpMsg2FlvTag struct {
//	msg base.RtmpMsg
//	tag []byte
//}
//
//func (l *LazyRtmpMsg2FlvTag) Init(msg base.RtmpMsg) {
//	l.msg = msg
//}
//
//func (l *LazyRtmpMsg2FlvTag) GetOriginal() []byte {
//	if l.tag == nil {
//		l.tag = RtmpMsg2FlvTag(l.msg).Raw
//	}
//	return l.tag
//}
//
//func (l *LazyRtmpMsg2FlvTag) GetEnsureWithSetDataFrame() []byte {
//	// TODO(chef): [refactor] 这个函数实际上用不上 202207
//	//nazalog.Errorf("LazyRtmpMsg2FlvTag::GetEnsureWithSetDataFrame() is not implemented")
//	return l.GetOriginal()
//}
//
//func (l *LazyRtmpMsg2FlvTag) GetEnsureWithoutSetDataFrame() []byte {
//	if l.tag == nil {
//		b, err := rtmp.MetadataEnsureWithoutSetDataFrame(l.msg.Payload)
//		if err != nil {
//			b = l.msg.Payload
//		}
//		l.msg.Payload = b
//		l.tag = RtmpMsg2FlvTag(l.msg).Raw
//	}
//	return l.tag
//}
