// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

// t_http_an__.go
//
// http-api和http-notify的共用部分
//

const (
	// AudioCodecAac StatGroup.AudioCodec
	AudioCodecAac = "AAC"

	// VideoCodecAvc StatGroup.VideoCodec
	VideoCodecAvc  = "H264"
	VideoCodecHevc = "H265"
)

type LalInfo struct {
	ServerId      string `json:"server_id"`
	BinInfo       string `json:"bin_info"`
	LalVersion    string `json:"lal_version"`
	ApiVersion    string `json:"api_version"`
	NotifyVersion string `json:"notify_version"`
	StartTime     string `json:"start_time"`
}

type StatGroup struct {
	StreamName  string    `json:"stream_name"`
	AudioCodec  string    `json:"audio_codec"`
	VideoCodec  string    `json:"video_codec"`
	VideoWidth  int       `json:"video_width"`
	VideoHeight int       `json:"video_height"`
	StatPub     StatPub   `json:"pub"`
	StatSubs    []StatSub `json:"subs"` // TODO(chef): [opt] 增加数量字段，因为这里不一定全部放入
	StatPull    StatPull  `json:"pull"`
}

type StatSession struct {
	SessionId string `json:"session_id"`
	Protocol  string `json:"protocol"`
	BaseType  string `json:"base_type"`

	StartTime string `json:"start_time"`

	RemoteAddr string `json:"remote_addr"`

	ReadBytesSum  uint64 `json:"read_bytes_sum"`
	WroteBytesSum uint64 `json:"wrote_bytes_sum"`
	Bitrate       int    `json:"bitrate"`
	ReadBitrate   int    `json:"read_bitrate"`
	WriteBitrate  int    `json:"write_bitrate"`

	typ SessionType
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

// ---------------------------------------------------------------------------------------------------------------------

func Session2StatPub(session ISession) StatPub {
	return StatPub{
		session.GetStat(),
	}
}

func Session2StatSub(session ISession) StatSub {
	return StatSub{
		session.GetStat(),
	}
}

func Session2StatPull(session ISession) StatPull {
	return StatPull{
		session.GetStat(),
	}
}
