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

var ErrURL = errors.New("lal.url: fxxk")

const (
	DefaultRTMPPort  = 1935
	DefaultHTTPPort  = 80
	DefaultHTTPSPort = 443
	DefaultRTSPPort  = 554
)

type URLPathContext struct {
	PathWithRawQuery    string
	Path                string
	PathWithoutLastItem string // 注意，没有前面的'/'，也没有后面的'/'
	LastItemOfPath      string // 注意，没有前面的'/'
	RawQuery            string
}

// TODO chef: 考虑把rawURL也放入其中
type URLContext struct {
	URL string

	Scheme       string
	Username     string
	Password     string
	StdHost      string // host or host:port
	HostWithPort string
	Host         string
	Port         int

	//URLPathContext
	PathWithRawQuery    string
	Path                string
	PathWithoutLastItem string // 注意，没有前面的'/'，也没有后面的'/'
	LastItemOfPath      string // 注意，没有前面的'/'
	RawQuery            string

	RawURLWithoutUserInfo string
}

func ParseURL(rawURL string, defaultPort int) (ctx URLContext, err error) {
	ctx.URL = rawURL

	stdURL, err := url.Parse(rawURL)
	if err != nil {
		return ctx, err
	}
	if stdURL.Scheme == "" {
		return ctx, ErrURL
	}

	ctx.Scheme = stdURL.Scheme
	ctx.StdHost = stdURL.Host
	ctx.Username = stdURL.User.Username()
	ctx.Password, _ = stdURL.User.Password()

	h, p, err := net.SplitHostPort(stdURL.Host)
	if err != nil {
		// url中端口不存r

		ctx.Host = stdURL.Host
		if defaultPort == -1 {
			ctx.HostWithPort = stdURL.Host
		} else {
			ctx.HostWithPort = net.JoinHostPort(stdURL.Host, fmt.Sprintf("%d", defaultPort))
			ctx.Port = defaultPort
		}
	} else {
		// 端口存在

		ctx.Port, err = strconv.Atoi(p)
		if err != nil {
			return ctx, err
		}
		ctx.Host = h
		ctx.HostWithPort = stdURL.Host

	}

	pathCtx, err := parseURLPath(stdURL)
	if err != nil {
		return ctx, err
	}
	ctx.PathWithRawQuery = pathCtx.PathWithRawQuery
	ctx.Path = pathCtx.Path
	ctx.PathWithoutLastItem = pathCtx.PathWithoutLastItem
	ctx.LastItemOfPath = pathCtx.LastItemOfPath
	ctx.RawQuery = pathCtx.RawQuery

	ctx.RawURLWithoutUserInfo = fmt.Sprintf("%s://%s%s", ctx.Scheme, ctx.StdHost, ctx.PathWithRawQuery)
	return ctx, nil
}

func ParseRTMPURL(rawURL string) (ctx URLContext, err error) {
	ctx, err = ParseURL(rawURL, DefaultRTMPPort)
	if err != nil {
		return
	}
	if ctx.Scheme != "rtmp" || ctx.Host == "" || ctx.Path == "" {
		return ctx, ErrURL
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

func ParseHTTPFLVURL(rawURL string, isHTTPS bool) (ctx URLContext, err error) {
	return ParseHTTPURL(rawURL, isHTTPS, ".flv")
}

func ParseHTTPTSURL(rawURL string, isHTTPS bool) (ctx URLContext, err error) {
	return ParseHTTPURL(rawURL, isHTTPS, ".ts")
}

func ParseRTSPURL(rawURL string) (ctx URLContext, err error) {
	ctx, err = ParseURL(rawURL, DefaultRTSPPort)
	if err != nil {
		return
	}
	if ctx.Scheme != "rtsp" || ctx.Host == "" || ctx.Path == "" {
		return ctx, ErrURL
	}

	return
}

func parseURLPath(stdURL *url.URL) (ctx URLPathContext, err error) {
	ctx.Path = stdURL.Path

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

	ctx.RawQuery = stdURL.RawQuery

	if ctx.RawQuery == "" {
		ctx.PathWithRawQuery = ctx.Path
	} else {
		ctx.PathWithRawQuery = fmt.Sprintf("%s?%s", ctx.Path, ctx.RawQuery)
	}

	return ctx, nil
}

func ParseHTTPURL(rawURL string, isHTTPS bool, suffix string) (ctx URLContext, err error) {
	var defaultPort int
	if isHTTPS {
		defaultPort = DefaultHTTPSPort
	} else {
		defaultPort = DefaultHTTPPort
	}

	ctx, err = ParseURL(rawURL, defaultPort)
	if err != nil {
		return
	}
	if (ctx.Scheme != "http" && ctx.Scheme != "https") || ctx.Host == "" || ctx.Path == "" || !strings.HasSuffix(ctx.LastItemOfPath, suffix) {
		return ctx, ErrURL
	}

	return
}
