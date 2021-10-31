// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// 见单元测试

// TODO chef: 考虑部分内容移入naza中

var ErrUrl = errors.New("lal.url: fxxk")

const (
	DefaultRtmpPort  = 1935
	DefaultHttpPort  = 80
	DefaultHttpsPort = 443
	DefaultRtspPort  = 554
)

type UrlPathContext struct {
	PathWithRawQuery    string
	Path                string
	PathWithoutLastItem string // 注意，没有前面的'/'，也没有后面的'/'
	LastItemOfPath      string // 注意，没有前面的'/'
	RawQuery            string
}

// TODO chef: 考虑把rawUrl也放入其中
type UrlContext struct {
	Url string

	Scheme       string
	Username     string
	Password     string
	StdHost      string // host or host:port
	HostWithPort string
	Host         string
	Port         int

	//UrlPathContext
	PathWithRawQuery    string
	Path                string
	PathWithoutLastItem string // 注意，没有前面的'/'，也没有后面的'/'
	LastItemOfPath      string // 注意，没有前面的'/'
	RawQuery            string

	RawUrlWithoutUserInfo string
}

func ParseUrl(rawUrl string, defaultPort int) (ctx UrlContext, err error) {
	ctx.Url = rawUrl

	stdUrl, err := url.Parse(rawUrl)
	if err != nil {
		return ctx, err
	}
	if stdUrl.Scheme == "" {
		return ctx, ErrUrl
	}

	ctx.Scheme = stdUrl.Scheme
	ctx.StdHost = stdUrl.Host
	ctx.Username = stdUrl.User.Username()
	ctx.Password, _ = stdUrl.User.Password()

	h, p, err := net.SplitHostPort(stdUrl.Host)
	if err != nil {
		// url中端口不存在

		ctx.Host = stdUrl.Host
		if defaultPort == -1 {
			ctx.HostWithPort = stdUrl.Host
		} else {
			ctx.HostWithPort = net.JoinHostPort(stdUrl.Host, fmt.Sprintf("%d", defaultPort))
			ctx.Port = defaultPort
		}
	} else {
		// 端口存在

		ctx.Port, err = strconv.Atoi(p)
		if err != nil {
			return ctx, err
		}
		ctx.Host = h
		ctx.HostWithPort = stdUrl.Host
	}

	pathCtx, err := parseUrlPath(stdUrl)
	if err != nil {
		return ctx, err
	}
	ctx.PathWithRawQuery = pathCtx.PathWithRawQuery
	ctx.Path = pathCtx.Path
	ctx.PathWithoutLastItem = pathCtx.PathWithoutLastItem
	ctx.LastItemOfPath = pathCtx.LastItemOfPath
	ctx.RawQuery = pathCtx.RawQuery

	ctx.RawUrlWithoutUserInfo = fmt.Sprintf("%s://%s%s", ctx.Scheme, ctx.StdHost, ctx.PathWithRawQuery)
	return ctx, nil
}

func ParseRtmpUrl(rawUrl string) (ctx UrlContext, err error) {
	ctx, err = ParseUrl(rawUrl, DefaultRtmpPort)
	if err != nil {
		return
	}
	if ctx.Scheme != "rtmp" || ctx.Host == "" || ctx.Path == "" {
		return ctx, ErrUrl
	}

	// 注意，使用ffmpeg推流时，会把`rtmp://127.0.0.1/test110`中的test110作为appName(streamName则为空)
	// 这种其实已不算十分合法的rtmp url了
	// 我们这里也处理一下，和ffmpeg保持一致
	if ctx.PathWithoutLastItem == "" && ctx.LastItemOfPath != "" {
		tmp := ctx.PathWithoutLastItem
		ctx.PathWithoutLastItem = ctx.LastItemOfPath
		ctx.LastItemOfPath = tmp
	}
	return
}

func ParseHttpflvUrl(rawUrl string, isHttps bool) (ctx UrlContext, err error) {
	return ParseHttpUrl(rawUrl, isHttps, ".flv")
}

func ParseHttptsUrl(rawUrl string, isHttps bool) (ctx UrlContext, err error) {
	return ParseHttpUrl(rawUrl, isHttps, ".ts")
}

func ParseRtspUrl(rawUrl string) (ctx UrlContext, err error) {
	ctx, err = ParseUrl(rawUrl, DefaultRtspPort)
	if err != nil {
		return
	}
	if ctx.Scheme != "rtsp" || ctx.Host == "" || ctx.Path == "" {
		return ctx, ErrUrl
	}

	return
}

func parseUrlPath(stdUrl *url.URL) (ctx UrlPathContext, err error) {
	ctx.Path = stdUrl.Path

	index := strings.LastIndexByte(ctx.Path, '/')
	if index == -1 {
		ctx.PathWithoutLastItem = ""
		ctx.LastItemOfPath = ""
	} else if index == 0 {
		if ctx.Path == "/" {
			ctx.PathWithoutLastItem = ""
			ctx.LastItemOfPath = ""
		} else {
			ctx.PathWithoutLastItem = ""
			ctx.LastItemOfPath = ctx.Path[1:]
		}
	} else {
		ctx.PathWithoutLastItem = ctx.Path[1:index]
		ctx.LastItemOfPath = ctx.Path[index+1:]
	}

	ctx.RawQuery = stdUrl.RawQuery

	if ctx.RawQuery == "" {
		ctx.PathWithRawQuery = ctx.Path
	} else {
		ctx.PathWithRawQuery = fmt.Sprintf("%s?%s", ctx.Path, ctx.RawQuery)
	}

	return ctx, nil
}

func ParseHttpUrl(rawUrl string, isHttps bool, suffix string) (ctx UrlContext, err error) {
	var defaultPort int
	if isHttps {
		defaultPort = DefaultHttpsPort
	} else {
		defaultPort = DefaultHttpPort
	}

	ctx, err = ParseUrl(rawUrl, defaultPort)
	if err != nil {
		return
	}
	if (ctx.Scheme != "http" && ctx.Scheme != "https") || ctx.Host == "" || ctx.Path == "" || !strings.HasSuffix(ctx.LastItemOfPath, suffix) {
		return ctx, ErrUrl
	}

	return
}
