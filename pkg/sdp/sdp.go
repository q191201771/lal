// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package sdp

import "github.com/q191201771/lal/pkg/base"

// rfc4566

const (
	ARtpMapEncodingNameH265  = "H265"
	ARtpMapEncodingNameH264  = "H264"
	ARtpMapEncodingNameAac   = "MPEG4-GENERIC"
	ARtpMapEncodingNameG711A = "PCMA"
	ARtpMapEncodingNameG711U = "PCMU"
	ArtpMapEncodingNameOpus  = "opus"
)

const (
	MediaDescPayloadTypeG711U = int(base.AvPacketPtG711U)
	MediaDescPayloadTypeG711A = int(base.AvPacketPtG711A)
	MediaDescPayloadTypeMp2   = int(base.AvPacketPtMp2)
)
