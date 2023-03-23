// Copyright 2023, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package sdp

import (
	"testing"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/assert"
)

var avcsps = []byte{
	0x67, 0x64, 0x00, 0x20, 0xac, 0xd9, 0x40, 0xc0, 0x29, 0xb0, 0x11, 0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x00, 0x03, 0x00, 0x32, 0x0f, 0x18, 0x31, 0x96,
}

var avcpps = []byte{
	0x68, 0xeb, 0xec, 0xb2, 0x2c,
}

var hevcsps = []byte{
	0x42, 0x01, 0x01, 0x01, 0x60, 0x00, 0x00, 0x03, 0x00, 0x90, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x50, 0x50, 0x40, 0x00, 0x00, 0x03, 0x00, 0x40, 0x00, 0x00, 0x07, 0x82,
}

var hevcpps = []byte{
	0x44, 0x01, 0xc1, 0x72, 0xb0, 0x62, 0x40,
}

var hevcvps = []byte{
	0x40, 0x01, 0x0c, 0x01, 0xff, 0xff, 0x01, 0x60, 0x00, 0x00, 0x03, 0x00, 0x90, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x78, 0x91, 0x14, 0x09,
}

var asc = []byte{
	0x12, 0x10,
}

func TestSdpPack(t *testing.T) {
	{
		// avc和aac
		video := VideoInfo{
			VideoPt: base.AvPacketPtAvc,
			Sps:     avcsps,
			Pps:     avcpps,
		}

		audio := AudioInfo{
			AudioPt:           base.AvPacketPtAac,
			SamplingFrequency: 44100,
			Asc:               asc,
		}

		sdpctx, err := Pack(video, audio)
		assert.Equal(t, nil, err)
		assert.Equal(t, avcsps, sdpctx.Sps)
		assert.Equal(t, avcpps, sdpctx.Pps)
		assert.Equal(t, asc, sdpctx.Asc)
	}
	{
		// video和audio无效
		video := VideoInfo{
			VideoPt: base.AvPacketPtUnknown,
		}
		audio := AudioInfo{
			AudioPt: base.AvPacketPtUnknown,
		}
		_, err := Pack(video, audio)
		assert.IsNotNil(t, err)
	}
	{
		// 只有video
		video := VideoInfo{
			VideoPt: base.AvPacketPtAvc,
			Sps:     avcsps,
			Pps:     avcpps,
		}
		audio := AudioInfo{
			AudioPt: base.AvPacketPtUnknown,
		}
		sdpctx, err := Pack(video, audio)
		assert.Equal(t, nil, err)
		assert.Equal(t, avcsps, sdpctx.Sps)
		assert.Equal(t, avcpps, sdpctx.Pps)
		assert.Equal(t, nil, sdpctx.Asc)
	}
	{
		// 只有audio
		video := VideoInfo{
			VideoPt: base.AvPacketPtUnknown,
		}
		audio := AudioInfo{
			AudioPt:           base.AvPacketPtAac,
			SamplingFrequency: 44100,
			Asc:               asc,
		}
		sdpctx, err := Pack(video, audio)
		assert.Equal(t, nil, err)
		assert.Equal(t, nil, sdpctx.Sps)
		assert.Equal(t, nil, sdpctx.Pps)
		assert.Equal(t, asc, sdpctx.Asc)
	}
	{
		// g711a
		video := VideoInfo{
			VideoPt: base.AvPacketPtUnknown,
		}
		audio := AudioInfo{
			AudioPt:           base.AvPacketPtG711A,
			SamplingFrequency: 44100,
		}
		sdpctx, err := Pack(video, audio)
		assert.Equal(t, nil, err)
		assert.Equal(t, nil, sdpctx.Sps)
		assert.Equal(t, nil, sdpctx.Pps)
		assert.Equal(t, nil, sdpctx.Asc)
		assert.Equal(t, base.AvPacketPtG711A, sdpctx.audioPayloadTypeBase)
		assert.Equal(t, 44100, sdpctx.AudioClockRate)
	}
	{
		// g711u
		video := VideoInfo{
			VideoPt: base.AvPacketPtUnknown,
		}
		audio := AudioInfo{
			AudioPt:           base.AvPacketPtG711U,
			SamplingFrequency: 44100,
		}
		sdpctx, err := Pack(video, audio)
		assert.Equal(t, nil, err)
		assert.Equal(t, nil, sdpctx.Sps)
		assert.Equal(t, nil, sdpctx.Pps)
		assert.Equal(t, nil, sdpctx.Asc)
		assert.Equal(t, base.AvPacketPtG711U, sdpctx.audioPayloadTypeBase)
		assert.Equal(t, 44100, sdpctx.AudioClockRate)
	}
	{
		// hevc和aac
		video := VideoInfo{
			VideoPt: base.AvPacketPtHevc,
			Sps:     hevcsps,
			Pps:     hevcpps,
			Vps:     hevcvps,
		}

		audio := AudioInfo{
			AudioPt:           base.AvPacketPtAac,
			SamplingFrequency: 44100,
			Asc:               asc,
		}

		sdpctx, err := Pack(video, audio)
		assert.Equal(t, nil, err)
		assert.Equal(t, hevcsps, sdpctx.Sps)
		assert.Equal(t, hevcpps, sdpctx.Pps)
		assert.Equal(t, hevcvps, sdpctx.Vps)
		assert.Equal(t, asc, sdpctx.Asc)
	}
}
