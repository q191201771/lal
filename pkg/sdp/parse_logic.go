// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package sdp

import (
	"fmt"
	"strings"

	"github.com/q191201771/lal/pkg/base"
)

type LogicContext struct {
	RawSdp []byte

	AudioClockRate int
	VideoClockRate int

	Asc []byte
	Vps []byte
	Sps []byte
	Pps []byte

	audioPayloadTypeBase base.AvPacketPt // lal内部定义的类型
	videoPayloadTypeBase base.AvPacketPt

	audioPayloadTypeOrigin int // 原始类型，sdp或rtp中的类型
	videoPayloadTypeOrigin int
	audioAControl          string
	videoAControl          string

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
	return lc.audioPayloadTypeBase == base.AvPacketPtAac && lc.Asc != nil
}

func (lc *LogicContext) IsVideoUnpackable() bool {
	return lc.videoPayloadTypeBase == base.AvPacketPtAvc ||
		lc.videoPayloadTypeBase == base.AvPacketPtHevc
}

func (lc *LogicContext) IsAudioUri(uri string) bool {
	return lc.audioAControl != "" && strings.HasSuffix(uri, lc.audioAControl)
}

func (lc *LogicContext) IsVideoUri(uri string) bool {
	return lc.videoAControl != "" && strings.HasSuffix(uri, lc.videoAControl)
}

func (lc *LogicContext) HasAudioAControl() bool {
	return lc.audioAControl != ""
}

func (lc *LogicContext) HasVideoAControl() bool {
	return lc.videoAControl != ""
}

func (lc *LogicContext) MakeAudioSetupUri(uri string) string {
	return lc.makeSetupUri(uri, lc.audioAControl)
}

func (lc *LogicContext) MakeVideoSetupUri(uri string) string {
	return lc.makeSetupUri(uri, lc.videoAControl)
}

func (lc *LogicContext) GetAudioPayloadTypeBase() base.AvPacketPt {
	return lc.audioPayloadTypeBase
}

func (lc *LogicContext) GetVideoPayloadTypeBase() base.AvPacketPt {
	return lc.videoPayloadTypeBase
}

func (lc *LogicContext) makeSetupUri(uri string, aControl string) string {
	if strings.HasPrefix(aControl, "rtsp://") {
		return aControl
	}
	return fmt.Sprintf("%s/%s", uri, aControl)
}

func ParseSdp2LogicContext(b []byte) (LogicContext, error) {
	var ret LogicContext

	c, err := ParseSdp2RawContext(b)
	if err != nil {
		return ret, err
	}

	for _, md := range c.MediaDescList {
		switch md.M.Media {
		case "audio":
			ret.hasAudio = true
			ret.AudioClockRate = md.ARtpMap.ClockRate
			ret.audioAControl = md.AControl.Value

			ret.audioPayloadTypeOrigin = md.ARtpMap.PayloadType
			if strings.EqualFold(md.ARtpMap.EncodingName, ARtpMapEncodingNameAac) {
				ret.audioPayloadTypeBase = base.AvPacketPtAac
				if md.AFmtPBase != nil {
					ret.Asc, err = ParseAsc(md.AFmtPBase)
					if err != nil {
						Log.Warnf("parse asc from afmtp failed. err=%+v", err)
					}
				} else {
					Log.Warnf("aac afmtp not exist.")
				}
			} else {
				ret.audioPayloadTypeBase = base.AvPacketPtUnknown
			}
		case "video":
			ret.hasVideo = true
			ret.VideoClockRate = md.ARtpMap.ClockRate
			ret.videoAControl = md.AControl.Value

			ret.videoPayloadTypeOrigin = md.ARtpMap.PayloadType
			switch md.ARtpMap.EncodingName {
			case ARtpMapEncodingNameH264:
				ret.videoPayloadTypeBase = base.AvPacketPtAvc
				if md.AFmtPBase != nil {
					ret.Sps, ret.Pps, err = ParseSpsPps(md.AFmtPBase)
					if err != nil {
						Log.Warnf("parse sps pps from afmtp failed. err=%+v", err)
					}
				} else {
					Log.Warnf("avc afmtp not exist.")
				}
			case ARtpMapEncodingNameH265:
				ret.videoPayloadTypeBase = base.AvPacketPtHevc
				if md.AFmtPBase != nil {
					ret.Vps, ret.Sps, ret.Pps, err = ParseVpsSpsPps(md.AFmtPBase)
					if err != nil {
						Log.Warnf("parse vps sps pps from afmtp failed. err=%+v", err)
					}
				} else {
					Log.Warnf("hevc afmtp not exist.")
				}
			default:
				ret.videoPayloadTypeBase = base.AvPacketPtUnknown
			}
		}
	}

	ret.RawSdp = b
	return ret, nil
}
