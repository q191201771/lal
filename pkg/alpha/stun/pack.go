// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package stun

import (
	"bytes"
	"crypto/rand"
	"fmt"

	"github.com/q191201771/naza/pkg/bele"
)

// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |0 0|     STUN Message Type     |         Message Length        |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                         Magic Cookie                          |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                                                               |
// |                     Transaction ID (96 bits)                  |
// |                                                               |
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//
// Figure 2: Format of STUN Message Header

const minStunMessageSize = 20

var (
	magicCookie   = []byte{0x21, 0x12, 0xa4, 0x42}
	magicCookieBE = 0x2112a442

	//typeBindSuccessResponse = []byte{0x1, 0x1}
	typeBindingRequest        = []byte{0x0, 0x1}
	typeBindSuccessResponseBE = 0x0101

	//attrTypeXORMappedAddress  = []byte{0x0, 0x20}
	//attrTypeXORMappedAddress2 = []byte{0x80, 0x20}
	//attrTypeMappedAddress     = []byte{0x0, 0x1}
	attrTypeXORMappedAddressBE  = 0x0020
	attrTypeXORMappedAddress2BE = 0x8020
	attrTypeMappedAddressBE     = 0x0001

	protocolFamilyIPv4 = []byte{0x0, 0x1}
)

func PackBindingRequest() ([]byte, error) {
	b := make([]byte, minStunMessageSize)
	copy(b, typeBindingRequest)
	// b[2:4] message length 0
	copy(b[4:], magicCookie)
	// transaction id
	if _, err := rand.Reader.Read(b[8:]); err != nil {
		return nil, err
	}
	return b, nil
}

func ParseMessage(b []byte) (ip string, port int, err error) {
	if len(b) < minStunMessageSize {
		return "", 0, ErrStun
	}
	// TODO chef: only impled bind success response
	if int(bele.BEUint16(b[:2])) != typeBindSuccessResponseBE {
		return "", 0, ErrStun
	}

	messageLength := bele.BEUint16(b[2:])

	if bytes.Compare(b[4:8], magicCookie) != 0 {
		return "", 0, ErrStun
	}

	// transaction id

	if len(b) < minStunMessageSize+int(messageLength) {
		return "", 0, ErrStun
	}

	// attr list
	pos := minStunMessageSize
	for {
		if len(b[pos:]) < 4 {
			return "", 0, ErrStun
		}
		at := int(bele.BEUint16(b[pos : pos+2]))
		al := int(bele.BEUint16(b[pos+2 : pos+4]))
		pos += 4
		if len(b[pos:]) < al {
			return "", 0, ErrStun
		}

		if at == attrTypeXORMappedAddressBE || at == attrTypeXORMappedAddress2BE {
			ip, port, err = parseAttrXORMappedAddress(b[pos:])
			if err != nil {
				return "", 0, err
			}
		}
		if at == attrTypeMappedAddressBE {
			ip, port, err = parseAttrMappedAddress(b[pos:])
			if err != nil {
				return "", 0, err
			}
		}

		pos += al
		if pos == minStunMessageSize+int(messageLength) {
			break
		}
	}

	return ip, port, nil
}

func parseAttrXORMappedAddress(b []byte) (ip string, port int, err error) {
	if bytes.Compare(b[:2], protocolFamilyIPv4) != 0 {
		return "", 0, ErrStun
	}

	port = int(bele.BEUint16(b[2:])) ^ (magicCookieBE >> 16)

	ipb := make([]byte, 4)
	xor(b[4:], magicCookie, ipb)
	ip = fmt.Sprintf("%d.%d.%d.%d", ipb[0], ipb[1], ipb[2], ipb[3])
	return
}

func parseAttrMappedAddress(b []byte) (ip string, port int, err error) {
	if bytes.Compare(b[:2], protocolFamilyIPv4) != 0 {
		return "", 0, ErrStun
	}

	port = int(bele.BEUint16(b[2:]))
	ip = fmt.Sprintf("%d.%d.%d.%d", b[4], b[5], b[6], b[7])
	return
}

func xor(a, b, res []byte) {
	n := len(a)
	if n > len(b) {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		res[i] = a[i] ^ b[i]
	}
}
