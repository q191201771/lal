package httpflv

import (
	"bufio"
	"fmt"
	"github.com/q191201771/lal/log"
	"github.com/q191201771/lal/util"
	"io"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
)

var flvHeaderSize = 13

var flvPrevTagFieldSize = 4

type PullSessionStat struct {
	ReadCount int64
	ReadByte  int64
}

type PullSession struct {
	//StartTick int64
	connectTimeout int64
	readTimeout    int64
	ConnStat       util.ConnStat

	obs  PullSessionObserver
	Conn *net.TCPConn // after Connect success, can direct visit net.TCPConn, useful for set socket options.
	rb   *bufio.Reader

	closeOnce sync.Once

	UniqueKey string
}

type PullSessionObserver interface {
	ReadHTTPRespHeaderCB()
	ReadFlvHeaderCB(flvHeader []byte)
	ReadTagCB(tag *Tag) // after cb, PullSession won't use this tag data
}

func NewPullSession(obs PullSessionObserver, connectTimeout int64, readTimeout int64) *PullSession {
	uk := util.GenUniqueKey("FLVPULL")
	log.Infof("lifecycle new PullSession. [%s]", uk)
	return &PullSession{
		connectTimeout: connectTimeout,
		readTimeout:    readTimeout,
		obs:            obs,
		UniqueKey:      uk,
	}
}

// @param timeout: timeout for connect operate. if 0, then no timeout
func (session *PullSession) Connect(rawURL string) error {
	session.ConnStat.Start(session.readTimeout, 0)

	url, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if url.Scheme != "http" || !strings.HasSuffix(url.Path, ".flv") {
		return fxxkErr
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
	if session.connectTimeout == 0 {
		conn, err = net.Dial("tcp", addr)
	} else {
		conn, err = net.DialTimeout("tcp", addr, time.Duration(session.connectTimeout)*time.Second)
	}
	if err != nil {
		return err
	}
	session.Conn = conn.(*net.TCPConn)
	session.rb = bufio.NewReaderSize(session.Conn, readBufSize)

	_, err = fmt.Fprintf(session.Conn,
		"GET %s HTTP/1.0\r\nAccept: */*\r\nRange: byte=0-\r\nConnection: close\r\nHost: %s\r\nIcy-MetaData: 1\r\n\r\n",
		uri, host)
	if err != nil {
		return err
	}

	return nil
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
		session.obs.ReadTagCB(tag)
	}
}

func (session *PullSession) readHTTPRespHeader() error {
	n, firstLine, headers, err := parseHTTPHeader(session.rb)
	if err != nil {
		return err
	}
	session.ConnStat.Read(n)

	if !strings.Contains(firstLine, "200") || len(headers) == 0 {
		return fxxkErr
	}
	log.Infof("-----> http response header. [%s]", session.UniqueKey)

	return nil
}

func (session *PullSession) readFlvHeader() ([]byte, error) {
	flvHeader := make([]byte, flvHeaderSize)
	_, err := io.ReadAtLeast(session.rb, flvHeader, flvHeaderSize)
	if err != nil {
		return flvHeader, err
	}
	session.ConnStat.Read(flvHeaderSize)
	log.Infof("-----> http flv header. [%s]", session.UniqueKey)

	// TODO chef: check flv header's value
	return flvHeader, nil
}

func (session *PullSession) readTag() (*Tag, error) {
	header, rawHeader, err := readTagHeader(session.rb)
	if err != nil {
		return nil, err
	}
	session.ConnStat.Read(tagHeaderSize)

	needed := int(header.DataSize) + flvPrevTagFieldSize
	tag := &Tag{}
	tag.Header = header
	tag.Raw = make([]byte, tagHeaderSize+needed)
	copy(tag.Raw, rawHeader)

	if _, err := io.ReadAtLeast(session.rb, tag.Raw[tagHeaderSize:], needed); err != nil {
		return nil, err
	}
	session.ConnStat.Read(needed)

	return tag, nil
}
