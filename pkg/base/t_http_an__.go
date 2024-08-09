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
	AudioCodecAac   = "AAC"
	AudioCodecG711U = "PCMU"
	AudioCodecG711A = "PCMA"
	AudioCodecOpus  = "OPUS"

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
	WebUiVersion  string `json:"WebUiVersion"`
	StartTime     string `json:"start_time"`
}

type StatGroup struct {
	StreamName  string    `json:"stream_name"`
	AppName     string    `json:"app_name"`
	AudioCodec  string    `json:"audio_codec"`
	VideoCodec  string    `json:"video_codec"`
	VideoWidth  int       `json:"video_width"`
	VideoHeight int       `json:"video_height"`
	StatPub     StatPub   `json:"pub"`
	StatSubs    []StatSub `json:"subs"` // TODO(chef): [opt] 增加数量字段，因为这里不一定全部放入
	StatPull    StatPull  `json:"pull"`

	Fps []RecordPerSec `json:"in_frame_per_sec"`
}

type RecordPerSec struct {
	UnixSec int64  `json:"unix_sec"`
	V       uint32 `json:"v"`
}

type StatSession struct {
	SessionId  string `json:"session_id"`
	Protocol   string `json:"protocol"`
	BaseType   string `json:"base_type"`
	RemoteAddr string `json:"remote_addr"`

	StartTime string `json:"start_time"` // 注意，格式见 ReadableNowTime

	ReadBytesSum      uint64 `json:"read_bytes_sum"`
	WroteBytesSum     uint64 `json:"wrote_bytes_sum"`
	BitrateKbits      int    `json:"bitrate_kbits"`
	ReadBitrateKbits  int    `json:"read_bitrate_kbits"`
	WriteBitrateKbits int    `json:"write_bitrate_kbits"`

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

type PeriodRecord struct {
	ringBuf []RecordPerSec
	nRecord int
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

// GetFpsFrom
//
// @note result s.Fps is not ordered
func (s *StatGroup) GetFpsFrom(p *PeriodRecord, nowUnixSec int64) {
	// s.Fps 是输出，p 是输入
	// p.nRecord 是有数据的元素个数，但是如果 nowUnixSec 和元素的 UnixSec 相等，那么这个元素的数据还没有完全记录，这个元素需要被忽略
	// 当前的实现，把 p.nRecord 和 元素的 UnixSec 重置了，也就是被GetFpsFrom获取过的元素，下次将不被获取
	//
	// 新的要解决的问题：
	// 1 获取过的，还可以再次获取
	// 2 数据是排序好的
	// 3 增加字段，最近1秒，5秒，10秒等时间段的fps
	// 4 考虑和bitrate等字段语义统一，详细的数据可以是detail样式的字段

	if s.Fps == nil || cap(s.Fps) < p.nRecord {
		s.Fps = make([]RecordPerSec, p.nRecord)
	} else {
		s.Fps = s.Fps[0:p.nRecord]
	}

	nRecord := 0
	//p.nRecord = 0
	for _, record := range p.ringBuf {
		if record.UnixSec == 0 {
			continue
		}
		if record.UnixSec == nowUnixSec {
			// value at nowUnixSec not completely recorded
			//p.nRecord++
			continue
		}
		s.Fps[nRecord] = record
		nRecord++
		//p.ringBuf[idx].UnixSec = 0
	}
	s.Fps = s.Fps[0:nRecord]
}

func NewPeriodRecord(bufSize int) PeriodRecord {
	return PeriodRecord{
		ringBuf: make([]RecordPerSec, bufSize),
		nRecord: 0,
	}
}

func (p *PeriodRecord) Add(unixSec int64, v uint32) {
	var index int64
	var record RecordPerSec
	index = unixSec % int64(len(p.ringBuf))
	record = p.ringBuf[index]
	if record.UnixSec == unixSec {
		p.ringBuf[index].V = record.V + v
	} else {
		if record.UnixSec == 0 {
			p.nRecord++
		}
		p.ringBuf[index].UnixSec = unixSec
		p.ringBuf[index].V = v
	}
	return
}

func (p *PeriodRecord) Clear() {
	for idx := range p.ringBuf {
		p.ringBuf[idx].UnixSec = 0
		p.ringBuf[idx].V = 0
	}
}
