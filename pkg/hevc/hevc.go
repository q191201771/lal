// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hevc

var NaluUintTypeMapping = map[uint8]string{
	NaluUnitTypeSliceTrailR: "SLICE",
	NaluUnitTypeSliceIDR:    "I",
	NaluUintTypeSliceIDRNLP: "IDR",
	NaluUnitTypeSEI:         "SEI",
	NaluUnitTypeSEISuffix:   "SEI",
}
var (
	NaluUnitTypeSliceTrailR uint8 = 1  // 0x01
	NaluUnitTypeSliceIDR    uint8 = 19 // 0x13
	NaluUintTypeSliceIDRNLP uint8 = 20 // 0x14
	NaluUnitTypeVPS         uint8 = 32 // 0x20
	NaluUnitTypeSPS         uint8 = 33 // 0x21
	NaluUnitTypePPS         uint8 = 34 // 0x22
	NaluUnitTypeSEI         uint8 = 39 // 0x27
	NaluUnitTypeSEISuffix   uint8 = 40 // 0x28
)

func CalcNaluTypeReadable(nalu []byte) string {
	b, ok := NaluUintTypeMapping[CalcNaluType(nalu)]
	if !ok {
		return "unknown"
	}
	return b
}

func CalcNaluType(nalu []byte) uint8 {
	return (nalu[0] & 0x7E) >> 1
}
