// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/q191201771/naza/pkg/nazahttp"

	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"
)

type PullSessionOption struct {
	ConnectTimeoutMS int // TCP连接时超时，单位毫秒，如果为0，则不设置超时
	ReadTimeoutMS    int // 接收数据超时，单位毫秒，如果为0，则不设置超时
}

var defaultPullSessionOption = PullSessionOption{
	ConnectTimeoutMS: 0,
	ReadTimeoutMS:    0,
}

type PullSession struct {
	UniqueKey string

	Conn connection.Connection

	option PullSessionOption

	host          string
	pathWithQuery string
	addr          string
}

type ModPullSessionOption func(option *PullSessionOption)

func NewPullSession(modOptions ...ModPullSessionOption) *PullSession {
	option := defaultPullSessionOption
	for _, fn := range modOptions {
		fn(&option)
	}

	uk := unique.GenUniqueKey("FLVPULL")
	nazalog.Infof("[%s] lifecycle new PullSession.", uk)
	return &PullSession{
		option:    option,
		UniqueKey: uk,
	}
}

type OnReadFLVTag func(tag Tag)

// 阻塞直到拉流失败
//
// @param rawURL 支持如下两种格式。（当然，前提是对端支持）
// http://{domain}/{app_name}/{stream_name}.flv
// http://{ip}/{domain}/{app_name}/{stream_name}.flv
//
// @param onReadFLVTag 读取到 flv tag 数据时回调。回调结束后，PullSession 不会再使用这块 <tag> 数据。
func (session *PullSession) Pull(rawURL string, onReadFLVTag OnReadFLVTag) error {
	if err := session.Connect(rawURL); err != nil {
		return err
	}
	if err := session.WriteHTTPRequest(); err != nil {
		return err
	}

	return session.runReadLoop(onReadFLVTag)
}

func (session *PullSession) Dispose() {
	nazalog.Infof("[%s] lifecycle dispose PullSession.", session.UniqueKey)
	_ = session.Conn.Close()
}

func (session *PullSession) Connect(rawURL string) error {
	// # 从 url 中解析 host uri addr
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme != "http" || !strings.HasSuffix(u.Path, ".flv") {
		return ErrHTTPFLV
	}

	session.host = u.Host
	if u.RawQuery == "" {
		session.pathWithQuery = u.Path
	} else {
		session.pathWithQuery = fmt.Sprintf("%s?%s", u.Path, u.RawQuery)
	}

	if strings.Contains(session.host, ":") {
		session.addr = session.host
	} else {
		session.addr = session.host + ":80"
	}

	nazalog.Debugf("[%s] > tcp connect.", session.UniqueKey)

	// # 建立连接
	conn, err := net.DialTimeout("tcp", session.addr, time.Duration(session.option.ConnectTimeoutMS)*time.Millisecond)
	if err != nil {
		return err
	}
	session.Conn = connection.New(conn, func(option *connection.Option) {
		option.ReadBufSize = readBufSize
		option.WriteTimeoutMS = session.option.ReadTimeoutMS // TODO chef: 为什么是 Read 赋值给 Write
		option.ReadTimeoutMS = session.option.ReadTimeoutMS
	})
	return nil
}

func (session *PullSession) WriteHTTPRequest() error {
	// # 发送 http GET 请求
	nazalog.Debugf("[%s] > W http request. GET %s", session.UniqueKey, session.pathWithQuery)
	req := fmt.Sprintf("GET %s HTTP/1.0\r\nAccept: */*\r\nRange: byte=0-\r\nConnection: close\r\nHost: %s\r\nIcy-MetaData: 1\r\n\r\n",
		session.pathWithQuery, session.host)
	_, err := session.Conn.Write([]byte(req))
	return err
}

func (session *PullSession) ReadHTTPRespHeader() (statusLine string, headers map[string]string, err error) {
	// TODO chef: timeout
	if statusLine, headers, err = nazahttp.ReadHTTPHeader(session.Conn); err != nil {
		return
	}
	_, code, _, err := nazahttp.ParseHTTPStatusLine(statusLine)
	if err != nil {
		return
	}

	nazalog.Debugf("[%s] < R http response header. code=%s", session.UniqueKey, code)
	return
}

func (session *PullSession) ReadFLVHeader() ([]byte, error) {
	flvHeader := make([]byte, flvHeaderSize)
	_, err := session.Conn.ReadAtLeast(flvHeader, flvHeaderSize)
	if err != nil {
		return flvHeader, err
	}
	nazalog.Debugf("[%s] < R http flv header.", session.UniqueKey)

	// TODO chef: check flv header's value
	return flvHeader, nil
}

func (session *PullSession) ReadTag() (Tag, error) {
	return readTag(session.Conn)
}

func (session *PullSession) runReadLoop(onReadFLVTag OnReadFLVTag) error {
	if _, _, err := session.ReadHTTPRespHeader(); err != nil {
		return err
	}

	if _, err := session.ReadFLVHeader(); err != nil {
		return err
	}

	for {
		tag, err := session.ReadTag()
		if err != nil {
			return err
		}
		onReadFLVTag(tag)
	}
}
