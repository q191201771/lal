// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package avc

import (
	"errors"
	"io"
	"math"

	"github.com/q191201771/naza/pkg/nazabits"

	"github.com/q191201771/naza/pkg/bele"
)

var ErrAVC = errors.New("lal.avc: fxxk")

var NaluStartCode = []byte{0x0, 0x0, 0x0, 0x1}

var NaluUintTypeMapping = map[uint8]string{
	1: "SLICE",
	5: "IDR",
	6: "SEI",
	7: "SPS",
	8: "PPS",
	9: "AUD",
}

var SliceTypeMapping = map[uint8]string{
	0: "P",
	1: "B",
	2: "I",
	3: "SP",
	4: "SI",
	5: "P",
	6: "B",
	7: "I",
	8: "SP",
	9: "SI",
}

const (
	NaluUnitTypeSlice    uint8 = 1
	NaluUnitTypeIDRSlice uint8 = 5
	NaluUnitTypeSEI      uint8 = 6
	NaluUintTypeSPS      uint8 = 7
	NaluUintTypePPS      uint8 = 8
	NaluUintTypeAUD      uint8 = 9 // TODO chef
)

const (
	SliceTypeP  uint8 = 0
	SliceTypeB  uint8 = 1
	SliceTypeI  uint8 = 2
	SliceTypeSP uint8 = 3 // TODO chef
	SliceTypeSI uint8 = 4 // TODO chef
)

func CalcSliceType(nalu []byte) uint8 {
	c := nalu[1]
	var leadingZeroBits int
	index := 6
	for ; index >= 0; index-- {
		v := nazabits.GetBit8(c, index)
		if v == 0 {
			leadingZeroBits++
		} else {
			break
		}
	}
	rbLeadingZeroBits := nazabits.GetBits8(c, index-1, leadingZeroBits)
	codeNum := int(math.Pow(2, float64(leadingZeroBits))) - 1 + rbLeadingZeroBits
	if codeNum > 4 {
		codeNum -= 5
	}
	return uint8(codeNum)
}

func CalcSliceTypeReadable(nalu []byte) string {
	t := CalcSliceType(nalu)
	ret, ok := SliceTypeMapping[t]
	if !ok {
		return "unknown"
	}
	return ret
}

func CalcNaluType(nalu []byte) uint8 {
	return nalu[0] & 0x1f
}

func CalcNaluTypeReadable(nalu []byte) string {
	t := nalu[0] & 0x1f
	ret, ok := NaluUintTypeMapping[t]
	if !ok {
		return "unknown"
	}
	return ret
}

// 从 rtmp avc sequence header 中解析 sps 和 pps
// @param <payload> rtmp message的payload部分 或者 flv tag的payload部分
func ParseAVCSeqHeader(payload []byte) (sps, pps []byte, err error) {
	// TODO chef: check if read out of <payload> range

	if payload[0] != 0x17 || payload[1] != 0x00 || payload[2] != 0 || payload[3] != 0 || payload[4] != 0 {
		err = ErrAVC
		return
	}

	// H.264-AVC-ISO_IEC_14496-15.pdf
	// 5.2.4 Decoder configuration information

	//configurationVersion := payload[5]
	//avcProfileIndication := payload[6]
	//profileCompatibility := payload[7]
	//avcLevelIndication := payload[8]
	//lengthSizeMinusOne := payload[9] & 0x03

	index := 10

	numOfSPS := int(payload[index] & 0x1F)
	index++
	// TODO chef: if the situation of multi sps exist?
	// only take the last one.
	for i := 0; i < numOfSPS; i++ {
		lenOfSPS := int(bele.BEUint16(payload[index:]))
		index += 2
		sps = append(sps, payload[index:index+lenOfSPS]...)
		index += lenOfSPS
	}

	numOfPPS := int(payload[index] & 0x1F)
	index++
	for i := 0; i < numOfPPS; i++ {
		lenOfPPS := int(bele.BEUint16(payload[index:]))
		index += 2
		pps = append(pps, payload[index:index+lenOfPPS]...)
		index += lenOfPPS
	}

	return
}

// 将rtmp avc数据转换成avc裸流
// @param <payload> rtmp message的payload部分 或者 flv tag的payload部分
func CaptureAVC(w io.Writer, payload []byte) error {
	// sps pps
	if payload[0] == 0x17 && payload[1] == 0x00 {
		sps, pps, err := ParseAVCSeqHeader(payload)
		if err != nil {
			return err
		}
		//utilErrors.PanicIfErrorOccur(err)
		_, _ = w.Write(NaluStartCode)
		_, _ = w.Write(sps)
		_, _ = w.Write(NaluStartCode)
		_, _ = w.Write(pps)
		return nil
	}

	// payload中可能存在多个nalu
	// 先跳过前面type的2字节，以及composition time的3字节
	for i := 5; i != len(payload); {
		naluLen := int(bele.BEUint32(payload[i:]))
		i += 4
		//naluUintType := payload[i] & 0x1f
		//log.Debugf("naluLen:%d t:%d %s\n", naluLen, naluUintType, avc.NaluUintTypeMapping[naluUintType])
		_, _ = w.Write(NaluStartCode)
		_, _ = w.Write(payload[i : i+naluLen])
		i += naluLen
		break
	}
	return nil
}
