package httpflv

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var flvHeaderSize = 13

var flvPrevTagFieldSize = 4

type PullSessionStat struct {
	readCount int64
	readByte  int64
}

type PullSession struct {
	StartTick int64

	obs          PullSessionObserver
	*net.TCPConn // after Connect success, can direct visit net.TCPConn, useful for set socket options.
	rb           *bufio.Reader
	closed       uint32

	stat      PullSessionStat
	prevStat  PullSessionStat
	statMutex sync.Mutex
}

type PullSessionObserver interface {
	ReadHttpRespHeaderCb()
	ReadFlvHeaderCb(flvHeader []byte)
	ReadTagCb(header *TagHeader, tag []byte)
}

func NewPullSession(obs PullSessionObserver) *PullSession {
	return &PullSession{obs: obs}
}

// @param timeout: timeout for connect operate. if 0, then no timeout
func (session *PullSession) Connect(url string, timeout time.Duration) error {
	session.StartTick = time.Now().Unix()

	if !strings.HasPrefix(url, "http://") || !strings.HasSuffix(url, ".flv") {
		return fxxkErr
	}
	p1 := 7 // len of "http://"
	p2 := strings.Index(url[p1:], "/")
	if p2 == -1 || p2 == 0 || p2 == len(url)-1 {
		return fxxkErr
	}
	p2 += p1

	host := url[p1:p2]
	uri := url[p2:]

	var addr string
	if strings.Contains(host, ":") {
		addr = host
	} else {
		addr = host + ":80"
	}

	var err error
	var conn net.Conn
	if timeout == 0 {
		conn, err = net.Dial("tcp", addr)
	} else {
		conn, err = net.DialTimeout("tcp", addr, timeout)
	}
	if err != nil {
		return err
	}
	session.TCPConn = conn.(*net.TCPConn)
	session.rb = bufio.NewReaderSize(session.TCPConn, readBufSize)

	// TODO chef: write succ len
	_, err = fmt.Fprintf(session.TCPConn,
		"GET %s HTTP/1.0\r\nAccept: */*\r\nRange: byte=0-\r\nConnection: close\r\nHost: %s\r\nIcy-MetaData: 1\r\n\r\n",
		uri, host)
	if err != nil {
		return nil
	}

	return nil
}

// if close by peer, return EOF
func (session *PullSession) RunLoop() error {
	err := session.runReadLoop()
	session.close()
	return err
}

func (session *PullSession) ForceClose() {
	log.Println("force close pull session.")
	session.close()
}

func (session *PullSession) GetStat() (now PullSessionStat, diff PullSessionStat) {
	session.statMutex.Lock()
	defer session.statMutex.Unlock()
	now = session.stat
	diff.readCount = session.stat.readCount - session.prevStat.readCount
	diff.readByte = session.stat.readByte - session.prevStat.readByte
	session.prevStat = session.stat
	return
}

func (session *PullSession) close() {
	if atomic.CompareAndSwapUint32(&session.closed, 0, 1) {
		session.Close()
	}
}

func (session *PullSession) runReadLoop() error {
	err := session.readHttpRespHeader()
	if err != nil {
		return err
	}
	session.obs.ReadHttpRespHeaderCb()

	flvHeader, err := session.readFlvHeader()
	if err != nil {
		return err
	}
	session.obs.ReadFlvHeaderCb(flvHeader)

	for {
		h, tag, err := session.readTag()
		if err != nil {
			return err
		}
		session.obs.ReadTagCb(h, tag)
	}
}

func (session *PullSession) readHttpRespHeader() error {
	firstLine, headers, err := parseHttpHeader(session.rb)
	if err != nil {
		return err
	}

	if !strings.Contains(firstLine, "200") || len(headers) == 0 {
		return fxxkErr
	}
	log.Println("readHttpRespHeader")

	return nil
}

func (session *PullSession) readFlvHeader() ([]byte, error) {
	flvHeader := make([]byte, flvHeaderSize)
	_, err := io.ReadAtLeast(session.rb, flvHeader, flvHeaderSize)
	if err != nil {
		return flvHeader, err
	}
	log.Println("readFlvHeader")
	// TODO chef: check flv header's value
	return flvHeader, nil
}

func (session *PullSession) readTag() (*TagHeader, []byte, error) {
	h, rawHeader, err := readTagHeader(session.rb)
	if err != nil {
		return nil, nil, err
	}
	session.addStat(tagHeaderSize)

	needed := int(h.dataSize) + flvPrevTagFieldSize
	rawBody := make([]byte, needed)
	if _, err := io.ReadAtLeast(session.rb, rawBody, needed); err != nil {
		log.Println(err)
		return nil, nil, err
	}
	session.addStat(needed)

	var tag []byte
	tag = append(tag, rawHeader...)
	tag = append(tag, rawBody...)
	//log.Println(h.t, h.timestamp, h.dataSize)

	return h, tag, nil
}

func (session *PullSession) addStat(readByte int) {
	session.statMutex.Lock()
	defer session.statMutex.Unlock()
	session.stat.readByte += int64(readByte)
	session.stat.readCount++
}
