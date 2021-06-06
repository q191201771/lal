// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

// 文档见： https://pengrl.com/p/20101/

const HttpNotifyVersion = "v0.1.0"

type SessionEventCommonInfo struct {
	Protocol      string `json:"protocol"`
	SessionId     string `json:"session_id"`
	RemoteAddr    string `json:"remote_addr"`
	ServerId      string `json:"server_id"`
	Url           string `json:"url"`
	AppName       string `json:"app_name"`
	StreamName    string `json:"stream_name"`
	UrlParam      string `json:"url_param"`
	HasInSession  bool   `json:"has_in_session"`
	HasOutSession bool   `json:"has_out_session"`
}

type UpdateInfo struct {
	ServerId string      `json:"server_id"`
	Groups   []StatGroup `json:"groups"`
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

type RtmpConnectInfo struct {
	ServerId   string `json:"server_id"`
	SessionId  string `json:"session_id"`
	RemoteAddr string `json:"remote_addr"`
	App        string `json:"app"`
	FlashVer   string `json:"flashVer"`
	TcUrl      string `json:"tcUrl"`
}
