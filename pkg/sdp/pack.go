// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package sdp

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/q191201771/lal/pkg/base"
)

func Pack(vps, sps, pps, asc []byte) (ctx LogicContext, raw []byte, err error) {
	var hasAudio, hasVideo, isHevc bool

	if sps != nil && pps != nil {
		hasVideo = true
		if vps != nil {
			isHevc = true
		}
	}
	if asc != nil {
		hasAudio = true
	}

	if !hasAudio && !hasVideo {
		err = ErrSdp
		return
	}

	sdpStr := fmt.Sprintf(`v=0
	o=- 0 0 IN IP4 127.0.0.1
	s=No Name
	c=IN IP4 127.0.0.1
	t=0 0
	a=tool:%s
`, base.LalPackSdp)

	streamid := 0

	if hasVideo {
		if isHevc {

		} else {
			tmpl := `m=video 0 RTP/AVP 96
	a=rtpmap:96 H264/90000
	a=fmtp:96 packetization-mode=1; sprop-parameter-sets=%s,%s; profile-level-id=640016
	a=control:streamid=%d
`
			sdpStr += fmt.Sprintf(tmpl, base64.StdEncoding.EncodeToString(sps), base64.StdEncoding.EncodeToString(pps), streamid)
		}

		streamid++
	}

	if hasAudio {
		tmpl := `m=audio 0 RTP/AVP 97
	b=AS:128
	a=rtpmap:97 MPEG4-GENERIC/44100/2
	a=fmtp:97 profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3; config=%s
	a=control:streamid=%d
`
		sdpStr += fmt.Sprintf(tmpl, hex.EncodeToString(asc), streamid)
	}

	raw = []byte(sdpStr)
	ctx, err = ParseSdp2LogicContext(raw)
	return
}
