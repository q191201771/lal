// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package stun

import (
	"crypto/rand"
	"fmt"

	"github.com/q191201771/naza/pkg/bele"
)

type Header struct {
	Typ           int
	Length        int
	MagicCookie   int
	TransactionID []byte
}

func PackHeaderTo(out []byte, typ int, length int) error {
	if len(out) < minStunMessageSize {
		return ErrStun
	}
	bele.BEPutUint16(out, uint16(typ))
	bele.BEPutUint16(out[2:], uint16(length))
	bele.BEPutUint32(out[4:], uint32(magicCookieBE))
	_, err := rand.Reader.Read(out[8:])
	return err
}

func PackBindingRequest() ([]byte, error) {
	b := make([]byte, minStunMessageSize)
	err := PackHeaderTo(b, typeBindingRequestBE, 0)
	return b, err
}

func PackBindingResponse(ip []byte, port int) ([]byte, error) {
	b := make([]byte, minStunMessageSize+4+attrTypeXORMappedAddressSize)
	err := PackHeaderTo(b, typeBindSuccessResponseBE, 4+attrTypeXORMappedAddressSize)
	if err != nil {
		return nil, err
	}

	bele.BEPutUint16(b[20:], uint16(attrTypeXORMappedAddressBE))
	bele.BEPutUint16(b[22:], attrTypeXORMappedAddressSize)
	packAttrXORMappedAddressTo(b[24:], ip, port)
	return b, nil
}

// @param out 输出参数，需保证len(b)>=8
//
// @return ip 4字节格式
func packAttrXORMappedAddressTo(out []byte, ip []byte, port int) {
	bele.BEPutUint32(out, uint32(protocolFamilyIPv4BE))
	bele.BEPutUint16(out[2:], uint16(port^(magicCookieBE>>16)))
	xor(ip, magicCookie, out[4:])
	return
}

//func unpackAttrXORMappedAddress(b []byte) (ip string, port int, err error) {
//	if bytes.Compare(b[:2], protocolFamilyIPv4) != 0 {
//		return "", 0, ErrStun
//	}
//
//	port = int(bele.BEUint16(b[2:])) ^ (magicCookieBE >> 16)
//
//	ipb := make([]byte, 4)
//	xor(b[4:], magicCookie, ipb)
//	ip = fmt.Sprintf("%d.%d.%d.%d", ipb[0], ipb[1], ipb[2], ipb[3])
//	return
//}

// ---------------------------------------------------------------------------

func UnpackHeader(b []byte) (h Header, err error) {
	if len(b) < minStunMessageSize {
		return h, ErrStun
	}
	h.Typ = int(bele.BEUint16(b[:2]))
	h.Length = int(bele.BEUint16(b[2:]))
	h.MagicCookie = int(bele.BEUint32(b[4:]))
	h.TransactionID = b[12:20]
	return
}

func UnpackResponseMessage(b []byte) (ip string, port int, err error) {
	h, err := UnpackHeader(b)
	if err != nil {
		return "", 0, err
	}

	// TODO chef: only impled bind success response
	if h.Typ != typeBindSuccessResponseBE {
		return "", 0, ErrStun
	}

	if h.MagicCookie != magicCookieBE {
		return "", 0, ErrStun
	}

	if len(b) < minStunMessageSize+h.Length {
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
			ip, port, err = unpackAttrXORMappedAddress(b[pos:])
			if err != nil {
				return "", 0, err
			}
		}
		if at == attrTypeMappedAddressBE {
			ip, port, err = unpackAttrMappedAddress(b[pos:])
			if err != nil {
				return "", 0, err
			}
		}

		pos += al
		if pos == minStunMessageSize+h.Length {
			break
		}
	}

	return ip, port, nil
}

func unpackAttrXORMappedAddress(b []byte) (ip string, port int, err error) {
	if int(bele.BEUint16(b[:2])) != protocolFamilyIPv4BE {
		return "", 0, ErrStun
	}

	port = int(bele.BEUint16(b[2:])) ^ (magicCookieBE >> 16)

	ipb := make([]byte, 4)
	xor(b[4:], magicCookie, ipb)
	ip = fmt.Sprintf("%d.%d.%d.%d", ipb[0], ipb[1], ipb[2], ipb[3])
	return
}

func unpackAttrMappedAddress(b []byte) (ip string, port int, err error) {
	if int(bele.BEUint16(b[:2])) != protocolFamilyIPv4BE {
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
