package base

import (
	"crypto/sha1"
	"encoding/base64"

	"github.com/q191201771/naza/pkg/bele"
)

/*
The WebSocket Protocol
https://tools.ietf.org/html/rfc6455

0                   1                   2                   3
0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-------+-+-------------+-------------------------------+
|F|R|R|R| opcode|M| Payload len |    Extended payload length    |
|I|S|S|S|  (4)  |A|     (7)     |             (16/64)           |
|N|V|V|V|       |S|             |   (if payload len==126/127)   |
| |1|2|3|       |K|             |                               |
+-+-+-+-+-------+-+-------------+ - - - - - - - - - - - - - - - +
|     Extended payload length continued, if payload len == 127  |
+ - - - - - - - - - - - - - - - +-------------------------------+
|                               |Masking-key, if MASK set to 1  |
+-------------------------------+-------------------------------+
| Masking-key (continued)       |          Payload Data         |
+-------------------------------- - - - - - - - - - - - - - - - +
:                     Payload Data continued ...                :
+ - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - +
|                     Payload Data continued ...                |
+---------------------------------------------------------------+
opcode:
*  %x0 denotes a continuation frame
*  %x1 denotes a text frame
*  %x2 denotes a binary frame
*  %x3-7 are reserved for further non-control frames
*  %x8 denotes a connection close
*  %x9 denotes a ping
*  %xA denotes a pong
*  %xB-F are reserved for further control frames
Payload length:  7 bits, 7+16 bits, or 7+64 bits
Masking-key:  0 or 4 bytes
*/
const WS_MAGIC_STR = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func MakeWSFrameHeader(AOpCode uint8, AFin bool, AMaskKey uint32, ADataSize uint64) (HeaderBytes []byte) {
	LHeaderSize := 2
	LPayload := uint64(0)
	if ADataSize < 126 {
		LPayload = ADataSize
	} else if ADataSize <= 0xFFFF {
		LPayload = 126
		LHeaderSize += 2
	} else {
		LPayload = 127
		LHeaderSize += 8
	}
	if AMaskKey != 0 {
		LHeaderSize += 4
	}
	HeaderBytes = make([]byte, LHeaderSize, LHeaderSize)
	if AFin {
		HeaderBytes[0] = HeaderBytes[0] | 0x80
	}

	HeaderBytes[0] = HeaderBytes[0] | (AOpCode & 0x0F)

	if AMaskKey != 0 {
		HeaderBytes[1] = HeaderBytes[1] | 0x80
	}
	HeaderBytes[1] = HeaderBytes[1] | (uint8(LPayload) & 0x7F)
	if LPayload == 126 {
		bele.BEPutUint16(HeaderBytes[2:4], uint16(ADataSize))
	} else if LPayload == 127 {
		bele.BEPutUint64(HeaderBytes[2:10], ADataSize)
	}

	if AMaskKey != 0 {
		bele.LEPutUint32(HeaderBytes[LHeaderSize-4:], AMaskKey)
	}
	return HeaderBytes
}
func UpdateWebSocketHeader(secWebSocketKey string) []byte {
	firstLine := "HTTP/1.1 101 Switching Protocol\r\n"
	sha1Sum := sha1.Sum([]byte(secWebSocketKey + WS_MAGIC_STR))
	secWebSocketAccept := base64.StdEncoding.EncodeToString(sha1Sum[:])
	webSocketResponseHeaderStr := firstLine +
		"Server: " + LALHTTPFLVSubSessionServer + "\r\n" +
		"Sec-WebSocket-Accept:" + secWebSocketAccept + "\r\n" +
		"Keep-Alive: timeout=15, max=100\r\n" +
		"Connection: Upgrade\r\n" +
		"Upgrade: websocket\r\n" +
		"Access-Control-Allow-Credentials: true\r\n" +
		"Access-Control-Allow-Origin: *\r\n" +
		"\r\n"
	return []byte(webSocketResponseHeaderStr)
}
