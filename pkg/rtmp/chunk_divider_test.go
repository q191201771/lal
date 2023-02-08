// Copyright 2023, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/assert"
	"github.com/q191201771/naza/pkg/bele"
	"testing"
)

func TestMessage2Chunks(t *testing.T) {
	capa := LocalChunkSize * 10
	goldBuf := make([]byte, capa)
	for i := 0; i < capa; i++ {
		goldBuf[i] = byte(i % 256)
	}

	var packFn = func(testLen uint32) []byte {
		h := base.RtmpHeader{
			Csid:         CsidVideo,
			MsgLen:       testLen,
			MsgTypeId:    base.RtmpTypeIdVideo,
			MsgStreamId:  Msid1,
			TimestampAbs: 123,
		}
		return Message2Chunks(goldBuf[:testLen], &h)
	}

	// 07    00 00 7b   00 00 01  09      01 00 00 00  00
	// csid  timestamp  len       typeid  streamid     v

	goldHeaderHex := []byte{7, 0, 0, 0x7b, 0, 0, 1, 9, 1, 0, 0, 0}
	testLenArray1 := []int{1, 2, 4095, 4096}
	for _, testLen := range testLenArray1 {
		m := packFn(uint32(testLen))
		bele.BePutUint24(goldHeaderHex[4:], uint32(testLen))
		assert.Equal(t, append(goldHeaderHex, goldBuf[:testLen]...), m)
	}

	goldHeaderHex21 := []byte{7, 0, 0, 0x7b, 0, 0, 1, 9, 1, 0, 0, 0}
	goldHeaderHex22 := []byte{0xc7} // c fmt=3, 7 csid
	testLenArray2 := []int{4097, 4098, 8191, 8192}
	for _, testLen := range testLenArray2 {
		m := packFn(uint32(testLen))
		bele.BePutUint24(goldHeaderHex21[4:], uint32(testLen))
		exp := append(goldHeaderHex21, goldBuf[:4096]...)
		exp = append(exp, goldHeaderHex22...)
		exp = append(exp, goldBuf[4096:testLen]...)
		assert.Equal(t, exp, m)
	}

	goldHeaderHex31 := []byte{7, 0, 0, 0x7b, 0, 0, 1, 9, 1, 0, 0, 0}
	goldHeaderHex32 := []byte{0xc7} // c fmt=3, 7 csid
	goldHeaderHex33 := []byte{0xc7} // c fmt=3, 7 csid
	testLenArray3 := []int{8193, 8194, 4096 * 3}
	for _, testLen := range testLenArray3 {
		m := packFn(uint32(testLen))
		bele.BePutUint24(goldHeaderHex31[4:], uint32(testLen))
		exp := append(goldHeaderHex31, goldBuf[:4096]...)
		exp = append(exp, goldHeaderHex32...)
		exp = append(exp, goldBuf[4096:8192]...)
		exp = append(exp, goldHeaderHex33...)
		exp = append(exp, goldBuf[8192:testLen]...)
		assert.Equal(t, exp, m)
	}
}
