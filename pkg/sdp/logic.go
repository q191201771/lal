// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package sdp

import (
	"fmt"
	"strings"

	"github.com/cfeeling/naza/pkg/nazalog"

	"github.com/cfeeling/lal/pkg/base"
)

type LogicContext struct {
	AudioClockRate int
	VideoClockRate int

	audioPayloadTypeBase base.AVPacketPT // lal内部定义的类型
	videoPayloadTypeBase base.AVPacketPT

	audioPayloadTypeOrigin int // 原始类型，sdp或rtp中的类型
	videoPayloadTypeOrigin int
	audioAControl          string
	videoAControl          string

	ASC []byte
	VPS []byte
	SPS []byte
	PPS []byte

	// 没有用上的
	hasAudio bool
	hasVideo bool
}

func (lc *LogicContext) IsAudioPayloadTypeOrigin(t int) bool {
	return lc.audioPayloadTypeOrigin == t
}

func (lc *LogicContext) IsVideoPayloadTypeOrigin(t int) bool {
	return lc.videoPayloadTypeOrigin == t
}

func (lc *LogicContext) IsPayloadTypeOrigin(t int) bool {
	return lc.audioPayloadTypeOrigin == t || lc.videoPayloadTypeOrigin == t
}

func (lc *LogicContext) IsAudioUnpackable() bool {
	return lc.audioPayloadTypeBase == base.AVPacketPTAAC
}

func (lc *LogicContext) IsVideoUnpackable() bool {
	return lc.videoPayloadTypeBase == base.AVPacketPTAVC ||
		lc.videoPayloadTypeBase == base.AVPacketPTHEVC
}

func (lc *LogicContext) IsAudioURI(uri string) bool {
	return lc.audioAControl != "" && strings.HasSuffix(uri, lc.audioAControl)
}

func (lc *LogicContext) IsVideoURI(uri string) bool {
	return lc.videoAControl != "" && strings.HasSuffix(uri, lc.videoAControl)
}

func (lc *LogicContext) HasAudioAControl() bool {
	return lc.audioAControl != ""
}

func (lc *LogicContext) HasVideoAControl() bool {
	return lc.videoAControl != ""
}

func (lc *LogicContext) MakeAudioSetupURI(uri string) string {
	return lc.makeSetupURI(uri, lc.audioAControl)
}

func (lc *LogicContext) MakeVideoSetupURI(uri string) string {
	return lc.makeSetupURI(uri, lc.videoAControl)
}

func (lc *LogicContext) GetAudioPayloadTypeBase() base.AVPacketPT {
	return lc.audioPayloadTypeBase
}

func (lc *LogicContext) GetVideoPayloadTypeBase() base.AVPacketPT {
	return lc.videoPayloadTypeBase
}

func (lc *LogicContext) makeSetupURI(uri string, aControl string) string {
	if strings.HasPrefix(aControl, "rtsp://") {
		return aControl
	}
	return fmt.Sprintf("%s/%s", uri, aControl)
}

func ParseSDP2LogicContext(b []byte) (LogicContext, error) {
	var ret LogicContext

	c, err := ParseSDP2RawContext(b)
	if err != nil {
		return ret, err
	}

	for _, md := range c.MediaDescList {
		switch md.M.Media {
		case "audio":
			ret.hasAudio = true
			ret.AudioClockRate = md.ARTPMap.ClockRate
			ret.audioAControl = md.AControl.Value

			ret.audioPayloadTypeOrigin = md.ARTPMap.PayloadType
			if md.ARTPMap.EncodingName == ARTPMapEncodingNameAAC {
				ret.audioPayloadTypeBase = base.AVPacketPTAAC
				if md.AFmtPBase != nil {
					ret.ASC, err = ParseASC(md.AFmtPBase)
					if err != nil {
						return ret, err
					}
				} else {
					nazalog.Warnf("aac afmtp not exist.")
				}
			} else {
				ret.audioPayloadTypeBase = base.AVPacketPTUnknown
			}
		case "video":
			ret.hasVideo = true
			ret.VideoClockRate = md.ARTPMap.ClockRate
			ret.videoAControl = md.AControl.Value

			ret.videoPayloadTypeOrigin = md.ARTPMap.PayloadType
			switch md.ARTPMap.EncodingName {
			case ARTPMapEncodingNameH264:
				ret.videoPayloadTypeBase = base.AVPacketPTAVC
				if md.AFmtPBase != nil {
					ret.SPS, ret.PPS, err = ParseSPSPPS(md.AFmtPBase)
					if err != nil {
						return ret, err
					}
				} else {
					nazalog.Warnf("avc afmtp not exist.")
				}
			case ARTPMapEncodingNameH265:
				ret.videoPayloadTypeBase = base.AVPacketPTHEVC
				if md.AFmtPBase != nil {
					ret.VPS, ret.SPS, ret.PPS, err = ParseVPSSPSPPS(md.AFmtPBase)
					if err != nil {
						return ret, err
					}
				} else {
					nazalog.Warnf("hevc afmtp not exist.")
				}
			default:
				ret.videoPayloadTypeBase = base.AVPacketPTUnknown
			}
		}
	}

	return ret, nil
}
