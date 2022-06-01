// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

// 文档见： https://pengrl.com/lal/#/HTTPAPI

// ----- request -------------------------------------------------------------------------------------------------------

const (
	PullRetryNumForever = -1
	PullRetryNumNever   = 0

	AutoStopPullAfterNoOutMsNever       = -1
	AutoStopPullAfterNoOutMsImmediately = 0

	RtspModeTcp = 0
	RtspModeUdp = 1
)

type ApiCtrlStartRelayPullReq struct {
	Url                      string `json:"url"`
	StreamName               string `json:"stream_name"`
	PullTimeoutMs            int    `json:"pull_timeout_ms"`
	PullRetryNum             int    `json:"pull_retry_num"`
	AutoStopPullAfterNoOutMs int    `json:"auto_stop_pull_after_no_out_ms"`
	RtspMode                 int    `json:"rtsp_mode"`
}

type ApiCtrlKickSession struct {
	StreamName string `json:"stream_name"`
	SessionId  string `json:"session_id"`
}

// ----- response ------------------------------------------------------------------------------------------------------

const (
	ErrorCodeSucc = 0
	DespSucc      = "succ"

	ErrorCodeGroupNotFound   = 1001
	DespGroupNotFound        = "group not found"
	ErrorCodeParamMissing    = 1002
	DespParamMissing         = "param missing"
	ErrorCodeSessionNotFound = 1003
	DespSessionNotFound      = "session not found"

	ErrorCodeStartRelayPullFail = 2001
)

// HttpResponseBasic
//
// TODO(chef): 因为ILalserver会直接使用这个接口，所以重命名为ApiResponseBasic
//
type HttpResponseBasic struct {
	ErrorCode int    `json:"error_code"`
	Desp      string `json:"desp"`
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
		StreamName string `json:"stream_name"`
		SessionId  string `json:"session_id"`
	} `json:"data"`
}

type ApiCtrlStopRelayPull struct {
	HttpResponseBasic
	Data struct {
		SessionId string `json:"session_id"`
	} `json:"data"`
}
