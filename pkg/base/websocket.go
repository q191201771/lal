// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: joestarzxh

package base

import (
	"crypto/sha1"
	"encoding/base64"
	"math"

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
mark 加密
for i := 0; i < datalen; i {
    m := markingkeys[i%4]
    data[i] = msg[i] ^ m
}
*/
type WsOpcode = uint8

const (
	WSO_Continuous WsOpcode = iota //连续消息片断
	WSO_Text                       //文本消息片断,
	WSO_Binary                     //二进制消息片断,
	//非控制消息片断保留的操作码,
	WSO_Rsv3
	WSO_Rsv4
	WSO_Rsv5
	WSO_Rsv6
	WSO_Rsv7
	WSO_Close //连接关闭,
	WSO_Ping  //心跳检查的ping,
	WSO_Pong  //心跳检查的pong,
	//为将来的控制消息片断的保留操作码
	WSO_RsvB
	WSO_RsvC
	WSO_RsvD
	WSO_RsvE
	WSO_RsvF
)

type WSHeader struct {
	Fin    bool
	Rsv1   bool
	Rsv2   bool
	Rsv3   bool
	Opcode WsOpcode

	PayloadLength uint64

	Masked  bool
	MaskKey uint32
}

const WS_MAGIC_STR = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

func MakeWSFrameHeader(wsHeader WSHeader) (buf []byte) {
	headerSize := 2
	payload := uint64(0)
	switch {
	case wsHeader.PayloadLength < 126:
		payload = wsHeader.PayloadLength
	case wsHeader.PayloadLength <= math.MaxUint16:
		payload = 126
		headerSize += 2
	case wsHeader.PayloadLength > math.MaxUint16:
		payload = 127
		headerSize += 8
	}
	if wsHeader.Masked {
		headerSize += 4
	}
	buf = make([]byte, headerSize, headerSize)
	if wsHeader.Fin {
		buf[0] |= 1 << 7
	}
	if wsHeader.Rsv1 {
		buf[0] |= 1 << 6
	}
	if wsHeader.Rsv2 {
		buf[0] |= 1 << 5
	}
	if wsHeader.Rsv3 {
		buf[0] |= 1 << 4
	}
	buf[0] |= wsHeader.Opcode

	if wsHeader.Masked {
		buf[1] |= 1 << 7
	}
	buf[1] |= (uint8(payload) & 0x7F)
	if payload == 126 {
		bele.BEPutUint16(buf[2:], uint16(wsHeader.PayloadLength))
	} else if payload == 127 {
		bele.BEPutUint64(buf[2:], wsHeader.PayloadLength)
	}

	if wsHeader.Masked {
		bele.LEPutUint32(buf[headerSize-4:], wsHeader.MaskKey)
	}
	return buf
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
