// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hevc

import (
	"encoding/hex"
	"errors"

	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazalog"
)

// AnnexB
//
// HVCC

var ErrHEVC = errors.New("lal.hevc: fxxk")

var (
	NALUStartCode4 = []byte{0x0, 0x0, 0x0, 0x1}
)

var NALUTypeMapping = map[uint8]string{
	NALUTypeSliceTrailR: "SLICE",
	NALUTypeSliceIDR:    "I",
	NALUTypeSliceIDRNLP: "IDR",
	NALUTypeSEI:         "SEI",
	NALUTypeSEISuffix:   "SEI",
}
var (
	NALUTypeSliceTrailR uint8 = 1  // 0x01
	NALUTypeSliceIDR    uint8 = 19 // 0x13
	NALUTypeSliceIDRNLP uint8 = 20 // 0x14
	NALUTypeVPS         uint8 = 32 // 0x20
	NALUTypeSPS         uint8 = 33 // 0x21
	NALUTypePPS         uint8 = 34 // 0x22
	NALUTypeSEI         uint8 = 39 // 0x27
	NALUTypeSEISuffix   uint8 = 40 // 0x28
)

func ParseNALUTypeReadable(v uint8) string {
	b, ok := NALUTypeMapping[ParseNALUType(v)]
	if !ok {
		return "unknown"
	}
	return b
}

func ParseNALUType(v uint8) uint8 {
	// 6 bit in middle
	// 0*** ***0
	// or return (nalu[0] >> 1) & 0x3F
	return (v & 0x7E) >> 1
}

// HVCC Seq Header -> AnnexB
// 注意，返回的内存块为独立的内存块，不依赖指向传输参数<payload>内存块
//
func VPSSPSPPSSeqHeader2AnnexB(payload []byte) ([]byte, error) {
	vps, sps, pps, err := ParseVPSSPSPPSFromSeqHeader(payload)
	if err != nil {
		return nil, ErrHEVC
	}
	var ret []byte
	ret = append(ret, NALUStartCode4...)
	ret = append(ret, vps...)
	ret = append(ret, NALUStartCode4...)
	ret = append(ret, sps...)
	ret = append(ret, NALUStartCode4...)
	ret = append(ret, pps...)
	return ret, nil
}

// 从HVCC格式的Seq Header中得到VPS，SPS，PPS内存块
//
// @param <payload> rtmp message的payload部分或者flv tag的payload部分
//                  注意，包含了头部2字节类型以及3字节的cts
//
// @return 注意，返回的vps，sps，pps内存块指向的是传入参数<payload>内存块的内存
//
func ParseVPSSPSPPSFromSeqHeader(payload []byte) (vps, sps, pps []byte, err error) {
	if len(payload) < 5 {
		return nil, nil, nil, ErrHEVC
	}

	if payload[0] != 0x1c || payload[1] != 0x00 || payload[2] != 0 || payload[3] != 0 || payload[4] != 0 {
		return nil, nil, nil, ErrHEVC
	}
	nazalog.Debugf("%s", hex.Dump(payload))

	if len(payload) < 33 {
		return nil, nil, nil, ErrHEVC
	}

	index := 27
	if numOfArrays := payload[index]; numOfArrays != 3 {
		return nil, nil, nil, ErrHEVC
	}
	index++

	if payload[index] != NALUTypeVPS&0x3f {
		return nil, nil, nil, ErrHEVC
	}
	if numNalus := int(bele.BEUint16(payload[index+1:])); numNalus != 1 {
		return nil, nil, nil, ErrHEVC
	}
	vpsLen := int(bele.BEUint16(payload[index+3:]))

	if len(payload) < 33+vpsLen {
		return nil, nil, nil, ErrHEVC
	}

	vps = payload[index+5 : index+5+vpsLen]
	index += 5 + vpsLen

	if len(payload) < 38+vpsLen {
		return nil, nil, nil, ErrHEVC
	}
	if payload[index] != NALUTypeSPS&0x3f {
		return nil, nil, nil, ErrHEVC
	}
	if numNalus := int(bele.BEUint16(payload[index+1:])); numNalus != 1 {
		return nil, nil, nil, ErrHEVC
	}
	spsLen := int(bele.BEUint16(payload[index+3:]))
	if len(payload) < 38+vpsLen+spsLen {
		return nil, nil, nil, ErrHEVC
	}
	sps = payload[index+5 : index+5+spsLen]
	index += 5 + spsLen

	if len(payload) < 43+vpsLen+spsLen {
		return nil, nil, nil, ErrHEVC
	}
	if payload[index] != NALUTypePPS&0x3f {
		return nil, nil, nil, ErrHEVC
	}
	if numNalus := int(bele.BEUint16(payload[index+1:])); numNalus != 1 {
		return nil, nil, nil, ErrHEVC
	}
	ppsLen := int(bele.BEUint16(payload[index+3:]))
	if len(payload) < 43+vpsLen+spsLen+ppsLen {
		return nil, nil, nil, ErrHEVC
	}
	pps = payload[index+5 : index+5+ppsLen]

	return
}
