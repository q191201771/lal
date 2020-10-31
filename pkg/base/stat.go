// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

const (
	// StatGroup.AudioCodec
	AudioCodecAAC = "AAC"

	// StatGroup.VideoCodec
	VideoCodecAVC  = "H264"
	VideoCodecHEVC = "H265"

	// StatSession.Protocol
	ProtocolRTMP    = "RTMP"
	ProtocolRTSP    = "RTSP"
	ProtocolHTTPFLV = "HTTP-FLV"
	ProtocolHTTPTS  = "HTTP-TS"
)

type StatGroup struct {
	StreamName  string    `json:"stream_name"`
	AudioCodec  string    `json:"audio_codec"`
	VideoCodec  string    `json:"video_codec"`
	VideoWidth  int       `json:"video_width"`
	VideoHeight int       `json:"video_height"`
	StatPub     StatPub   `json:"pub"`
	StatSubs    []StatSub `json:"subs"`
	StatPull    StatPull  `json:"pull"`
}

type StatPub struct {
	StatSession
}

type StatSub struct {
	StatSession
}

type StatPull struct {
	StatSession
}

type StatSession struct {
	Protocol      string `json:"protocol"`
	StartTime     string `json:"start_time"`
	RemoteAddr    string `json:"remote_addr"`
	ReadBytesSum  uint64 `json:"read_bytes_sum"`
	WroteBytesSum uint64 `json:"wrote_bytes_sum"`
	Bitrate       int    `json:"bitrate"`
}

func StatSession2Pub(ss StatSession) (ret StatPub) {
	ret.Protocol = ss.Protocol
	ret.StartTime = ss.StartTime
	ret.RemoteAddr = ss.RemoteAddr
	ret.ReadBytesSum = ss.ReadBytesSum
	ret.WroteBytesSum = ss.WroteBytesSum
	ret.Bitrate = ss.Bitrate
	return
}

func StatSession2Sub(ss StatSession) (ret StatSub) {
	ret.Protocol = ss.Protocol
	ret.StartTime = ss.StartTime
	ret.RemoteAddr = ss.RemoteAddr
	ret.ReadBytesSum = ss.ReadBytesSum
	ret.WroteBytesSum = ss.WroteBytesSum
	ret.Bitrate = ss.Bitrate
	return
}

func StatSession2Pull(ss StatSession) (ret StatPull) {
	ret.Protocol = ss.Protocol
	ret.StartTime = ss.StartTime
	ret.RemoteAddr = ss.RemoteAddr
	ret.ReadBytesSum = ss.ReadBytesSum
	ret.WroteBytesSum = ss.WroteBytesSum
	ret.Bitrate = ss.Bitrate
	return
}
