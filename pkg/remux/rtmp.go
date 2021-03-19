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
	"github.com/cfeeling/lal/pkg/rtmp"
)

func MakeDefaultRTMPHeader(in base.RTMPHeader) (out base.RTMPHeader) {
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
