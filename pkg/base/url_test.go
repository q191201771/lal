// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base_test

import (
	"fmt"
	"testing"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/assert"
)

type in struct {
	rawUrl      string
	defaultPort int
}

// TODO chef: 测试IPv6的case

func TestParseUrl(t *testing.T) {
	// 非法url
	_, err := base.ParseUrl("invalidurl", -1)
	assert.IsNotNil(t, err)

	golden := map[in]base.UrlContext{
		// 常见url，url中无端口，另外设置默认端口
		{rawUrl: "rtmp://127.0.0.1/live/test110", defaultPort: 1935}: {
			Url:                   "rtmp://127.0.0.1/live/test110",
			Scheme:                "rtmp",
			StdHost:               "127.0.0.1",
			HostWithPort:          "127.0.0.1:1935",
			Host:                  "127.0.0.1",
			Port:                  1935,
			PathWithRawQuery:      "/live/test110",
			Path:                  "/live/test110",
			PathWithoutLastItem:   "live",
			LastItemOfPath:        "test110",
			RawQuery:              "",
			RawUrlWithoutUserInfo: "rtmp://127.0.0.1/live/test110",
		},
		// 域名url
		{rawUrl: "rtmp://localhost/live/test110", defaultPort: 1935}: {
			Url:                   "rtmp://localhost/live/test110",
			Scheme:                "rtmp",
			StdHost:               "localhost",
			HostWithPort:          "localhost:1935",
			Host:                  "localhost",
			Port:                  1935,
			PathWithRawQuery:      "/live/test110",
			Path:                  "/live/test110",
			PathWithoutLastItem:   "live",
			LastItemOfPath:        "test110",
			RawQuery:              "",
			RawUrlWithoutUserInfo: "rtmp://localhost/live/test110",
		},
		// 带参数url
		{rawUrl: "rtmp://127.0.0.1/live/test110?a=1", defaultPort: 1935}: {
			Url:                   "rtmp://127.0.0.1/live/test110?a=1",
			Scheme:                "rtmp",
			StdHost:               "127.0.0.1",
			HostWithPort:          "127.0.0.1:1935",
			Host:                  "127.0.0.1",
			Port:                  1935,
			PathWithRawQuery:      "/live/test110?a=1",
			Path:                  "/live/test110",
			PathWithoutLastItem:   "live",
			LastItemOfPath:        "test110",
			RawQuery:              "a=1",
			RawUrlWithoutUserInfo: "rtmp://127.0.0.1/live/test110?a=1",
		},
		// path多级
		{rawUrl: "rtmp://127.0.0.1:19350/a/b/test110", defaultPort: 1935}: {
			Url:                   "rtmp://127.0.0.1:19350/a/b/test110",
			Scheme:                "rtmp",
			StdHost:               "127.0.0.1:19350",
			HostWithPort:          "127.0.0.1:19350",
			Host:                  "127.0.0.1",
			Port:                  19350,
			PathWithRawQuery:      "/a/b/test110",
			Path:                  "/a/b/test110",
			PathWithoutLastItem:   "a/b",
			LastItemOfPath:        "test110",
			RawQuery:              "",
			RawUrlWithoutUserInfo: "rtmp://127.0.0.1:19350/a/b/test110",
		},
		// url中无端口，没有设置默认端口
		{rawUrl: "rtmp://127.0.0.1/live/test110?a=1", defaultPort: -1}: {
			Url:                   "rtmp://127.0.0.1/live/test110?a=1",
			Scheme:                "rtmp",
			StdHost:               "127.0.0.1",
			HostWithPort:          "127.0.0.1:1935",
			Host:                  "127.0.0.1",
			Port:                  1935,
			PathWithRawQuery:      "/live/test110?a=1",
			Path:                  "/live/test110",
			PathWithoutLastItem:   "live",
			LastItemOfPath:        "test110",
			RawQuery:              "a=1",
			RawUrlWithoutUserInfo: "rtmp://127.0.0.1/live/test110?a=1",
		},
		// url中有端口，设置默认端口
		{rawUrl: "rtmp://127.0.0.1:19350/live/test110?a=1", defaultPort: 1935}: {
			Url:                   "rtmp://127.0.0.1:19350/live/test110?a=1",
			Scheme:                "rtmp",
			StdHost:               "127.0.0.1:19350",
			HostWithPort:          "127.0.0.1:19350",
			Host:                  "127.0.0.1",
			Port:                  19350,
			PathWithRawQuery:      "/live/test110?a=1",
			Path:                  "/live/test110",
			PathWithoutLastItem:   "live",
			LastItemOfPath:        "test110",
			RawQuery:              "a=1",
			RawUrlWithoutUserInfo: "rtmp://127.0.0.1:19350/live/test110?a=1",
		},
		// 无path
		{rawUrl: "rtmp://127.0.0.1:19350", defaultPort: 1935}: {
			Url:                   "rtmp://127.0.0.1:19350",
			Scheme:                "rtmp",
			StdHost:               "127.0.0.1:19350",
			HostWithPort:          "127.0.0.1:19350",
			Host:                  "127.0.0.1",
			Port:                  19350,
			PathWithRawQuery:      "",
			Path:                  "",
			PathWithoutLastItem:   "",
			LastItemOfPath:        "",
			RawQuery:              "",
			RawUrlWithoutUserInfo: "rtmp://127.0.0.1:19350",
		},
		// 无path2
		{rawUrl: "rtmp://127.0.0.1:19350/", defaultPort: 1935}: {
			Url:                   "rtmp://127.0.0.1:19350/",
			Scheme:                "rtmp",
			StdHost:               "127.0.0.1:19350",
			HostWithPort:          "127.0.0.1:19350",
			Host:                  "127.0.0.1",
			Port:                  19350,
			PathWithRawQuery:      "/",
			Path:                  "/",
			PathWithoutLastItem:   "",
			LastItemOfPath:        "",
			RawQuery:              "",
			RawUrlWithoutUserInfo: "rtmp://127.0.0.1:19350/",
		},
		// path不完整
		{rawUrl: "rtmp://127.0.0.1:19350/live", defaultPort: 1935}: {
			Url:                   "rtmp://127.0.0.1:19350/live",
			Scheme:                "rtmp",
			StdHost:               "127.0.0.1:19350",
			HostWithPort:          "127.0.0.1:19350",
			Host:                  "127.0.0.1",
			Port:                  19350,
			PathWithRawQuery:      "/live",
			Path:                  "/live",
			PathWithoutLastItem:   "",
			LastItemOfPath:        "live",
			RawQuery:              "",
			RawUrlWithoutUserInfo: "rtmp://127.0.0.1:19350/live",
		},
		// path不完整2
		{rawUrl: "rtmp://127.0.0.1:19350/live/", defaultPort: 1935}: {
			Url:                   "rtmp://127.0.0.1:19350/live/",
			Scheme:                "rtmp",
			StdHost:               "127.0.0.1:19350",
			HostWithPort:          "127.0.0.1:19350",
			Host:                  "127.0.0.1",
			Port:                  19350,
			PathWithRawQuery:      "/live/",
			Path:                  "/live/",
			PathWithoutLastItem:   "live",
			LastItemOfPath:        "",
			RawQuery:              "",
			RawUrlWithoutUserInfo: "rtmp://127.0.0.1:19350/live/",
		},
	}

	for k, v := range golden {
		ctx, err := base.ParseUrl(k.rawUrl, k.defaultPort)
		assert.Equal(t, nil, err)
		assert.Equal(t, v, ctx, k.rawUrl)
	}
}

func TestParseRtmpUrl(t *testing.T) {
	//testParseRtmpUrlCase1(t)
	testParseRtmpUrlCase2(t)
}

func TestParseRtspUrl(t *testing.T) {
	golden := map[string]base.UrlContext{
		// 其他测试见ParseUrl
		// 密码中有@
		"rtsp://admin:P!@1988@127.0.0.1:5554/h264/ch33/main/av_stream": {
			Url:                   "rtsp://admin:P!@1988@127.0.0.1:5554/h264/ch33/main/av_stream",
			Scheme:                "rtsp",
			Username:              "admin",
			Password:              "P!@1988",
			StdHost:               "127.0.0.1:5554",
			HostWithPort:          "127.0.0.1:5554",
			Host:                  "127.0.0.1",
			Port:                  5554,
			PathWithRawQuery:      "/h264/ch33/main/av_stream",
			Path:                  "/h264/ch33/main/av_stream",
			PathWithoutLastItem:   "h264/ch33/main",
			LastItemOfPath:        "av_stream",
			RawQuery:              "",
			RawUrlWithoutUserInfo: "rtsp://127.0.0.1:5554/h264/ch33/main/av_stream",
		},
		// 没有url path
		"rtsp://admin:abd@123@192.168.1.71": {
			Url:                   "rtsp://admin:abd@123@192.168.1.71",
			Scheme:                "rtsp",
			Username:              "admin",
			Password:              "abd@123",
			StdHost:               "192.168.1.71",
			HostWithPort:          "192.168.1.71:554",
			Host:                  "192.168.1.71",
			Port:                  554,
			PathWithRawQuery:      "",
			Path:                  "",
			PathWithoutLastItem:   "",
			LastItemOfPath:        "",
			RawQuery:              "",
			RawUrlWithoutUserInfo: "rtsp://192.168.1.71",
		},
	}
	for k, v := range golden {
		ctx, err := base.ParseRtspUrl(k)
		assert.Equal(t, nil, err)
		assert.Equal(t, v, ctx, k)
	}
}

func testParseRtmpUrlCase1(t *testing.T) {
	golden := map[string]base.UrlContext{
		// 特殊case，其他测试见ParseUrl
		"rtmp://127.0.0.1/test110": {
			Url:                   "rtmp://127.0.0.1/test110",
			Scheme:                "rtmp",
			StdHost:               "127.0.0.1",
			HostWithPort:          "127.0.0.1:1935",
			Host:                  "127.0.0.1",
			Port:                  1935,
			PathWithRawQuery:      "/test110",
			Path:                  "/test110",
			PathWithoutLastItem:   "test110",
			LastItemOfPath:        "",
			RawQuery:              "",
			RawUrlWithoutUserInfo: "rtmp://127.0.0.1/test110",
		},
	}
	for k, v := range golden {
		ctx, err := base.ParseRtmpUrl(k)
		assert.Equal(t, nil, err)
		assert.Equal(t, v, ctx, k)
	}
}

func testParseRtmpUrlCase2(t *testing.T) {
	// TODO(chef): [refactor] 抽象一个解析rtmp url的接口，让业务方有手段自己实现 202207
	var appNameFn = func(ctx base.UrlContext) string {
		return ctx.PathWithoutLastItem
	}

	var tcUrlFn = func(ctx base.UrlContext) string {
		return fmt.Sprintf("%s://%s/%s", ctx.Scheme, ctx.StdHost, ctx.PathWithoutLastItem)
	}

	var streamNameWithRawQueryFn = func(ctx base.UrlContext) string {
		if ctx.RawQuery == "" {
			return ctx.LastItemOfPath
		}
		return fmt.Sprintf("%s?%s", ctx.LastItemOfPath, ctx.RawQuery)
	}

	url := "rtmp://xxx.com:1935/vyun?vhost=thirdVhost?token=88F4/lss_7"
	//url := "rtmp://rs.live.vhou.net/vhall?vhost=thirdVhost?token=2A317D14t25690"
	ctx, err := base.ParseRtmpUrl(url)
	assert.Equal(t, nil, err)
	assert.Equal(t, "vyun?vhost=thirdVhost?token=88F4", appNameFn(ctx))
	assert.Equal(t, "rtmp://xxx.com:1935/vyun?vhost=thirdVhost?token=88F4", tcUrlFn(ctx))
	assert.Equal(t, "lss_7", streamNameWithRawQueryFn(ctx))

	url = "rtmp://xxx.net/vhall?vhost=thirdVhost?token=2A317D14t25690/138521921"
	ctx, err = base.ParseRtmpUrl(url)
	assert.Equal(t, nil, err)
	assert.Equal(t, "vhall?vhost=thirdVhost?token=2A317D14t25690", appNameFn(ctx))
	assert.Equal(t, "rtmp://xxx.net/vhall?vhost=thirdVhost?token=2A317D14t25690", tcUrlFn(ctx))
	assert.Equal(t, "138521921", streamNameWithRawQueryFn(ctx))
}
