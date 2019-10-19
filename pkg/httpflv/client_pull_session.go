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
	"sync"
	"time"

	"github.com/q191201771/naza/pkg/connection"
	log "github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"
)

type PullSessionConfig struct {
	ConnectTimeoutMS int // TCP连接时超时，单位毫秒，如果为0，则不设置超时
	ReadTimeoutMS    int // 接收数据超时，单位毫秒，如果为0，则不设置超时
}

type PullSession struct {
	UniqueKey string

	config PullSessionConfig

	Conn      connection.Connection
	closeOnce sync.Once

	host string
	uri  string
	addr string

	readFlvTagCB ReadFlvTagCB
}

func NewPullSession(config PullSessionConfig) *PullSession {
	uk := unique.GenUniqueKey("FLVPULL")
	log.Infof("lifecycle new PullSession. [%s]", uk)
	return &PullSession{
		config:    config,
		UniqueKey: uk,
	}
}

type ReadFlvTagCB func(tag *Tag)

// 阻塞直到拉流失败
//
// @param rawURL 支持如下两种格式。（当然，前提是对端支持）
// http://{domain}/{app_name}/{stream_name}.flv
// http://{ip}/{domain}/{app_name}/{stream_name}.flv
//
// @param readFlvTagCB 读取到 flv tag 数据时回调。回调结束后，PullSession不会再使用 <tag> 数据。
func (session *PullSession) Pull(rawURL string, readFlvTagCB ReadFlvTagCB) error {
	if err := session.Connect(rawURL); err != nil {
		return err
	}
	if err := session.WriteHTTPRequest(); err != nil {
		return err
	}

	return session.runReadLoop(readFlvTagCB)
}

func (session *PullSession) Dispose(err error) {
	session.closeOnce.Do(func() {
		log.Infof("lifecycle dispose PullSession. [%s] reason=%v", session.UniqueKey, err)
		if err := session.Conn.Close(); err != nil {
			log.Errorf("conn close error. [%s] err=%v", session.UniqueKey, err)
		}
	})
}

func (session *PullSession) Connect(rawURL string) error {
	// # 从 url 中解析 host uri addr
	url, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if url.Scheme != "http" || !strings.HasSuffix(url.Path, ".flv") {
		return httpFlvErr
	}

	session.host = url.Host
	// TODO chef: uri with url.RawQuery?
	session.uri = url.Path

	if strings.Contains(session.host, ":") {
		session.addr = session.host
	} else {
		session.addr = session.host + ":80"
	}

	// # 建立连接
	conn, err := net.DialTimeout("tcp", session.addr, time.Duration(session.config.ConnectTimeoutMS)*time.Millisecond)
	if err != nil {
		return err
	}
	session.Conn = connection.New(conn, func(option *connection.Option) {
		option.ReadBufSize = readBufSize
		option.WriteTimeoutMS = session.config.ReadTimeoutMS // TODO chef: 为什么是 Read 赋值给 Write
		option.ReadTimeoutMS = session.config.ReadTimeoutMS
	})
	return nil
}

func (session *PullSession) WriteHTTPRequest() error {
	// # 发送 http GET 请求
	req := fmt.Sprintf("GET %s HTTP/1.0\r\nAccept: */*\r\nRange: byte=0-\r\nConnection: close\r\nHost: %s\r\nIcy-MetaData: 1\r\n\r\n",
		session.uri, session.host)
	_, err := session.Conn.Write([]byte(req))
	return err
}

func (session *PullSession) ReadHTTPRespHeader() (firstLine string, headers map[string]string, err error) {
	// TODO chef: timeout
	_, firstLine, headers, err = parseHTTPHeader(session.Conn)
	if err != nil {
		return
	}

	if !strings.Contains(firstLine, "200") || len(headers) == 0 {
		err = httpFlvErr
		return
	}
	log.Infof("-----> http response header. [%s]", session.UniqueKey)

	return
}

func (session *PullSession) ReadFlvHeader() ([]byte, error) {
	flvHeader := make([]byte, flvHeaderSize)
	_, err := session.Conn.ReadAtLeast(flvHeader, flvHeaderSize)
	if err != nil {
		return flvHeader, err
	}
	log.Infof("-----> http flv header. [%s]", session.UniqueKey)

	// TODO chef: check flv header's value
	return flvHeader, nil
}

func (session *PullSession) ReadTag() (*Tag, error) {
	return readTag(session.Conn)
}

func (session *PullSession) runReadLoop(readFlvTagCB ReadFlvTagCB) error {
	if _, _, err := session.ReadHTTPRespHeader(); err != nil {
		return err
	}

	if _, err := session.ReadFlvHeader(); err != nil {
		return err
	}

	for {
		tag, err := session.ReadTag()
		if err != nil {
			return err
		}
		readFlvTagCB(tag)
	}
}
