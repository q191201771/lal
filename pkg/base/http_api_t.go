// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

// 文档见： https://pengrl.com/p/20100/

const HttpApiVersion = "v0.2.0"

const (
	ErrorCodeSucc = 0
	DespSucc      = "succ"

	ErrorCodeGroupNotFound   = 1001
	DespGroupNotFound        = "group not found"
	ErrorCodeParamMissing    = 1002
	DespParamMissing         = "param missing"
	ErrorCodeSessionNotFound = 1003
	DespSessionNotFound      = "session not found"

	ErrorCodeStartRelayPullFail = 2004
)

type HttpResponseBasic struct {
	ErrorCode int    `json:"error_code"`
	Desp      string `json:"desp"`
}

type LalInfo struct {
	ServerId      string `json:"server_id"`
	BinInfo       string `json:"bin_info"`
	LalVersion    string `json:"lal_version"`
	ApiVersion    string `json:"api_version"`
	NotifyVersion string `json:"notify_version"`
	StartTime     string `json:"start_time"`
}

type ApiStatLalInfo struct {
	HttpResponseBasic
	Data LalInfo `json:"data"`
}

type ApiStatAllGroup struct {
	HttpResponseBasic
	Data struct {
		Groups []StatGroup `json:"groups"`
	} `json:"data"`
}

type ApiStatGroup struct {
	HttpResponseBasic
	Data *StatGroup `json:"data"`
}

type ApiCtrlStartRelayPull struct {
	HttpResponseBasic
	Data struct {
		SessionId string `json:"session_id"`
	} `json:"data"`
}

// ---------------------------------------------------------------------------------------------------------------------

type ApiCtrlStartRelayPullReq struct {
	Url        string `json:"url"`
	StreamName string `json:"stream_name"`
}

type ApiCtrlKickOutSession struct {
	StreamName string `json:"stream_name"`
	SessionId  string `json:"session_id"`
}
