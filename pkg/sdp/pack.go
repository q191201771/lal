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
	"strings"

	"github.com/q191201771/lal/pkg/aac"

	"github.com/q191201771/lal/pkg/base"
)

func Pack(vps, sps, pps, asc []byte) (ctx LogicContext, err error) {
	// 判断音频、视频是否存在，以及视频是H264还是H265
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

	// 判断AAC的采样率
	var samplingFrequency int
	if asc != nil {
		var ascCtx *aac.AscContext
		ascCtx, err = aac.NewAscContext(asc)
		if err != nil {
			return
		}
		samplingFrequency, err = ascCtx.GetSamplingFrequency()
		if err != nil {
			return
		}
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
			tmpl := `m=video 0 RTP/AVP 98
a=rtpmap:98 H265/90000
a=fmtp:98 profile-id=1;sprop-sps=%s;sprop-pps=%s;sprop-vps=%s
a=control:streamid=%d
`
			sdpStr += fmt.Sprintf(tmpl, base64.StdEncoding.EncodeToString(sps), base64.StdEncoding.EncodeToString(pps), base64.StdEncoding.EncodeToString(vps), streamid)
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
a=rtpmap:97 MPEG4-GENERIC/%d/2
a=fmtp:97 profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3; config=%s
a=control:streamid=%d
`
		sdpStr += fmt.Sprintf(tmpl, samplingFrequency, hex.EncodeToString(asc), streamid)
	}

	raw := []byte(strings.ReplaceAll(sdpStr, "\n", "\r\n"))
	ctx, err = ParseSdp2LogicContext(raw)
	return
}
