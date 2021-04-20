// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package stun

import "errors"

// TODO chef:
// - attr soft

// Session Traversal Utilities for NAT
//
// rfc 5389
//

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

var ErrStun = errors.New("lal.stun: fxxk")

var DefaultPort = 3478

const (
	minStunMessageSize = 20

	attrTypeXORMappedAddressSize = 8
)

var (
	magicCookie = []byte{0x21, 0x12, 0xa4, 0x42}
)

const (
	magicCookieBE = 0x2112a442

	typeBindingRequestBE      = 0x0001
	typeBindSuccessResponseBE = 0x0101

	attrTypeXORMappedAddressBE  = 0x0020
	attrTypeXORMappedAddress2BE = 0x8020
	attrTypeMappedAddressBE     = 0x0001

	protocolFamilyIPv4BE = 0x0001
)
