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

	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazabits"
)

var ErrAVC = errors.New("lal.avc: fxxk")

var NALUStartCode = []byte{0x0, 0x0, 0x0, 0x1}

var NALUTypeMapping = map[uint8]string{
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
	NALUTypeSlice    uint8 = 1
	NALUTypeIDRSlice uint8 = 5
	NALUTypeSEI      uint8 = 6
	NALUTypeSPS      uint8 = 7
	NALUTypePPS      uint8 = 8
	NALUTypeAUD      uint8 = 9
)

const (
	SliceTypeP  uint8 = 0
	SliceTypeB  uint8 = 1
	SliceTypeI  uint8 = 2
	SliceTypeSP uint8 = 3 // TODO chef
	SliceTypeSI uint8 = 4 // TODO chef
)

func CalcNALUType(nalu []byte) uint8 {
	return nalu[0] & 0x1f
}

// TODO chef: 考虑将error返回给上层
func CalcSliceType(nalu []byte) uint8 {
	br := nazabits.NewBitReader(nalu[1:])
	// first_mb_in_slice
	_, err := br.ReadGolomb()
	if err != nil {
		return 0
	}
	sliceType, err := br.ReadGolomb()
	if err != nil {
		return 0
	}
	// TODO chef: 检查非法数据，slice type范围 [0, 9]
	if sliceType > 4 {
		sliceType -= 5
	}
	return uint8(sliceType)
}

func CalcNALUTypeReadable(nalu []byte) string {
	t := nalu[0] & 0x1f
	ret, ok := NALUTypeMapping[t]
	if !ok {
		return "unknown"
	}
	return ret
}

func CalcSliceTypeReadable(nalu []byte) string {
	naluType := CalcNALUType(nalu)
	switch naluType {
	case NALUTypeSEI:
		fallthrough
	case NALUTypeSPS:
		fallthrough
	case NALUTypePPS:
		return ""
	}

	t := CalcSliceType(nalu)
	ret, ok := SliceTypeMapping[t]
	if !ok {
		return "unknown"
	}
	return ret
}

// TODO chef: 参考 hls session的代码，重构这个函数
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

// TODO chef: 和HLS中的代码有重复，合并一下

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
		_, _ = w.Write(NALUStartCode)
		_, _ = w.Write(sps)
		_, _ = w.Write(NALUStartCode)
		_, _ = w.Write(pps)
		return nil
	}

	// payload中可能存在多个nalu
	// 先跳过前面type的2字节，以及composition time的3字节
	for i := 5; i != len(payload); {
		naluLen := int(bele.BEUint32(payload[i:]))
		i += 4
		//naluType := payload[i] & 0x1f
		//log.Debugf("naluLen:%d t:%d %s\n", naluLen, naluType, avc.NALUTypeMapping[naluUintType])
		_, _ = w.Write(NALUStartCode)
		_, _ = w.Write(payload[i : i+naluLen])
		i += naluLen
		break
	}
	return nil
}
