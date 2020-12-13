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
	"time"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/nazahttp"

	"github.com/q191201771/naza/pkg/connection"
	"github.com/q191201771/naza/pkg/nazalog"
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
	UniqueKey string            // const after ctor
	option    PullSessionOption // const after ctor

	conn         connection.Connection
	prevConnStat connection.Stat
	staleStat    *connection.Stat
	stat         base.StatSession

	urlCtx base.URLContext
}

type ModPullSessionOption func(option *PullSessionOption)

func NewPullSession(modOptions ...ModPullSessionOption) *PullSession {
	option := defaultPullSessionOption
	for _, fn := range modOptions {
		fn(&option)
	}

	uk := base.GenUniqueKey(base.UKPFLVPullSession)
	s := &PullSession{
		option:    option,
		UniqueKey: uk,
	}
	nazalog.Infof("[%s] lifecycle new httpflv PullSession. session=%p", uk, s)
	return s
}

type OnReadFLVTag func(tag Tag)

// 阻塞直到拉流失败
//
// @param rawURL 支持如下两种格式。（当然，关键点是对端支持）
// http://{domain}/{app_name}/{stream_name}.flv
// http://{ip}/{domain}/{app_name}/{stream_name}.flv
//
// @param onReadFLVTag 读取到 flv tag 数据时回调。回调结束后，PullSession 不会再使用这块 <tag> 数据。
func (session *PullSession) Pull(rawURL string, onReadFLVTag OnReadFLVTag) error {
	if err := session.connect(rawURL); err != nil {
		return err
	}
	if err := session.writeHTTPRequest(); err != nil {
		return err
	}

	return session.runReadLoop(onReadFLVTag)
}

func (session *PullSession) Dispose() {
	nazalog.Infof("[%s] lifecycle dispose httpflv PullSession.", session.UniqueKey)
	_ = session.conn.Close()
}

func (session *PullSession) UpdateStat(interval uint32) {
	currStat := session.conn.GetStat()
	rDiff := currStat.ReadBytesSum - session.prevConnStat.ReadBytesSum
	session.stat.ReadBitrate = int(rDiff * 8 / 1024 / uint64(interval))
	wDiff := currStat.WroteBytesSum - session.prevConnStat.WroteBytesSum
	session.stat.WriteBitrate = int(wDiff * 8 / 1024 / uint64(interval))
	session.stat.Bitrate = session.stat.ReadBitrate
	session.prevConnStat = currStat
}

func (session *PullSession) GetStat() base.StatSession {
	connStat := session.conn.GetStat()
	session.stat.ReadBytesSum = connStat.ReadBytesSum
	session.stat.WroteBytesSum = connStat.WroteBytesSum
	return session.stat
}

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

func (session *PullSession) AppName() string {
	return session.urlCtx.PathWithoutLastItem
}

func (session *PullSession) StreamName() string {
	return session.urlCtx.LastItemOfPath
}

func (session *PullSession) RawQuery() string {
	return session.urlCtx.RawQuery
}

func (session *PullSession) connect(rawURL string) (err error) {
	session.urlCtx, err = base.ParseHTTPFLVURL(rawURL)
	if err != nil {
		return
	}

	nazalog.Debugf("[%s] > tcp connect.", session.UniqueKey)

	// # 建立连接
	conn, err := net.DialTimeout("tcp", session.urlCtx.HostWithPort, time.Duration(session.option.ConnectTimeoutMS)*time.Millisecond)
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
	nazalog.Debugf("[%s] > W http request. GET %s", session.UniqueKey, session.urlCtx.PathWithRawQuery)
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

	nazalog.Debugf("[%s] < R http response header. code=%s", session.UniqueKey, code)
	return
}

func (session *PullSession) readFLVHeader() ([]byte, error) {
	flvHeader := make([]byte, flvHeaderSize)
	_, err := session.conn.ReadAtLeast(flvHeader, flvHeaderSize)
	if err != nil {
		return flvHeader, err
	}
	nazalog.Debugf("[%s] < R http flv header.", session.UniqueKey)

	// TODO chef: check flv header's value
	return flvHeader, nil
}

func (session *PullSession) readTag() (Tag, error) {
	return readTag(session.conn)
}

func (session *PullSession) runReadLoop(onReadFLVTag OnReadFLVTag) error {
	if _, _, err := session.readHTTPRespHeader(); err != nil {
		return err
	}

	if _, err := session.readFLVHeader(); err != nil {
		return err
	}

	for {
		tag, err := session.readTag()
		if err != nil {
			return err
		}
		onReadFLVTag(tag)
	}
}
