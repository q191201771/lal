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
	"net/http"
	"sync"
	"time"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/nazahttp"

	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazalog"
)

type PullSessionOption struct {
	// 从调用Pull函数，到接收音视频数据的前一步，也即发送完HTTP请求的超时时间
	// 如果为0，则没有超时时间
	PullTimeoutMs int

	ReadTimeoutMs int // 接收数据超时，单位毫秒，如果为0，则不设置超时
}

var defaultPullSessionOption = PullSessionOption{
	PullTimeoutMs: 10000,
	ReadTimeoutMs: 0,
}

type PullSession struct {
	uniqueKey string            // const after ctor
	option    PullSessionOption // const after ctor

	conn         connection.Connection
	prevConnStat connection.Stat
	staleStat    *connection.Stat
	stat         base.StatSession

	urlCtx base.UrlContext

	disposeOnce sync.Once
}

type ModPullSessionOption func(option *PullSessionOption)

func NewPullSession(modOptions ...ModPullSessionOption) *PullSession {
	option := defaultPullSessionOption
	for _, fn := range modOptions {
		fn(&option)
	}

	uk := base.GenUkFlvPullSession()
	s := &PullSession{
		uniqueKey: uk,
		option:    option,
	}
	nazalog.Infof("[%s] lifecycle new httpflv PullSession. session=%p", uk, s)
	return s
}

// @param tag: 底层保证回调上来的Raw数据长度是完整的（但是不会分析Raw内部的编码数据）
type OnReadFlvTag func(tag Tag)

// Pull 阻塞直到和对端完成拉流前，握手部分的工作，或者发生错误
//
// 注意，握手指的是发送完HTTP Request，不包含接收任何数据，因为有的httpflv服务端，如果流不存在不会发送任何内容，此时我们也应该认为是握手完成了
//
// @param rawUrl 支持如下两种格式。（当然，关键点是对端支持）
//               http://{domain}/{app_name}/{stream_name}.flv
//               http://{ip}/{domain}/{app_name}/{stream_name}.flv
//
// @param onReadFlvTag 读取到 flv tag 数据时回调。回调结束后，PullSession 不会再使用这块 <tag> 数据。
//
func (session *PullSession) Pull(rawUrl string, onReadFlvTag OnReadFlvTag) error {
	nazalog.Debugf("[%s] pull. url=%s", session.uniqueKey, rawUrl)

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	if session.option.PullTimeoutMs == 0 {
		ctx, cancel = context.WithCancel(context.Background())
	} else {
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(session.option.PullTimeoutMs)*time.Millisecond)
	}
	defer cancel()
	return session.pullContext(ctx, rawUrl, onReadFlvTag)
}

// ---------------------------------------------------------------------------------------------------------------------
// IClientSessionLifecycle interface
// ---------------------------------------------------------------------------------------------------------------------

// Dispose 文档请参考： IClientSessionLifecycle interface
//
func (session *PullSession) Dispose() error {
	return session.dispose(nil)
}

// WaitChan 文档请参考： IClientSessionLifecycle interface
//
func (session *PullSession) WaitChan() <-chan error {
	return session.conn.Done()
}

// ---------------------------------------------------------------------------------------------------------------------

// 文档请参考： interface ISessionUrlContext
func (session *PullSession) Url() string {
	return session.urlCtx.Url
}

// 文档请参考： interface ISessionUrlContext
func (session *PullSession) AppName() string {
	return session.urlCtx.PathWithoutLastItem
}

// 文档请参考： interface ISessionUrlContext
func (session *PullSession) StreamName() string {
	return session.urlCtx.LastItemOfPath
}

// 文档请参考： interface ISessionUrlContext
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

func (session *PullSession) pullContext(ctx context.Context, rawUrl string, onReadFlvTag OnReadFlvTag) error {
	errChan := make(chan error, 1)

	// 异步握手
	go func() {
		if err := session.connect(rawUrl); err != nil {
			errChan <- err
			return
		}
		if err := session.writeHttpRequest(); err != nil {
			errChan <- err
			return
		}

		errChan <- nil
	}()

	// 等待握手结果，或者超时通知
	select {
	case <-ctx.Done():
		// 注意，如果超时，可能连接已经建立了，要dispose避免泄漏
		_ = session.dispose(nil)
		return ctx.Err()
	case err := <-errChan:
		// 握手消息，不为nil则握手失败
		if err != nil {
			_ = session.dispose(err)
			return err
		}
	}

	// 握手成功，开启收数据协程
	go session.runReadLoop(onReadFlvTag)
	return nil
}

func (session *PullSession) connect(rawUrl string) (err error) {
	session.urlCtx, err = base.ParseHttpflvUrl(rawUrl, false)
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
		option.WriteTimeoutMs = session.option.ReadTimeoutMs // TODO chef: 为什么是 Read 赋值给 Write
		option.ReadTimeoutMs = session.option.ReadTimeoutMs
	})
	return nil
}

func (session *PullSession) writeHttpRequest() error {
	// # 发送 http GET 请求
	nazalog.Debugf("[%s] > W http request. GET %s", session.uniqueKey, session.urlCtx.PathWithRawQuery)
	req := fmt.Sprintf("GET %s HTTP/1.0\r\nUser-Agent: %s\r\nAccept: */*\r\nRange: byte=0-\r\nConnection: close\r\nHost: %s\r\nIcy-MetaData: 1\r\n\r\n",
		session.urlCtx.PathWithRawQuery, base.LalHttpflvPullSessionUa, session.urlCtx.StdHost)
	_, err := session.conn.Write([]byte(req))
	return err
}

func (session *PullSession) readHttpRespHeader() (statusLine string, headers http.Header, err error) {
	// TODO chef: timeout
	if statusLine, headers, err = nazahttp.ReadHttpHeader(session.conn); err != nil {
		return
	}
	_, code, _, err := nazahttp.ParseHttpStatusLine(statusLine)
	if err != nil {
		return
	}

	nazalog.Debugf("[%s] < R http response header. code=%s", session.uniqueKey, code)
	return
}

func (session *PullSession) readFlvHeader() ([]byte, error) {
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

func (session *PullSession) runReadLoop(onReadFlvTag OnReadFlvTag) {
	var err error
	defer func() {
		_ = session.dispose(err)
	}()

	if _, _, err = session.readHttpRespHeader(); err != nil {
		return
	}

	if _, err = session.readFlvHeader(); err != nil {
		return
	}

	for {
		var tag Tag
		tag, err = session.readTag()
		if err != nil {
			return
		}
		onReadFlvTag(tag)
	}
}

func (session *PullSession) dispose(err error) error {
	var retErr error
	session.disposeOnce.Do(func() {
		nazalog.Infof("[%s] lifecycle dispose httpflv PullSession. err=%+v", session.uniqueKey, err)
		if session.conn == nil {
			retErr = base.ErrSessionNotStarted
			return
		}
		retErr = session.conn.Close()
	})
	return retErr
}
