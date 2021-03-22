// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base_test

import (
	"testing"

	"github.com/cfeeling/lal/pkg/base"
	"github.com/cfeeling/naza/pkg/assert"
)

type in struct {
	rawURL      string
	defaultPort int
}

// TODO chef: 测试IPv6的case

func TestParseURL(t *testing.T) {
	// 非法url
	_, err := base.ParseURL("invalidurl", -1)
	assert.IsNotNil(t, err)

	golden := map[in]base.URLContext{
		// 常见url，url中无端口，另外设置默认端口
		in{rawURL: "rtmp://127.0.0.1/live/test110", defaultPort: 1935}: {
			URL:                   "rtmp://127.0.0.1/live/test110",
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
			RawURLWithoutUserInfo: "rtmp://127.0.0.1/live/test110",
		},
		// 域名url
		in{rawURL: "rtmp://localhost/live/test110", defaultPort: 1935}: {
			URL:                   "rtmp://localhost/live/test110",
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
			RawURLWithoutUserInfo: "rtmp://localhost/live/test110",
		},
		// 带参数url
		in{rawURL: "rtmp://127.0.0.1/live/test110?a=1", defaultPort: 1935}: {
			URL:                   "rtmp://127.0.0.1/live/test110?a=1",
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
			RawURLWithoutUserInfo: "rtmp://127.0.0.1/live/test110?a=1",
		},
		// path多级
		in{rawURL: "rtmp://127.0.0.1:19350/a/b/test110", defaultPort: 1935}: {
			URL:                   "rtmp://127.0.0.1:19350/a/b/test110",
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
			RawURLWithoutUserInfo: "rtmp://127.0.0.1:19350/a/b/test110",
		},
		// url中无端口，没有设置默认端口
		in{rawURL: "rtmp://127.0.0.1/live/test110?a=1", defaultPort: -1}: {
			URL:                   "rtmp://127.0.0.1/live/test110?a=1",
			Scheme:                "rtmp",
			StdHost:               "127.0.0.1",
			HostWithPort:          "127.0.0.1",
			Host:                  "127.0.0.1",
			Port:                  0,
			PathWithRawQuery:      "/live/test110?a=1",
			Path:                  "/live/test110",
			PathWithoutLastItem:   "live",
			LastItemOfPath:        "test110",
			RawQuery:              "a=1",
			RawURLWithoutUserInfo: "rtmp://127.0.0.1/live/test110?a=1",
		},
		// url中有端口，设置默认端口
		in{rawURL: "rtmp://127.0.0.1:19350/live/test110?a=1", defaultPort: 1935}: {
			URL:                   "rtmp://127.0.0.1:19350/live/test110?a=1",
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
			RawURLWithoutUserInfo: "rtmp://127.0.0.1:19350/live/test110?a=1",
		},
		// 无path
		in{rawURL: "rtmp://127.0.0.1:19350", defaultPort: 1935}: {
			URL:                   "rtmp://127.0.0.1:19350",
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
			RawURLWithoutUserInfo: "rtmp://127.0.0.1:19350",
		},
		// 无path2
		in{rawURL: "rtmp://127.0.0.1:19350/", defaultPort: 1935}: {
			URL:                   "rtmp://127.0.0.1:19350/",
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
			RawURLWithoutUserInfo: "rtmp://127.0.0.1:19350/",
		},
		// path不完整
		in{rawURL: "rtmp://127.0.0.1:19350/live", defaultPort: 1935}: {
			URL:                   "rtmp://127.0.0.1:19350/live",
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
			RawURLWithoutUserInfo: "rtmp://127.0.0.1:19350/live",
		},
		// path不完整2
		in{rawURL: "rtmp://127.0.0.1:19350/live/", defaultPort: 1935}: {
			URL:                   "rtmp://127.0.0.1:19350/live/",
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
			RawURLWithoutUserInfo: "rtmp://127.0.0.1:19350/live/",
		},
	}

	for k, v := range golden {
		ctx, err := base.ParseURL(k.rawURL, k.defaultPort)
		assert.Equal(t, nil, err)
		assert.Equal(t, v, ctx, k.rawURL)
	}
}

func TestParseRTMPURL(t *testing.T) {
	golden := map[string]base.URLContext{
		// 其他测试见ParseURL
		"rtmp://127.0.0.1/test110": {
			URL:                   "rtmp://127.0.0.1/test110",
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
			RawURLWithoutUserInfo: "rtmp://127.0.0.1/test110",
		},
	}
	for k, v := range golden {
		ctx, err := base.ParseRTMPURL(k)
		assert.Equal(t, nil, err)
		assert.Equal(t, v, ctx, k)
	}
}

func TestParseRTSPURL(t *testing.T) {
	golden := map[string]base.URLContext{
		// 其他测试见ParseURL
		"rtsp://admin:P!@1988@127.0.0.1:5554/h264/ch33/main/av_stream": {
			URL:                   "rtsp://admin:P!@1988@127.0.0.1:5554/h264/ch33/main/av_stream",
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
			RawURLWithoutUserInfo: "rtsp://127.0.0.1:5554/h264/ch33/main/av_stream",
		},
	}
	for k, v := range golden {
		ctx, err := base.ParseRTSPURL(k)
		assert.Equal(t, nil, err)
		assert.Equal(t, v, ctx, k)
	}
}
