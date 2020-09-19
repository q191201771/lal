// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hevc

import (
	"errors"

	"github.com/q191201771/naza/pkg/nazabits"

	"github.com/q191201771/naza/pkg/bele"
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

type Context struct {
	// unsigned int(8) configurationVersion = 1;
	configurationVersion uint8
	// unsigned int(2) general_profile_space;
	// unsigned int(1) general_tier_flag;
	// unsigned int(5) general_profile_idc;
	generalProfileSpace uint8
	generalTierFlag     uint8
	generalProfileIDC   uint8
	// unsigned int(32) general_profile_compatibility_flags;
	generalProfileCompatibilityFlags uint32
	// unsigned int(48) general_constraint_indicator_flags;
	generalConstraintIndicatorFlags uint64
	generalLevelIDC                 uint8

	numTemporalLayers uint8

	chromaFormat         uint32
	bitDepthLumaMinus8   uint32
	bitDepthChromaMinus8 uint32
}

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
	//nazalog.Debugf("%s", hex.Dump(payload))

	if len(payload) < 33 {
		return nil, nil, nil, ErrHEVC
	}

	index := 27
	if numOfArrays := payload[index]; numOfArrays != 3 && numOfArrays != 4 {
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

func BuildSeqHeaderFromVPSSPSPPS(vps, sps, pps []byte) ([]byte, error) {
	var sh []byte
	sh = make([]byte, 1024)
	sh[0] = 0x1c
	sh[1] = 0x0
	sh[2] = 0x0
	sh[3] = 0x0
	sh[4] = 0x0

	// unsigned int(8) configurationVersion = 1;
	sh[5] = 0x1

	ctx := newContext()
	if err := ParseVPS(vps, ctx); err != nil {
		return nil, err
	}
	if err := ParseSPS(sps, ctx); err != nil {
		return nil, err
	}

	// unsigned int(2) general_profile_space;
	// unsigned int(1) general_tier_flag;
	// unsigned int(5) general_profile_idc;
	sh[6] = ctx.generalProfileSpace<<6 | ctx.generalTierFlag<<5 | ctx.generalProfileIDC
	// unsigned int(32) general_profile_compatibility_flags
	bele.BEPutUint32(sh[7:], ctx.generalProfileCompatibilityFlags)
	// unsigned int(48) general_constraint_indicator_flags
	bele.BEPutUint32(sh[11:], uint32(ctx.generalConstraintIndicatorFlags>>16))
	bele.BEPutUint16(sh[15:], uint16(ctx.generalConstraintIndicatorFlags))
	sh[17] = ctx.generalLevelIDC

	return sh, nil
}

func ParseVPS(vps []byte, ctx *Context) error {
	br := nazabits.NewBitReader(vps)

	// type
	if _, err := br.ReadBits8(8); err != nil {
		return err
	}

	// skip
	// vps_video_parameter_set_id u(4)
	// vps_reserved_three_2bits   u(2)
	// vps_max_layers_minus1      u(6)
	if _, err := br.ReadBits16(12); err != nil {
		return ErrHEVC
	}

	vpsMaxSubLayersMinus1, err := br.ReadBits8(3)
	if err != nil {
		return ErrHEVC
	}
	if vpsMaxSubLayersMinus1+1 > ctx.numTemporalLayers {
		ctx.numTemporalLayers = vpsMaxSubLayersMinus1 + 1
	}

	// skip
	// vps_temporal_id_nesting_flag u(1)
	// vps_reserved_0xffff_16bits   u(16)
	if _, err := br.ReadBits32(17); err != nil {
		return ErrHEVC
	}

	return parsePTL(br, ctx, vpsMaxSubLayersMinus1)
}

func ParseSPS(sps []byte, ctx *Context) error {
	var err error
	br := nazabits.NewBitReader(sps)

	// type
	if _, err = br.ReadBits8(8); err != nil {
		return err
	}

	// sps_video_parameter_set_id
	if _, err = br.ReadBits8(4); err != nil {
		return err
	}

	spsMaxSubLayersMinus1, err := br.ReadBits8(3)
	if err != nil {
		return err
	}

	if _, err := br.ReadBit(); err != nil {
		return err
	}

	if err = parsePTL(br, ctx, spsMaxSubLayersMinus1); err != nil {
		return err
	}

	if _, err = br.ReadGolomb(); err != nil {
		return err
	}

	if ctx.chromaFormat, err = br.ReadGolomb(); err != nil {
		return err
	}
	if ctx.chromaFormat == 3 {
		if _, err = br.ReadBit(); err != nil {
			return err
		}
	}

	if _, err = br.ReadGolomb(); err != nil {
		return err
	}
	if _, err = br.ReadGolomb(); err != nil {
		return err
	}

	conformanceWindowFlag, err := br.ReadBit()
	if err != nil {
		return err
	}
	if conformanceWindowFlag != 0 {
		if _, err = br.ReadGolomb(); err != nil {
			return err
		}
		if _, err = br.ReadGolomb(); err != nil {
			return err
		}
		if _, err = br.ReadGolomb(); err != nil {
			return err
		}
		if _, err = br.ReadGolomb(); err != nil {
			return err
		}
	}

	if ctx.bitDepthLumaMinus8, err = br.ReadGolomb(); err != nil {
		return err
	}
	if ctx.bitDepthChromaMinus8, err = br.ReadGolomb(); err != nil {
		return err
	}
	_, err = br.ReadGolomb()
	if err != nil {
		return err
	}
	spsSubLayerOrderingInfoPresentFlag, err := br.ReadBit()
	if err != nil {
		return err
	}
	var i uint8
	if spsSubLayerOrderingInfoPresentFlag != 0 {
		i = 0
	} else {
		i = spsMaxSubLayersMinus1
	}
	for ; i <= spsMaxSubLayersMinus1; i++ {
		if _, err = br.ReadGolomb(); err != nil {
			return err
		}
		if _, err = br.ReadGolomb(); err != nil {
			return err
		}
		if _, err = br.ReadGolomb(); err != nil {
			return err
		}
	}

	if _, err = br.ReadGolomb(); err != nil {
		return err
	}
	if _, err = br.ReadGolomb(); err != nil {
		return err
	}
	if _, err = br.ReadGolomb(); err != nil {
		return err
	}
	if _, err = br.ReadGolomb(); err != nil {
		return err
	}
	if _, err = br.ReadGolomb(); err != nil {
		return err
	}
	if _, err = br.ReadGolomb(); err != nil {
		return err
	}

	return nil
}

func parsePTL(br nazabits.BitReader, ctx *Context, maxSubLayersMinus1 uint8) error {
	var err error
	var ptl Context
	if ptl.generalProfileSpace, err = br.ReadBits8(2); err != nil {
		return err
	}
	if ptl.generalTierFlag, err = br.ReadBit(); err != nil {
		return err
	}
	if ptl.generalProfileIDC, err = br.ReadBits8(5); err != nil {
		return err
	}
	if ptl.generalProfileCompatibilityFlags, err = br.ReadBits32(32); err != nil {
		return err
	}
	if ptl.generalConstraintIndicatorFlags, err = br.ReadBits64(48); err != nil {
		return err
	}
	if ptl.generalLevelIDC, err = br.ReadBits8(8); err != nil {
		return err
	}
	updatePTL(ctx, &ptl)

	/*
		subLayerProfilePresentFlag := make([]uint8, maxSubLayersMinus1)
		subLayerLevelPresentFlag := make([]uint8, maxSubLayersMinus1)
		for i := uint8(0); i < maxSubLayersMinus1; i++ {
			if subLayerProfilePresentFlag[i], err = br.ReadBit(); err != nil {
				return err
			}
			if subLayerLevelPresentFlag[i], err = br.ReadBit(); err != nil {
				return err
			}
		}
		if maxSubLayersMinus1 > 0 {
			for i := maxSubLayersMinus1; i < 8; i++ {
				if _, err = br.ReadBits8(2); err != nil {
					return err
				}
			}
		}

		for i := uint8(0); i < maxSubLayersMinus1; i++ {
			if subLayerProfilePresentFlag[i] != 0 {
				if _, err = br.ReadBits32(32); err != nil {
					return err
				}
				if _, err = br.ReadBits32(32); err != nil {
					return err
				}
				if _, err = br.ReadBits32(24); err != nil {
					return err
				}
			}

			if subLayerLevelPresentFlag[i] != 0 {
				if _, err = br.ReadBits8(8); err != nil {
					return err
				}
			}
		}
	*/

	return nil
}

func updatePTL(ctx, ptl *Context) {
	ctx.generalProfileSpace = ptl.generalProfileSpace

	if ctx.generalTierFlag < ptl.generalTierFlag {
		ctx.generalLevelIDC = ptl.generalLevelIDC
	} else {
		if ptl.generalLevelIDC > ctx.generalLevelIDC {
			ctx.generalLevelIDC = ptl.generalLevelIDC
		}

		ctx.generalTierFlag = ptl.generalTierFlag
	}

	if ptl.generalProfileIDC > ctx.generalProfileIDC {
		ctx.generalProfileIDC = ptl.generalProfileIDC
	}

	ctx.generalProfileCompatibilityFlags &= ptl.generalProfileCompatibilityFlags

	ctx.generalConstraintIndicatorFlags &= ptl.generalConstraintIndicatorFlags
}

func newContext() *Context {
	return &Context{
		configurationVersion: 1,
		//hvcc->lengthSizeMinusOne   = 3; // 4 bytes
		generalProfileCompatibilityFlags: 0xffffffff,
		generalConstraintIndicatorFlags:  0xffffffffffff,
		//hvcc->min_spatial_segmentation_idc = MAX_SPATIAL_SEGMENTATION + 1;
	}
}

//func skipScalingListData(br nazabits.BitReader) error {
//	var numCoeffs int
//	var i int
//
//	for i = 0; i < 4; i++ {
//		k := 6
//		if i == 3 {
//			k = 2
//		}
//		for j := 0; j < k; j++ {
//			f, err := br.ReadBit()
//			if err != nil {
//				return err
//			}
//			if f != 0 {
//				if _, err := br.ReadGolomb(); err != nil {
//					return err
//				}
//			} else {
//				numCoeffs = 1 << (4+(uint32(i)<<1))
//				if numCoeffs > 64 {
//					numCoeffs = 64
//				}
//			}
//		}
//	}
//
//	if i > 1 {
//		if _, err := br.ReadGolomb(); err != nil {
//			return err
//		}
//	}
//	for i := 0; i < numCoeffs; i++ {
//		if _, err := br.ReadGolomb(); err != nil {
//			return err
//		}
//	}
//
//	return nil
//}
