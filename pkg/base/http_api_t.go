// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

// 文档见： https://pengrl.com/p/20100/

const HTTPAPIVersion = "v0.1.2"

const (
	ErrorCodeSucc            = 0
	DespSucc                 = "succ"
	ErrorCodeGroupNotFound   = 1001
	DespGroupNotFound        = "group not found"
	ErrorCodeParamMissing    = 1002
	DespParamMissing         = "param missing"
	ErrorCodeSessionNotFound = 1003
	DespSessionNotFound      = "session not found"
)

type HTTPResponseBasic struct {
	ErrorCode int    `json:"error_code"`
	Desp      string `json:"desp"`
}

type LALInfo struct {
	ServerID      string `json:"server_id"`
	BinInfo       string `json:"bin_info"`
	LalVersion    string `json:"lal_version"`
	APIVersion    string `json:"api_version"`
	NotifyVersion string `json:"notify_version"`
	StartTime     string `json:"start_time"`
}

type APIStatLALInfo struct {
	HTTPResponseBasic
	Data LALInfo `json:"data"`
}

type APIStatAllGroup struct {
	HTTPResponseBasic
	Data struct {
		Groups []StatGroup `json:"groups"`
	} `json:"data"`
}

type APIStatGroup struct {
	HTTPResponseBasic
	Data *StatGroup `json:"data"`
}

type APICtrlStartPullReq struct {
	Protocol   string `json:"protocol"`
	Addr       string `json:"addr"`
	AppName    string `json:"app_name"`
	StreamName string `json:"stream_name"`
	URLParam   string `json:"url_param"`
}

type APICtrlKickOutSession struct {
	StreamName string `json:"stream_name"`
	SessionID  string `json:"session_id"`
}
