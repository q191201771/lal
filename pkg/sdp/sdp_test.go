// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package sdp

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/nazalog"

	"github.com/q191201771/naza/pkg/assert"
)

var goldenSDP = "v=0" + "\r\n" +
	"o=- 0 0 IN IP6 ::1" + "\r\n" +
	"s=No Name" + "\r\n" +
	"c=IN IP6 ::1" + "\r\n" +
	"t=0 0" + "\r\n" +
	"a=tool:libavformat 57.83.100" + "\r\n" +
	"m=video 0 RTP/AVP 96" + "\r\n" +
	"b=AS:212" + "\r\n" +
	"a=rtpmap:96 H264/90000" + "\r\n" +
	"a=fmtp:96 packetization-mode=1; sprop-parameter-sets=Z2QAIKzZQMApsBEAAAMAAQAAAwAyDxgxlg==,aOvssiw=; profile-level-id=640020" + "\r\n" +
	"a=control:streamid=0" + "\r\n" +
	"m=audio 0 RTP/AVP 97" + "\r\n" +
	"b=AS:30" + "\r\n" +
	"a=rtpmap:97 MPEG4-GENERIC/44100/2" + "\r\n" +
	"a=fmtp:97 profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3; config=1210" + "\r\n" +
	"a=control:streamid=1" + "\r\n"

var goldenSPS = []byte{
	0x67, 0x64, 0x00, 0x20, 0xAC, 0xD9, 0x40, 0xC0, 0x29, 0xB0, 0x11, 0x00, 0x00, 0x03, 0x00, 0x01, 0x00, 0x00, 0x03, 0x00, 0x32, 0x0F, 0x18, 0x31, 0x96,
}

var goldenPPS = []byte{
	0x68, 0xEB, 0xEC, 0xB2, 0x2C,
}

func TestParseSDP2RawContext(t *testing.T) {
	sdpCtx, err := ParseSDP2RawContext([]byte(goldenSDP))
	assert.Equal(t, nil, err)
	nazalog.Debugf("sdp=%+v", sdpCtx)
}

func TestParseARTPMap(t *testing.T) {
	golden := map[string]ARTPMap{
		"rtpmap:96 H264/90000": {
			PayloadType:        96,
			EncodingName:       "H264",
			ClockRate:          90000,
			EncodingParameters: "",
		},
		"rtpmap:97 MPEG4-GENERIC/44100/2": {
			PayloadType:        97,
			EncodingName:       "MPEG4-GENERIC",
			ClockRate:          44100,
			EncodingParameters: "2",
		},
		"a=rtpmap:96 H265/90000": {
			PayloadType:        96,
			EncodingName:       "H265",
			ClockRate:          90000,
			EncodingParameters: "",
		},
	}
	for in, out := range golden {
		actual, err := ParseARTPMap(in)
		assert.Equal(t, nil, err)
		assert.Equal(t, out, actual)
	}
}

func TestParseFmtPBase(t *testing.T) {
	golden := map[string]AFmtPBase{
		"a=fmtp:96 packetization-mode=1; sprop-parameter-sets=Z2QAIKzZQMApsBEAAAMAAQAAAwAyDxgxlg==,aOvssiw=; profile-level-id=640020": {
			Format: 96,
			Parameters: map[string]string{
				"packetization-mode":   "1",
				"sprop-parameter-sets": "Z2QAIKzZQMApsBEAAAMAAQAAAwAyDxgxlg==,aOvssiw=",
				"profile-level-id":     "640020",
			},
		},
		"a=fmtp:97 profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3; config=1210": {
			Format: 97,
			Parameters: map[string]string{
				"profile-level-id": "1",
				"mode":             "AAC-hbr",
				"sizelength":       "13",
				"indexlength":      "3",
				"indexdeltalength": "3",
				"config":           "1210",
			},
		},
		"a=fmtp:96 sprop-vps=QAEMAf//AWAAAAMAkAAAAwAAAwA/ugJA; sprop-sps=QgEBAWAAAAMAkAAAAwAAAwA/oAUCAXHy5bpKTC8BAQAAAwABAAADAA8I; sprop-pps=RAHAc8GJ": {
			Format: 96,
			Parameters: map[string]string{
				"sprop-vps": "QAEMAf//AWAAAAMAkAAAAwAAAwA/ugJA",
				"sprop-sps": "QgEBAWAAAAMAkAAAAwAAAwA/oAUCAXHy5bpKTC8BAQAAAwABAAADAA8I",
				"sprop-pps": "RAHAc8GJ",
			},
		},
	}
	for in, out := range golden {
		actual, err := ParseAFmtPBase(in)
		assert.Equal(t, nil, err)
		assert.Equal(t, out, actual)
	}
}

func TestParseSPSPPS(t *testing.T) {
	s := "a=fmtp:96 packetization-mode=1; sprop-parameter-sets=Z2QAIKzZQMApsBEAAAMAAQAAAwAyDxgxlg==,aOvssiw=; profile-level-id=640020"
	f, err := ParseAFmtPBase(s)
	assert.Equal(t, nil, err)
	sps, pps, err := ParseSPSPPS(&f)
	assert.Equal(t, nil, err)
	assert.Equal(t, goldenSPS, sps)
	assert.Equal(t, goldenPPS, pps)
}

func TestParseASC(t *testing.T) {
	s := "a=fmtp:97 profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3; config=1210"
	f, err := ParseAFmtPBase(s)
	assert.Equal(t, nil, err)
	asc, err := ParseASC(&f)
	assert.Equal(t, nil, err)
	assert.Equal(t, []byte{0x12, 0x10}, asc)
}

// TODO chef 补充assert判断
//[]byte{0x40, 0x01, 0x0c, 0x01, 0xff, 0xff, 0x01, 0x60, 0x00, 0x00, 0x03, 0x00, 0x90, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x3f, 0x95, 0x98, 0x09}
//[]byte{0x42, 0x01, 0x01, 0x01, 0x60, 0x00, 0x00, 0x03, 0x00, 0x90, 0x00, 0x00, 0x03, 0x00, 0x00, 0x03, 0x00, 0x3f, 0xa0, 0x05, 0x02, 0x01, 0x69, 0x65, 0x95, 0x9a, 0x49, 0x32, 0xbc, 0x04, 0x04, 0x00, 0x00, 0x03, 0x00, 0x04, 0x00, 0x00, 0x03, 0x00, 0x3c, 0x20}
//[]byte{0x44, 0x01, 0xc1, 0x72, 0xb4, 0x62, 0x40}
func TestParseVPSSPSPPS(t *testing.T) {
	s := "a=fmtp:96 sprop-vps=QAEMAf//AWAAAAMAkAAAAwAAAwA/ugJA; sprop-sps=QgEBAWAAAAMAkAAAAwAAAwA/oAUCAXHy5bpKTC8BAQAAAwABAAADAA8I; sprop-pps=RAHAc8GJ"
	f, err := ParseAFmtPBase(s)
	assert.Equal(t, nil, err)
	vps, sps, pps, err := ParseVPSSPSPPS(&f)
	assert.Equal(t, nil, err)
	nazalog.Debugf("%s", hex.Dump(vps))
	nazalog.Debugf("%s", hex.Dump(sps))
	nazalog.Debugf("%s", hex.Dump(pps))
}

func TestParseSDP2LogicContext(t *testing.T) {
	ctx, err := ParseSDP2LogicContext([]byte(goldenSDP))
	assert.Equal(t, nil, err)
	assert.Equal(t, true, ctx.hasAudio)
	assert.Equal(t, true, ctx.hasVideo)
	assert.Equal(t, 44100, ctx.AudioClockRate)
	assert.Equal(t, 90000, ctx.VideoClockRate)
	assert.Equal(t, true, ctx.IsAudioPayloadTypeOrigin(97))
	assert.Equal(t, true, ctx.IsVideoPayloadTypeOrigin(96))
	assert.Equal(t, base.AVPacketPTAAC, ctx.GetAudioPayloadTypeBase())
	assert.Equal(t, base.AVPacketPTAVC, ctx.GetVideoPayloadTypeBase())
	assert.Equal(t, "streamid=1", ctx.audioAControl)
	assert.Equal(t, "streamid=0", ctx.videoAControl)
	assert.IsNotNil(t, ctx.ASC)
	assert.Equal(t, nil, ctx.VPS)
	assert.IsNotNil(t, ctx.SPS)
	assert.IsNotNil(t, ctx.PPS)
}

func TestCase2(t *testing.T) {
	golden := `v=0
o=- 2252316233 2252316233 IN IP4 0.0.0.0
s=Media Server
c=IN IP4 0.0.0.0
t=0 0
a=control:*
a=packetization-supported:DH
a=rtppayload-supported:DH
a=range:npt=now-
m=video 0 RTP/AVP 98
a=control:trackID=0
a=framerate:25.000000
a=rtpmap:98 H265/90000
a=fmtp:98 profile-id=1;sprop-sps=QgEBAWAAAAMAsAAAAwAAAwBaoAWCAJBY2uSTL5A=;sprop-pps=RAHA8vA8kA==;sprop-vps=QAEMAf//AWAAAAMAsAAAAwAAAwBarAk=
a=recvonly`
	golden = strings.ReplaceAll(golden, "\n", "\r\n")
	ctx, err := ParseSDP2LogicContext([]byte(golden))
	assert.Equal(t, nil, err)
	assert.Equal(t, false, ctx.hasAudio)
	assert.Equal(t, true, ctx.hasVideo)
	assert.Equal(t, 90000, ctx.VideoClockRate)
	assert.Equal(t, true, ctx.IsVideoPayloadTypeOrigin(98))
	assert.Equal(t, base.AVPacketPTHEVC, ctx.GetVideoPayloadTypeBase())
	assert.Equal(t, "trackID=0", ctx.videoAControl)
	assert.Equal(t, nil, ctx.ASC)
	assert.IsNotNil(t, ctx.VPS)
	assert.IsNotNil(t, ctx.SPS)
	assert.IsNotNil(t, ctx.PPS)
	nazalog.Debugf("%+v", ctx)
}

func TestCase3(t *testing.T) {
	golden := `v=0
o=- 2252310609 2252310609 IN IP4 0.0.0.0
s=Media Server
c=IN IP4 0.0.0.0
t=0 0
a=control:*
a=packetization-supported:DH
a=rtppayload-supported:DH
a=range:npt=now-
m=video 0 RTP/AVP 96
a=control:trackID=0
a=framerate:25.000000
a=rtpmap:96 H264/90000
a=fmtp:96 packetization-mode=1;profile-level-id=4D002A;sprop-parameter-sets=Z00AKp2oHgCJ+WbgICAoAAAfQAAGGoQgAA==,aO48gAA=
a=recvonly
m=audio 0 RTP/AVP 97
a=control:trackID=1
a=rtpmap:97 MPEG4-GENERIC/48000
a=fmtp:97 streamtype=5;profile-level-id=1;mode=AAC-hbr;sizelength=13;indexlength=3;indexdeltalength=3;config=1188
a=recvonly`
	golden = strings.ReplaceAll(golden, "\n", "\r\n")
	ctx, err := ParseSDP2LogicContext([]byte(golden))
	assert.Equal(t, nil, err)
	assert.Equal(t, true, ctx.hasAudio)
	assert.Equal(t, true, ctx.hasVideo)
	assert.Equal(t, 48000, ctx.AudioClockRate)
	assert.Equal(t, 90000, ctx.VideoClockRate)
	assert.Equal(t, true, ctx.IsAudioPayloadTypeOrigin(97))
	assert.Equal(t, true, ctx.IsVideoPayloadTypeOrigin(96))
	assert.Equal(t, base.AVPacketPTAAC, ctx.GetAudioPayloadTypeBase())
	assert.Equal(t, base.AVPacketPTAVC, ctx.GetVideoPayloadTypeBase())
	assert.Equal(t, "trackID=1", ctx.audioAControl)
	assert.Equal(t, "trackID=0", ctx.videoAControl)
	assert.IsNotNil(t, ctx.ASC)
	assert.Equal(t, nil, ctx.VPS)
	assert.IsNotNil(t, ctx.SPS)
	assert.IsNotNil(t, ctx.PPS)
	nazalog.Debugf("%+v", ctx)
}

func TestCase4(t *testing.T) {
	golden := `v=0
o=- 1109162014219182 0 IN IP4 0.0.0.0
s=HIK Media Server V3.4.103
i=HIK Media Server Session Description : standard
e=NONE
c=IN IP4 0.0.0.0
t=0 0
a=control:*
b=AS:1034
a=range:npt=now-
m=video 0 RTP/AVP 96
i=Video Media
a=rtpmap:96 H264/90000
a=fmtp:96 profile-level-id=4D0014;packetization-mode=0;sprop-parameter-sets=Z2QAIK2EAQwgCGEAQwgCGEAQwgCEO1AoA803AQEBQAAAAwBAAAAMoQ==,aO48sA==
a=control:trackID=video
b=AS:1024
m=audio 0 RTP/AVP 8
i=Audio Media
a=rtpmap:8 PCMA/8000
a=control:trackID=audio
b=AS:10
a=Media_header:MEDIAINFO=494D4B48020100000400000111710110401F000000FA000000000000000000000000000000000000;
a=appversion:1.0`
	golden = strings.ReplaceAll(golden, "\n", "\r\n")
	ctx, err := ParseSDP2LogicContext([]byte(golden))
	assert.Equal(t, nil, err)
	assert.Equal(t, true, ctx.hasAudio)
	assert.Equal(t, true, ctx.hasVideo)
	assert.Equal(t, 8000, ctx.AudioClockRate)
	assert.Equal(t, 90000, ctx.VideoClockRate)
	assert.Equal(t, true, ctx.IsAudioPayloadTypeOrigin(8))
	assert.Equal(t, true, ctx.IsVideoPayloadTypeOrigin(96))
	//assert.Equal(t, base.AVPacketPTAAC, ctx.AudioPayloadType)
	assert.Equal(t, base.AVPacketPTAVC, ctx.GetVideoPayloadTypeBase())
	assert.Equal(t, "trackID=audio", ctx.audioAControl)
	assert.Equal(t, "trackID=video", ctx.videoAControl)
	assert.Equal(t, nil, ctx.ASC)
	assert.Equal(t, nil, ctx.VPS)
	assert.IsNotNil(t, ctx.SPS)
	assert.IsNotNil(t, ctx.PPS)
	nazalog.Debugf("%+v", ctx)
}

func TestCase5(t *testing.T) {
	golden := `v=0
o=- 1001 1 IN IP4 192.168.0.221
s=VCP IPC Realtime stream
m=video 0 RTP/AVP 105
c=IN IP4 192.168.0.221
a=control:rtsp://192.168.0.221/media/video1/video
a=rtpmap:105 H264/90000
a=fmtp:105 profile-level-id=64002a; packetization-mode=1; sprop-parameter-sets=Z2QAKq2EAQwgCGEAQwgCGEAQwgCEO1A8ARPyzcBAQFAAAD6AAAnECEA=,aO4xshs=
a=recvonly
m=application 0 RTP/AVP 107
c=IN IP4 192.168.0.221
a=control:rtsp://192.168.0.221/media/video1/metadata
a=rtpmap:107 vnd.onvif.metadata/90000
a=fmtp:107 DecoderTag=h3c-v3 RTCP=0
a=recvonly`
	golden = strings.ReplaceAll(golden, "\n", "\r\n")
	ctx, err := ParseSDP2LogicContext([]byte(golden))
	assert.Equal(t, nil, err)
	assert.Equal(t, false, ctx.hasAudio)
	assert.Equal(t, true, ctx.hasVideo)
	assert.Equal(t, 90000, ctx.VideoClockRate)
	assert.Equal(t, true, ctx.IsVideoPayloadTypeOrigin(105))
	assert.Equal(t, base.AVPacketPTAVC, ctx.GetVideoPayloadTypeBase())
	assert.Equal(t, "rtsp://192.168.0.221/media/video1/video", ctx.videoAControl)
	assert.Equal(t, nil, ctx.VPS)
	assert.IsNotNil(t, ctx.SPS)
	assert.IsNotNil(t, ctx.PPS)
	nazalog.Debugf("%+v", ctx)
}

func TestCase6(t *testing.T) {
	golden := `v=0
o=- 1109162014219182 0 IN IP4 0.0.0.0
s=HIK Media Server V3.4.96
i=HIK Media Server Session Description : standard
e=NONE
c=IN IP4 0.0.0.0
t=0 0
a=control:*
b=AS:2058
a=range:npt=now-
m=video 0 RTP/AVP 96
i=Video Media
a=rtpmap:96 H265/90000
a=control:trackID=video
b=AS:2048
m=audio 0 RTP/AVP 8
i=Audio Media
a=rtpmap:8 PCMA/8000
a=control:trackID=audio
b=AS:10
a=Media_header:MEDIAINFO=494D4B48020100000400050011710110401F000000FA000000000000000000000000000000000000;
a=appversion:1.0`

	golden = strings.ReplaceAll(golden, "\n", "\r\n")
	ctx, err := ParseSDP2LogicContext([]byte(golden))
	assert.Equal(t, nil, err)
	assert.Equal(t, 8000, ctx.AudioClockRate)
	assert.Equal(t, 90000, ctx.VideoClockRate)
	assert.Equal(t, base.AVPacketPTUnknown, ctx.audioPayloadTypeBase)
	assert.Equal(t, base.AVPacketPTHEVC, ctx.videoPayloadTypeBase)
	assert.Equal(t, 8, ctx.audioPayloadTypeOrigin)
	assert.Equal(t, 96, ctx.videoPayloadTypeOrigin)
	assert.Equal(t, "trackID=audio", ctx.audioAControl)
	assert.Equal(t, "trackID=video", ctx.videoAControl)
	assert.Equal(t, nil, ctx.ASC)
	assert.Equal(t, nil, ctx.VPS)
	assert.Equal(t, nil, ctx.SPS)
	assert.Equal(t, nil, ctx.PPS)
	assert.Equal(t, true, ctx.hasAudio)
	assert.Equal(t, true, ctx.hasVideo)
	nazalog.Debugf("%+v", ctx)
}
