// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

// 文档见： https://pengrl.com/lal/#/HTTPNotify

type SessionEventCommonInfo struct {
	ServerId string `json:"server_id"`

	SessionId  string `json:"session_id"`
	Protocol   string `json:"protocol"`
	BaseType   string `json:"base_type"`
	RemoteAddr string `json:"remote_addr"`

	Url        string `json:"url"`
	AppName    string `json:"app_name"`
	StreamName string `json:"stream_name"`
	UrlParam   string `json:"url_param"`

	HasInSession  bool `json:"has_in_session"`
	HasOutSession bool `json:"has_out_session"`
}

type UpdateInfo struct {
	ServerId string `json:"server_id"`

	Groups []StatGroup `json:"groups"`
}

type PubStartInfo struct {
	SessionEventCommonInfo
}

type PubStopInfo struct {
	SessionEventCommonInfo
}

type SubStartInfo struct {
	SessionEventCommonInfo
}

type SubStopInfo struct {
	SessionEventCommonInfo
}

type PullStartInfo struct {
	SessionEventCommonInfo
}

type PullStopInfo struct {
	SessionEventCommonInfo
}

type RtmpConnectInfo struct {
	ServerId string `json:"server_id"`

	SessionId  string `json:"session_id"`
	RemoteAddr string `json:"remote_addr"`
	App        string `json:"app"`
	FlashVer   string `json:"flashVer"`
	TcUrl      string `json:"tcUrl"`
}

type HlsMakeTsInfo struct {
	Event          string  `json:"event"`
	StreamName     string  `json:"stream_name"`
	Cwd            string  `json:"cwd"`
	TsFile         string  `json:"ts_file"`
	LiveM3u8File   string  `json:"live_m3u8_file"`
	RecordM3u8File string  `json:"record_m3u8_file"`
	Id             int     `json:"id"`
	Duration       float64 `json:"duration"`
	ServerId       string  `json:"server_id"`
}

// ---------------------------------------------------------------------------------------------------------------------

func Session2PubStartInfo(session ISession) PubStartInfo {
	return PubStartInfo{
		session2EventCommonInfo(session),
	}
}

func Session2PubStopInfo(session ISession) PubStopInfo {
	return PubStopInfo{
		session2EventCommonInfo(session),
	}
}

func Session2SubStartInfo(session ISession) SubStartInfo {
	return SubStartInfo{
		session2EventCommonInfo(session),
	}
}

func Session2SubStopInfo(session ISession) SubStopInfo {
	return SubStopInfo{
		session2EventCommonInfo(session),
	}
}

func Session2PullStartInfo(session ISession) PullStartInfo {
	return PullStartInfo{
		session2EventCommonInfo(session),
	}
}

func Session2PullStopInfo(session ISession) PullStopInfo {
	return PullStopInfo{
		session2EventCommonInfo(session),
	}
}

// ---------------------------------------------------------------------------------------------------------------------

func session2EventCommonInfo(session ISession) SessionEventCommonInfo {
	var info SessionEventCommonInfo
	stat := session.GetStat()
	info.SessionId = stat.SessionId
	info.Protocol = stat.Protocol
	info.BaseType = stat.BaseType
	info.RemoteAddr = stat.RemoteAddr

	info.Url = session.Url()
	info.AppName = session.AppName()
	info.StreamName = session.StreamName()
	info.Url = session.Url()
	info.UrlParam = session.RawQuery()
	return info
}
