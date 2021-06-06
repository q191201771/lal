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
