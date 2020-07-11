// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hevc

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

func CalcNALUTypeReadable(nalu []byte) string {
	b, ok := NALUTypeMapping[CalcNALUType(nalu)]
	if !ok {
		return "unknown"
	}
	return b
}

func CalcNALUType(nalu []byte) uint8 {
	// 6 bit in middle
	// 0*** ***0
	// or return (nalu[0] >> 1) & 0x3F
	return (nalu[0] & 0x7E) >> 1
}
