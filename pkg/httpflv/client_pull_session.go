// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/nazahttp"

	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazalog"
)

type PullSessionOption struct {
	// 从调用Pull函数，到接收音视频数据的前一步，也即发送完HTTP请求的超时时间
	// 如果为0，则没有超时时间
	PullTimeoutMS int

	ReadTimeoutMS int // 接收数据超时，单位毫秒，如果为0，则不设置超时
}

var defaultPullSessionOption = PullSessionOption{
	PullTimeoutMS: 10000,
	ReadTimeoutMS: 0,
}

type PullSession struct {
	uniqueKey string            // const after ctor
	option    PullSessionOption // const after ctor

	conn         connection.Connection
	prevConnStat connection.Stat
	staleStat    *connection.Stat
	stat         base.StatSession

	urlCtx base.URLContext

	waitChan chan error
}

type ModPullSessionOption func(option *PullSessionOption)

func NewPullSession(modOptions ...ModPullSessionOption) *PullSession {
	option := defaultPullSessionOption
	for _, fn := range modOptions {
		fn(&option)
	}

	uk := base.GenUKFLVPullSession()
	s := &PullSession{
		uniqueKey: uk,
		option:    option,
		waitChan:  make(chan error, 1),
	}
	nazalog.Infof("[%s] lifecycle new httpflv PullSession. session=%p", uk, s)
	return s
}

// @param tag: 底层保证回调上来的Raw数据长度是完整的（但是不会分析Raw内部的编码数据）
type OnReadFLVTag func(tag Tag)

// 阻塞直到和对端完成拉流前，握手部分的工作（也即发送完HTTP Request），或者发生错误
//
// @param rawURL 支持如下两种格式。（当然，关键点是对端支持）
//               http://{domain}/{app_name}/{stream_name}.flv
//               http://{ip}/{domain}/{app_name}/{stream_name}.flv
//
// @param onReadFLVTag 读取到 flv tag 数据时回调。回调结束后，PullSession 不会再使用这块 <tag> 数据。
func (session *PullSession) Pull(rawURL string, onReadFLVTag OnReadFLVTag) error {
	nazalog.Debugf("[%s] pull. url=%s", session.uniqueKey, rawURL)

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if session.option.PullTimeoutMS == 0 {
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(session.option.PullTimeoutMS)*time.Millisecond)
	}
	defer cancel()
	return session.pullContext(ctx, rawURL, onReadFLVTag)
}

// 文档请参考： interface IClientSessionLifecycle
func (session *PullSession) Dispose() error {
	nazalog.Infof("[%s] lifecycle dispose httpflv PullSession.", session.uniqueKey)
	if session.conn == nil {
		return base.ErrSessionNotStarted
	}
	return session.conn.Close()
}

// 文档请参考： interface IClientSessionLifecycle
func (session *PullSession) WaitChan() <-chan error {
	return session.waitChan
}

// 文档请参考： interface ISessionURLContext
func (session *PullSession) URL() string {
	return session.urlCtx.URL
}

// 文档请参考： interface ISessionURLContext
func (session *PullSession) AppName() string {
	return session.urlCtx.PathWithoutLastItem
}

// 文档请参考： interface ISessionURLContext
func (session *PullSession) StreamName() string {
	return session.urlCtx.LastItemOfPath
}

// 文档请参考： interface ISessionURLContext
func (session *PullSession) RawQuery() string {
	return session.urlCtx.RawQuery
}

// 文档请参考： interface IObject
func (session *PullSession) UniqueKey() string {
	return session.uniqueKey
}

// 文档请参考： interface ISessionStat
func (session *PullSession) UpdateStat(intervalSec uint32) {
	currStat := session.conn.GetStat()
	rDiff := currStat.ReadBytesSum - session.prevConnStat.ReadBytesSum
	session.stat.ReadBitrate = int(rDiff * 8 / 1024 / uint64(intervalSec))
	wDiff := currStat.WroteBytesSum - session.prevConnStat.WroteBytesSum
	session.stat.WriteBitrate = int(wDiff * 8 / 1024 / uint64(intervalSec))
	session.stat.Bitrate = session.stat.ReadBitrate
	session.prevConnStat = currStat
}

// 文档请参考： interface ISessionStat
func (session *PullSession) GetStat() base.StatSession {
	connStat := session.conn.GetStat()
	session.stat.ReadBytesSum = connStat.ReadBytesSum
	session.stat.WroteBytesSum = connStat.WroteBytesSum
	return session.stat
}

// 文档请参考： interface ISessionStat
func (session *PullSession) IsAlive() (readAlive, writeAlive bool) {
	currStat := session.conn.GetStat()
	if session.staleStat == nil {
		session.staleStat = new(connection.Stat)
		*session.staleStat = currStat
		return true, true
	}

	readAlive = !(currStat.ReadBytesSum-session.staleStat.ReadBytesSum == 0)
	writeAlive = !(currStat.WroteBytesSum-session.staleStat.WroteBytesSum == 0)
	*session.staleStat = currStat
	return
}

func (session *PullSession) pullContext(ctx context.Context, rawURL string, onReadFLVTag OnReadFLVTag) error {
	errChan := make(chan error, 1)

	go func() {
		if err := session.connect(rawURL); err != nil {
			errChan <- err
			return
		}
		if err := session.writeHTTPRequest(); err != nil {
			errChan <- err
			return
		}

		errChan <- nil
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		if err != nil {
			return err
		}
	}

	go session.runReadLoop(onReadFLVTag)
	return nil
}

func (session *PullSession) connect(rawURL string) (err error) {
	session.urlCtx, err = base.ParseHTTPFLVURL(rawURL, false)
	if err != nil {
		return
	}

	nazalog.Debugf("[%s] > tcp connect.", session.uniqueKey)

	// # 建立连接
	conn, err := net.Dial("tcp", session.urlCtx.HostWithPort)
	if err != nil {
		return err
	}
	session.conn = connection.New(conn, func(option *connection.Option) {
		option.ReadBufSize = readBufSize
		option.WriteTimeoutMS = session.option.ReadTimeoutMS // TODO chef: 为什么是 Read 赋值给 Write
		option.ReadTimeoutMS = session.option.ReadTimeoutMS
	})
	return nil
}

func (session *PullSession) writeHTTPRequest() error {
	// # 发送 http GET 请求
	nazalog.Debugf("[%s] > W http request. GET %s", session.uniqueKey, session.urlCtx.PathWithRawQuery)
	req := fmt.Sprintf("GET %s HTTP/1.0\r\nUser-Agent: %s\r\nAccept: */*\r\nRange: byte=0-\r\nConnection: close\r\nHost: %s\r\nIcy-MetaData: 1\r\n\r\n",
		session.urlCtx.PathWithRawQuery, base.LALHTTPFLVPullSessionUA, session.urlCtx.StdHost)
	_, err := session.conn.Write([]byte(req))
	return err
}

func (session *PullSession) readHTTPRespHeader() (statusLine string, headers map[string]string, err error) {
	// TODO chef: timeout
	if statusLine, headers, err = nazahttp.ReadHTTPHeader(session.conn); err != nil {
		return
	}
	_, code, _, err := nazahttp.ParseHTTPStatusLine(statusLine)
	if err != nil {
		return
	}

	nazalog.Debugf("[%s] < R http response header. code=%s", session.uniqueKey, code)
	return
}

func (session *PullSession) readFLVHeader() ([]byte, error) {
	flvHeader := make([]byte, flvHeaderSize)
	_, err := session.conn.ReadAtLeast(flvHeader, flvHeaderSize)
	if err != nil {
		return flvHeader, err
	}
	nazalog.Debugf("[%s] < R http flv header.", session.uniqueKey)

	// TODO chef: check flv header's value
	return flvHeader, nil
}

func (session *PullSession) readTag() (Tag, error) {
	return readTag(session.conn)
}

func (session *PullSession) runReadLoop(onReadFLVTag OnReadFLVTag) {
	if _, _, err := session.readHTTPRespHeader(); err != nil {
		session.waitChan <- err
		return
	}

	if _, err := session.readFLVHeader(); err != nil {
		session.waitChan <- err
		return
	}

	for {
		tag, err := session.readTag()
		if err != nil {
			session.waitChan <- err
			return
		}
		onReadFLVTag(tag)
	}
}
