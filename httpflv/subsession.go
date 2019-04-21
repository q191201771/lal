package httpflv

import (
	"bufio"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

var flvHttpResponseHeader = "HTTP/1.1 200 OK\r\n" +
	"Cache-Control: no-cache\r\n" +
	"Content-Type: video/x-flv\r\n" +
	"Connection: close\r\n" +
	"Expires: -1\r\n" +
	"Pragma: no-cache\r\n" +
	"\r\n"

var flvHeaderBuf13 = []byte{0x46, 0x4c, 0x56, 0x01, 0x05, 0x0, 0x0, 0x0, 0x09, 0x0, 0x0, 0x0, 0x0}

var wChanSize = 32

// TODO chef: type use enum inside Go style.
type writeMsg struct {
	t   int
	pkt []byte
}

type SubSessionStat struct {
	wannaWriteCount int64
	wannaWriteByte  int64
	writeCount      int64
	writeByte       int64
}

type SubSession struct {
	StartTick  int64
	StreamName string
	AppName    string
	Uri        string
	Headers    map[string]string

	conn      net.Conn
	rb        *bufio.Reader
	closeChan chan int
	wChan     chan writeMsg

	stat      SubSessionStat
	prevStat  SubSessionStat
	statMutex sync.Mutex
}

func NewSubSession(conn net.Conn) *SubSession {
	return &SubSession{
		conn:      conn,
		rb:        bufio.NewReaderSize(conn, readBufSize),
		wChan:     make(chan writeMsg, wChanSize),
		closeChan: make(chan int, 1),
	}
}

func (session *SubSession) ReadRequest() error {
	session.StartTick = time.Now().Unix()

	var err error
	var firstLine string
	firstLine, session.Headers, err = parseHttpHeader(session.rb)
	if err != nil {
		return err
	}

	items := strings.Split(string(firstLine), " ")
	if len(items) != 3 || items[0] != "GET" {
		return fxxkErr
	}
	session.Uri = items[1]
	if !strings.HasSuffix(session.Uri, ".flv") {
		return fxxkErr
	}
	//log.Println("uri:", session.uri)
	items = strings.Split(session.Uri, "/")
	if len(items) != 3 {
		return fxxkErr
	}
	session.AppName = items[1]
	items = strings.Split(items[2], ".")
	if len(items) < 2 {
		return fxxkErr
	}
	session.StreamName = items[0]

	return nil
}

func (session *SubSession) RunLoop() error {
	go func() {
		// TODO chef: close by self.
		buf := make([]byte, 128)
		_, err := session.conn.Read(buf)
		if err != nil {
			session.closeChan <- 1
		}
	}()

	err := session.runWriteLoop()
	session.conn.Close()
	return err
}

func (session *SubSession) WritePacket(pkt []byte) {
	session.addWannaWriteStat(len(pkt))
	session.wChan <- writeMsg{t: 2, pkt: pkt}
}

func (session *SubSession) GetStat() (now SubSessionStat, diff SubSessionStat) {
	session.statMutex.Lock()
	defer session.statMutex.Unlock()
	now = session.stat
	diff.wannaWriteCount = session.stat.wannaWriteCount - session.prevStat.wannaWriteCount
	diff.wannaWriteByte = session.stat.wannaWriteByte - session.prevStat.wannaWriteByte
	diff.writeCount = session.stat.writeCount - session.prevStat.writeCount
	diff.writeByte = session.stat.writeByte - session.prevStat.writeByte
	session.prevStat = session.stat
	return
}

func (session *SubSession) ForceClose() {
	session.closeChan <- 2
}

func (session *SubSession) runWriteLoop() error {
	for {
		select {
		case msg := <-session.wChan:
			// TODO chef: fix me write less than pkt
			n, err := session.conn.Write(msg.pkt)
			if err != nil {
				log.Println("sub session write failed. ", err)
				return err
			} else {
				session.addWriteStat(n)
			}

		case closeFlag := <-session.closeChan:
			// TODO chef: hardcode number
			switch closeFlag {
			case 1:
				log.Println("sub session close by peer.")
				return io.EOF
			case 2:
				log.Println("sub session close by self.")
				return nil
			}
		}
	}
}

func (session *SubSession) writeHttpResponseHeader() {
	session.addWannaWriteStat(len(flvHttpResponseHeader))
	session.wChan <- writeMsg{t: 0, pkt: []byte(flvHttpResponseHeader)}
}

func (session *SubSession) writeFlvHeader() {
	session.addWannaWriteStat(len(flvHeaderBuf13))
	session.wChan <- writeMsg{t: 1, pkt: flvHeaderBuf13}
}

func (session *SubSession) addWannaWriteStat(wannaWriteByte int) {
	session.statMutex.Lock()
	defer session.statMutex.Unlock()
	session.stat.wannaWriteByte += int64(wannaWriteByte)
	session.stat.wannaWriteCount++
}

func (session *SubSession) addWriteStat(writeByte int) {
	session.statMutex.Lock()
	defer session.statMutex.Unlock()
	session.stat.writeByte += int64(writeByte)
	session.stat.writeCount++
}
