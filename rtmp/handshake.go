package rtmp

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"github.com/q191201771/lal/bele"
	"github.com/q191201771/lal/log"
	"io"
	"time"
)

// rtmp握手分为两种模式：简单模式和复杂模式
// 本源码文件中
// HandshakeClient作为客户端握手，只实现了简单模式
// HandshakeServer作为服务端握手，实现了简单模式和复杂模式

// TODO chef: HandshakeClient with complex mode.

var version = uint8(3)

//var c1Len = 1536
var c0c1Len = 1537
var s0s1Len = 1537
var c2Len = 1536
var s2Len = 1536
var s0s1s2Len = 3073

var serverVersion = []byte{
	0x0D, 0x0E, 0x0A, 0x0D,
}

// 30+32
var clientKey = []byte{
	'G', 'e', 'n', 'u', 'i', 'n', 'e', ' ', 'A', 'd', 'o', 'b', 'e', ' ',
	'F', 'l', 'a', 's', 'h', ' ', 'P', 'l', 'a', 'y', 'e', 'r', ' ',
	'0', '0', '1',

	0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8, 0x2E, 0x00, 0xD0, 0xD1,
	0x02, 0x9E, 0x7E, 0x57, 0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
	0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
}

// 36+32
var serverKey = []byte{
	'G', 'e', 'n', 'u', 'i', 'n', 'e', ' ', 'A', 'd', 'o', 'b', 'e', ' ',
	'F', 'l', 'a', 's', 'h', ' ', 'M', 'e', 'd', 'i', 'a', ' ',
	'S', 'e', 'r', 'v', 'e', 'r', ' ',
	'0', '0', '1',

	0xF0, 0xEE, 0xC2, 0x4A, 0x80, 0x68, 0xBE, 0xE8, 0x2E, 0x00, 0xD0, 0xD1,
	0x02, 0x9E, 0x7E, 0x57, 0x6E, 0xEC, 0x5D, 0x2D, 0x29, 0x80, 0x6F, 0xAB,
	0x93, 0xB8, 0xE6, 0x36, 0xCF, 0xEB, 0x31, 0xAE,
}

var clientPartKeyLen = 30
var serverPartKeyLen = 36
var serverFullKeyLen = 68
var keyLen = 32

type HandshakeClient struct {
	c0c1 []byte
	c2   []byte
}

type HandshakeServer struct {
	isSimpleMode bool
	s0s1s2       []byte
}

func (c *HandshakeClient) WriteC0C1(writer io.Writer) error {
	c.c0c1 = make([]byte, c0c1Len)
	c.c0c1[0] = version
	bele.BEPutUint32(c.c0c1[1:5], uint32(time.Now().Unix()))

	// TODO chef: random [9:]

	_, err := writer.Write(c.c0c1)
	log.Infof("<----- Handshake C0+C1")
	return err
}

func (c *HandshakeClient) ReadS0S1S2(reader io.Reader) error {
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

func (c *HandshakeClient) WriteC2(write io.Writer) error {
	_, err := write.Write(c.c2)
	log.Infof("<----- Handshake C2")
	return err
}

func (s *HandshakeServer) ReadC0C1(reader io.Reader) error {
	c0c1 := make([]byte, c0c1Len)
	if _, err := io.ReadAtLeast(reader, c0c1, c0c1Len); err != nil {
		return err
	}
	log.Infof("-----> Handshake C0+C1")

	s.s0s1s2 = make([]byte, s0s1s2Len)

	if err := s.parseChallenge(c0c1); err != nil {
		return err
	}

	var readC1Timestamp uint32
	if s.isSimpleMode {
		readC1Timestamp = bele.BEUint32(c0c1[1:])
	}

	s.s0s1s2[0] = version

	s1 := s.s0s1s2[1:]
	writeS1Timestamp := uint32(time.Now().Unix())
	bele.BEPutUint32(s1, writeS1Timestamp)

	if s.isSimpleMode {
		bele.BEPutUint32(s1[4:], 0)
		// TODO chef: random s1 1528

		// s2
		bele.BEPutUint32(s.s0s1s2[s0s1Len:], readC1Timestamp)
		bele.BEPutUint32(s.s0s1s2[s0s1Len+4:], writeS1Timestamp)
		// TODO chef: random

	} else {
		copy(s1[4:], serverVersion)
		// TODO chef: random s1

		offs := int(s1[8]) + int(s.s0s1s2[9]) + int(s1[10]) + int(s1[11])
		offs = (offs % 728) + 12
		makeDigestWithoutCenterKey(s1, offs, serverKey[:serverPartKeyLen], s1[offs:])
	}

	return nil
}

func (s *HandshakeServer) WriteS0S1S2(write io.Writer) error {
	_, err := write.Write(s.s0s1s2)
	log.Infof("<----- Handshake S0S1S2")
	return err
}

func (s *HandshakeServer) ReadC2(reader io.Reader) error {
	c2 := make([]byte, c2Len)
	if _, err := io.ReadAtLeast(reader, c2, c2Len); err != nil {
		return err
	}
	log.Infof("-----> Handshake C2")
	return nil
}

func (s *HandshakeServer) parseChallenge(c0c1 []byte) error {
	if c0c1[0] != version {
		return rtmpErr
	}
	//peerEpoch := bele.BEUint32(c0c1[1:])
	ver := bele.BEUint32(c0c1[5:])
	if ver == 0 {
		log.Debug("handshake simple mode.")
		s.isSimpleMode = true
		return nil
	}

	// assume digest in second-half
	// | peer epoch | ver |      | digest |
	// | 4          | 4   | 764  | 764    |
	// offs [0~727] + 764 + 8 + 4
	//      [776~1503]
	offs := findDigest(c0c1[1:], 764+8, clientKey[:clientPartKeyLen])
	if offs == -1 {
		// try digest in first-half
		// | peer epoch | ver | digest |    |
		// offs [0~727] + 8 + 4
		//      [12~739]
		offs = findDigest(c0c1[1:], 8, clientKey[:clientPartKeyLen])
	}
	if offs == -1 {
		log.Warn("get digest offs failed. roll back to try simple handshake.")
		s.isSimpleMode = true
		return nil
	}

	s.isSimpleMode = false

	// use c0c1 digest to make a new digest
	digest := makeDigest(c0c1[1+offs:1+offs+keyLen], serverKey[:serverFullKeyLen])

	// hardcode reply offs
	replyOffs := s2Len - keyLen

	// use new digest as key to make another digest.
	digest2 := make([]byte, keyLen)
	makeDigestWithoutCenterKey(s.s0s1s2[s0s1Len:], replyOffs, digest, digest2)

	// copy digest2 to s2
	copy(s.s0s1s2[s0s1Len+replyOffs:], digest2)

	return nil
}

// <b> could be `c1`
func findDigest(b []byte, base int, key []byte) int {
	// calc offs
	offs := int(b[base]) + int(b[base+1]) + int(b[base+2]) + int(b[base+3])
	offs = (offs % 728) + base + 4
	// calc digest
	digest := make([]byte, keyLen)
	makeDigestWithoutCenterKey(b, offs, key, digest)
	// compare origin digest in buffer with calced digest
	if bytes.Compare(digest, b[offs:offs+keyLen]) == 0 {
		return offs
	}
	return -1
}

// <b> could be `c1` or `s2`
func makeDigestWithoutCenterKey(b []byte, offs int, key []byte, out []byte) {
	mac := hmac.New(sha256.New, key)
	// left
	if offs != 0 {
		mac.Write(b[:offs])
	}

	// right
	if len(b)-offs-keyLen > 0 {
		mac.Write(b[offs+keyLen:])
	}
	copy(out, mac.Sum(nil))
}

func makeDigest(b []byte, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(b)
	return mac.Sum(nil)
}
