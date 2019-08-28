package httpflv

import (
	"github.com/q191201771/nezha/pkg/connection"
	"github.com/q191201771/nezha/pkg/log"
	"github.com/q191201771/nezha/pkg/unique"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

type PullSession struct {
	connectTimeoutMS int
	readTimeoutMS    int

	obs  PullSessionObserver
	Conn connection.Connection

	closeOnce sync.Once

	UniqueKey string
}

type PullSessionObserver interface {
	ReadHTTPRespHeaderCB()
	ReadFlvHeaderCB(flvHeader []byte)
	ReadFlvTagCB(tag *Tag) // after cb, PullSession won't use this tag data
}

// @param connectTimeoutMS TCP连接时超时，单位毫秒，如果为0，则不设置超时
// @param readTimeoutMS 接收数据超时，单位毫秒，如果为0，则不设置超时
func NewPullSession(obs PullSessionObserver, connectTimeoutMS int, readTimeoutMS int) *PullSession {
	uk := unique.GenUniqueKey("FLVPULL")
	log.Infof("lifecycle new PullSession. [%s]", uk)
	return &PullSession{
		connectTimeoutMS: connectTimeoutMS,
		readTimeoutMS:    readTimeoutMS,
		obs:              obs,
		UniqueKey:        uk,
	}
}

// 支持如下两种格式。当然，前提是对端支持
// http://{domain}/{app_name}/{stream_name}.flv
// http://{ip}/{domain}/{app_name}/{stream_name}.flv
func (session *PullSession) Pull(rawURL string) error {
	url, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if url.Scheme != "http" || !strings.HasSuffix(url.Path, ".flv") {
		return httpFlvErr
	}

	host := url.Host
	// TODO chef: uri with url.RawQuery?
	uri := url.Path

	var addr string
	if strings.Contains(host, ":") {
		addr = host
	} else {
		addr = host + ":80"
	}

	var conn net.Conn
	if session.connectTimeoutMS == 0 {
		conn, err = net.Dial("tcp", addr)
	} else {
		conn, err = net.DialTimeout("tcp", addr, time.Duration(session.connectTimeoutMS)*time.Millisecond)
	}
	if err != nil {
		return err
	}
	session.Conn = connection.New(conn, &connection.Config{ReadBufSize: readBufSize})

	_, err = session.Conn.PrintfWithTimeout(
		session.readTimeoutMS,
		"GET %s HTTP/1.0\r\nAccept: */*\r\nRange: byte=0-\r\nConnection: close\r\nHost: %s\r\nIcy-MetaData: 1\r\n\r\n",
		uri, host)

	return err
}

func (session *PullSession) RunLoop() error {
	err := session.runReadLoop()
	session.Dispose(err)
	return err
}

func (session *PullSession) Dispose(err error) {
	session.closeOnce.Do(func() {
		log.Infof("lifecycle dispose PullSession. [%s] reason=%v", session.UniqueKey, err)
		if err := session.Conn.Close(); err != nil {
			log.Error("conn close error. [%s] err=%v", session.UniqueKey, err)
		}
	})
}

func (session *PullSession) runReadLoop() error {
	if err := session.readHTTPRespHeader(); err != nil {
		return err
	}
	// TODO chef: 把内容返回给上层
	session.obs.ReadHTTPRespHeaderCB()

	flvHeader, err := session.readFlvHeader()
	if err != nil {
		return err
	}
	session.obs.ReadFlvHeaderCB(flvHeader)

	for {
		tag, err := session.readTag()
		if err != nil {
			return err
		}
		session.obs.ReadFlvTagCB(tag)
	}
}

func (session *PullSession) readHTTPRespHeader() error {
	// TODO chef: timeout
	_, firstLine, headers, err := parseHTTPHeader(session.Conn)
	if err != nil {
		return err
	}

	if !strings.Contains(firstLine, "200") || len(headers) == 0 {
		return httpFlvErr
	}
	log.Infof("-----> http response header. [%s]", session.UniqueKey)

	return nil
}

func (session *PullSession) readFlvHeader() ([]byte, error) {
	flvHeader := make([]byte, flvHeaderSize)
	_, err := session.Conn.ReadAtLeastWithTimeout(flvHeader, flvHeaderSize, session.readTimeoutMS)
	if err != nil {
		return flvHeader, err
	}
	log.Infof("-----> http flv header. [%s]", session.UniqueKey)

	// TODO chef: check flv header's value
	return flvHeader, nil
}

func (session *PullSession) readTag() (*Tag, error) {
	rawHeader := make([]byte, TagHeaderSize)
	if _, err := session.Conn.ReadAtLeastWithTimeout(rawHeader, TagHeaderSize, session.readTimeoutMS); err != nil {
		return nil, err
	}
	header := parseTagHeader(rawHeader)

	needed := int(header.DataSize) + prevTagFieldSize
	tag := &Tag{}
	tag.Header = header
	tag.Raw = make([]byte, TagHeaderSize+needed)
	copy(tag.Raw, rawHeader)

	if _, err := session.Conn.ReadAtLeastWithTimeout(tag.Raw[TagHeaderSize:], needed, session.readTimeoutMS); err != nil {
		return nil, err
	}

	return tag, nil
}
