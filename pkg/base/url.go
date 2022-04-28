// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// 见单元测试

// TODO chef: 考虑部分内容移入naza中

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

	filenameWithoutType string
	fileType            string
}

func (u *UrlContext) GetFilenameWithoutType() string {
	u.calcFilenameAndTypeIfNeeded()
	return u.filenameWithoutType
}

func (u *UrlContext) GetFileType() string {
	u.calcFilenameAndTypeIfNeeded()
	return u.fileType
}

func (u *UrlContext) calcFilenameAndTypeIfNeeded() {
	if len(u.filenameWithoutType) == 0 || len(u.fileType) == 0 {
		ss := strings.Split(u.LastItemOfPath, ".")
		u.filenameWithoutType = ss[0]
		if len(ss) > 1 {
			u.fileType = ss[1]
		}
	}
}

// ---------------------------------------------------------------------------------------------------------------------

// ParseUrl
//
// @param defaultPort: 注意，如果rawUrl中显示指定了端口，则该参数不生效
//                     注意，如果设置为-1，内部依然会对常见协议(http, https, rtmp, rtsp)设置官方默认端口
//
func ParseUrl(rawUrl string, defaultPort int) (ctx UrlContext, err error) {
	ctx.Url = rawUrl

	stdUrl, err := url.Parse(rawUrl)
	if err != nil {
		return ctx, err
	}
	if stdUrl.Scheme == "" {
		return ctx, fmt.Errorf("%w. url=%s", ErrInvalidUrl, rawUrl)
	}
	// 如果不存在，则设置默认的
	if defaultPort == -1 {
		// TODO(chef): 测试大小写的情况
		switch stdUrl.Scheme {
		case "http":
			defaultPort = DefaultHttpPort
		case "https":
			defaultPort = DefaultHttpsPort
		case "rtmp":
			defaultPort = DefaultRtmpPort
		case "rtsp":
			defaultPort = DefaultRtspPort
		}
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

// ---------------------------------------------------------------------------------------------------------------------

func ParseRtmpUrl(rawUrl string) (ctx UrlContext, err error) {
	ctx, err = ParseUrl(rawUrl, -1)
	if err != nil {
		return
	}
	if ctx.Scheme != "rtmp" || ctx.Host == "" || ctx.Path == "" {
		return ctx, fmt.Errorf("%w. url=%s", ErrInvalidUrl, rawUrl)
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

func ParseRtspUrl(rawUrl string) (ctx UrlContext, err error) {
	ctx, err = ParseUrl(rawUrl, -1)
	if err != nil {
		return
	}
	if ctx.Scheme != "rtsp" || ctx.Host == "" || ctx.Path == "" {
		return ctx, fmt.Errorf("%w. url=%s", ErrInvalidUrl, rawUrl)
	}

	return
}

func ParseHttpflvUrl(rawUrl string) (ctx UrlContext, err error) {
	return parseHttpUrl(rawUrl, ".flv")
}

// ---------------------------------------------------------------------------------------------------------------------

// ParseHttpRequest
//
// @return 完整url
//
func ParseHttpRequest(req *http.Request) string {
	// TODO(chef): [refactor] scheme是否能从从req.URL.Scheme获取
	var scheme string
	if req.TLS == nil {
		scheme = "http"
	} else {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s%s", scheme, req.Host, req.RequestURI)
}

// ----- private -------------------------------------------------------------------------------------------------------

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

func parseHttpUrl(rawUrl string, filetype string) (ctx UrlContext, err error) {
	ctx, err = ParseUrl(rawUrl, -1)
	if err != nil {
		return
	}
	if (ctx.Scheme != "http" && ctx.Scheme != "https") || ctx.Host == "" || ctx.Path == "" || !strings.HasSuffix(ctx.LastItemOfPath, filetype) {
		return ctx, fmt.Errorf("%w. url=%s", ErrInvalidUrl, rawUrl)
	}

	return
}
