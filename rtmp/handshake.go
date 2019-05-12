package rtmp

import (
	"bytes"
	"github.com/q191201771/lal/bele"
	"github.com/q191201771/lal/log"
	"io"
	"time"
)

var version = uint8(3)

var c0c1Len = 1537

var c2Len = 1536

var s0s1s2Len = 3073

type HandshakeClient struct {
	c0c1 []byte
	c2   []byte
}

func (c *HandshakeClient) writeC0C1(writer io.Writer) error {
	c.c0c1 = make([]byte, c0c1Len)
	c.c0c1[0] = version
	bele.BEPutUint32(c.c0c1[1:5], uint32(time.Now().Unix()))

	// TODO chef: random [9:]

	_, err := writer.Write(c.c0c1)
	log.Infof("<----- Handshake C0+C1")
	return err
}

func (c *HandshakeClient) readS0S1S2(reader io.Reader) error {
	s0s1s2 := make([]byte, s0s1s2Len)
	if _, err := io.ReadAtLeast(reader, s0s1s2, s0s1s2Len); err != nil {
		return err
	}
	log.Infof("-----> Handshake S0+S1+S2")
	if s0s1s2[0] != version {
		return rtmpErr
	}
	if bytes.Compare(c.c0c1[9:c0c1Len], s0s1s2[c0c1Len+8:2*c0c1Len-1]) != 0 {
		return rtmpErr
	}
	c.c2 = append(c.c2, s0s1s2[1:1+c2Len]...)

	return nil
}

func (c *HandshakeClient) writeC2(write io.Writer) error {
	_, err := write.Write(c.c2)
	log.Infof("<----- Handshake C2")
	return err
}
