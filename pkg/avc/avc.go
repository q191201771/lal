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

// AnnexB:
//   keywords: MPEG-2 transport stream, ElementaryStream(ES),
//   nalu with start code.
//   e.g. ts
//
// AVCC:
//   keywords: AVC1, MPEG-4, extradata, sequence header, AVCDecoderConfigurationRecord
//   nalu with length prefix.
//   e.g. rtmp, flv

var ErrAVC = errors.New("lal.avc: fxxk")

var (
	NALUStartCode3 = []byte{0x0, 0x0, 0x1}
	NALUStartCode4 = []byte{0x0, 0x0, 0x0, 0x1}

	// aud nalu
	AUDNALU = []byte{0x00, 0x00, 0x00, 0x01, 0x09, 0xf0}
)

// H.264-AVC-ISO_IEC_14496-15.pdf
// Table 1 - NAL unit types in elementary streams
var NALUTypeMapping = map[uint8]string{
	1:  "SLICE",
	5:  "IDR",
	6:  "SEI",
	7:  "SPS",
	8:  "PPS",
	9:  "AUD",
	12: "FD",
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
	NALUTypeAUD      uint8 = 9  // Access Unit Delimiter
	NALUTypeFD       uint8 = 12 // Filler Data
)

const (
	SliceTypeP  uint8 = 0
	SliceTypeB  uint8 = 1
	SliceTypeI  uint8 = 2
	SliceTypeSP uint8 = 3
	SliceTypeSI uint8 = 4
)

type Context struct {
	Profile uint8
	Level   uint8
	Width   uint32
	Height  uint32
}

// H.264-AVC-ISO_IEC_14496-15.pdf
// 5.2.4 Decoder configuration information
type DecoderConfigurationRecord struct {
	ConfigurationVersion uint8
	AVCProfileIndication uint8
	ProfileCompatibility uint8
	AVCLevelIndication   uint8
	LengthSizeMinusOne   uint8
	NumOfSPS             uint8
	SPSLength            uint16
	NumOfPPS             uint8
	PPSLength            uint16
}

// ISO-14496-10.pdf
// 7.3.2.1 Sequence parameter set RBSP syntax
// 7.4.2.1 Sequence parameter set RBSP semantics
type SPS struct {
	ProfileIdc                     uint8
	ConstraintSet0Flag             uint8
	ConstraintSet1Flag             uint8
	ConstraintSet2Flag             uint8
	LevelIdc                       uint8
	SPSId                          uint32
	ChromaFormatIdc                uint32
	ResidualColorTransformFlag     uint8
	BitDepthLuma                   uint32
	BitDepthChroma                 uint32
	TransFormBypass                uint8
	Log2MaxFrameNumMinus4          uint32
	PicOrderCntType                uint32
	Log2MaxPicOrderCntLsb          uint32
	NumRefFrames                   uint32 // num_ref_frames
	GapsInFrameNumValueAllowedFlag uint8  // gaps_in_frame_num_value_allowed_flag
	PicWidthInMbsMinusOne          uint32 // pic_width_in_mbs_minus1
	PicHeightInMapUnitsMinusOne    uint32 // pic_height_in_map_units_minus1
	FrameMbsOnlyFlag               uint8  // frame_mbs_only_flag
	MbAdaptiveFrameFieldFlag       uint8  // mb_adaptive_frame_field_flag
	Direct8X8InferenceFlag         uint8  // direct_8x8_inference_flag
	FrameCroppingFlag              uint8  // frame_cropping_flag
	FrameCropLeftOffset            uint32 // frame_crop_left_offset
	FrameCropRightOffset           uint32 // frame_crop_right_offset
	FrameCropTopOffset             uint32 // frame_crop_top_offset
	FrameCropBottomOffset          uint32 // frame_crop_bottom_offset
}

func ParseNALUType(v uint8) uint8 {
	return v & 0x1f
}

func ParseSliceType(nalu []byte) (uint8, error) {
	if len(nalu) < 2 {
		return 0, ErrAVC
	}

	br := nazabits.NewBitReader(nalu[1:])

	// skip first_mb_in_slice
	if _, err := br.ReadGolomb(); err != nil {
		return 0, err
	}

	sliceType, err := br.ReadGolomb()
	if err != nil {
		return 0, err
	}

	// range: [0, 9]
	if sliceType > 9 {
		return 0, ErrAVC
	}

	if sliceType > 4 {
		sliceType -= 5
	}
	return uint8(sliceType), nil
}

func ParseNALUTypeReadable(v uint8) string {
	t := ParseNALUType(v)
	ret, ok := NALUTypeMapping[t]
	if !ok {
		return "unknown"
	}
	return ret
}

func ParseSliceTypeReadable(nalu []byte) (string, error) {
	naluType := ParseNALUType(nalu[0])

	// 这些类型不属于视频帧数据类型，没有slice type
	switch naluType {
	case NALUTypeSEI:
		fallthrough
	case NALUTypeSPS:
		fallthrough
	case NALUTypePPS:
		return "", nil
	}

	t, err := ParseSliceType(nalu)
	if err != nil {
		return "unknown", err
	}
	ret, ok := SliceTypeMapping[t]
	if !ok {
		return "unknown", ErrAVC
	}
	return ret, nil
}

// AVCC Seq Header -> AnnexB
//
// @param payload: rtmp message的payload部分或者flv tag的payload部分
//                 注意，包含了头部2字节类型以及3字节的cts
//
// @return 注意，返回的内存块为独立的内存块，不依赖指向传输参数<payload>内存块
//
func SPSPPSSeqHeader2AnnexB(payload []byte) ([]byte, error) {
	sps, pps, err := ParseSPSPPSFromSeqHeader(payload)
	if err != nil {
		return nil, ErrAVC
	}
	var ret []byte
	ret = append(ret, NALUStartCode4...)
	ret = append(ret, sps...)
	ret = append(ret, NALUStartCode4...)
	ret = append(ret, pps...)
	return ret, nil
}

// 从AVCC格式的Seq Header中得到SPS和PPS内存块
//
// @param payload: rtmp message的payload部分或者flv tag的payload部分
//                 注意，包含了头部2字节类型以及3字节的cts
//
// @return 注意，返回的sps，pps内存块指向的是传入参数<payload>内存块的内存
//
func ParseSPSPPSFromSeqHeader(payload []byte) (sps, pps []byte, err error) {
	if len(payload) < 5 {
		return nil, nil, ErrAVC
	}
	if payload[0] != 0x17 || payload[1] != 0x00 || payload[2] != 0 || payload[3] != 0 || payload[4] != 0 {
		return nil, nil, ErrAVC
	}

	if len(payload) < 13 {
		return nil, nil, ErrAVC
	}

	index := 10
	numOfSPS := int(payload[index] & 0x1F)
	index++
	if numOfSPS != 1 {
		return nil, nil, ErrAVC
	}
	spsLength := int(bele.BEUint16(payload[index:]))
	index += 2

	if len(payload) < 13+spsLength {
		return nil, nil, ErrAVC
	}

	sps = payload[index : index+spsLength]
	index += spsLength

	if len(payload) < 16+spsLength {
		return nil, nil, ErrAVC
	}

	numOfPPS := int(payload[index] & 0x1F)
	index++
	if numOfPPS != 1 {
		return nil, nil, ErrAVC
	}
	ppsLength := int(bele.BEUint16(payload[index:]))
	index += 2

	if len(payload) < 16+spsLength+ppsLength {
		return nil, nil, ErrAVC
	}

	pps = payload[index : index+ppsLength]
	return
}

// 返回的内存块为新申请的独立内存块
func BuildSeqHeaderFromSPSPPS(sps, pps []byte) ([]byte, error) {
	var sh []byte
	sh = make([]byte, 16+len(sps)+len(pps))
	sh[0] = 0x17
	sh[1] = 0x0
	sh[2] = 0x0
	sh[3] = 0x0
	sh[4] = 0x0

	// H.264-AVC-ISO_IEC_14496-15.pdf
	// 5.2.4 Decoder configuration information
	sh[5] = 0x1 // configurationVersion

	var ctx Context
	if err := ParseSPS(sps, &ctx); err != nil {
		return nil, err
	}

	sh[6] = ctx.Profile // AVCProfileIndication
	sh[7] = 0           // profile_compatibility
	sh[8] = ctx.Level   // AVCLevelIndication
	sh[9] = 0xFF        // lengthSizeMinusOne '111111'b | (4-1)
	sh[10] = 0xE1       // numOfSequenceParameterSets '111'b | 1

	sh[11] = uint8((len(sps) >> 8) & 0xFF) // sequenceParameterSetLength
	sh[12] = uint8(len(sps) & 0xFF)

	i := 13
	copy(sh[i:], sps)
	i += len(sps)

	sh[i] = 0x1 // numOfPictureParameterSets 1
	i++

	sh[i] = uint8((len(pps) >> 8) & 0xFF) // sequenceParameterSetLength
	sh[i+1] = uint8(len(pps) & 0xFF)
	i += 2

	copy(sh[i:], pps)

	return sh, nil
}

// AVCC -> AnnexB
//
// @param payload: rtmp message的payload部分或者flv tag的payload部分
//                 注意，包含了头部2字节类型以及3字节的cts
//
func CaptureAVCC2AnnexB(w io.Writer, payload []byte) error {
	// sps pps
	if payload[0] == 0x17 && payload[1] == 0x00 {
		spspps, err := SPSPPSSeqHeader2AnnexB(payload)
		if err != nil {
			return err
		}
		_, _ = w.Write(spspps)
		return nil
	}

	// payload中可能存在多个nalu
	for i := 5; i != len(payload); {
		naluLen := int(bele.BEUint32(payload[i:]))
		i += 4
		_, _ = w.Write(NALUStartCode4)
		_, _ = w.Write(payload[i : i+naluLen])
		i += naluLen
		break
	}
	return nil
}

// 遍历直到找到第一个nalu start code的位置
//
// @param start: 从`nalu`的start位置开始查找
//
// @return pos:    start code的起始位置（包含start code自身）
//         length: start code的长度，可能是3或者4
//         注意，如果找不到start code，则返回-1, -1
//
func IterateNALUStartCode(nalu []byte, start int) (pos, length int) {
	if nalu == nil || start >= len(nalu) {
		return -1, -1
	}
	count := 0
	for i := range nalu[start:] {
		switch nalu[start+i] {
		case 0:
			count++
		case 1:
			if count >= 2 {
				return start + i - count, count + 1
			}
			count = 0
		default:
			count = 0
		}
	}
	return -1, -1
}

// 遍历AnnexB格式，去掉start code，获取nal包，正常情况下可能为1个或多个，异常情况下可能一个也没有
//
// 具体见单元测试
//
func IterateNALUAnnexB(nals []byte) (nalList [][]byte, err error) {
	if nals == nil {
		err = ErrAVC
		return
	}
	prePos, preLength := IterateNALUStartCode(nals, 0)
	if prePos == -1 {
		nalList = append(nalList, nals)
		err = ErrAVC
		return
	}

	for {
		pos, length := IterateNALUStartCode(nals, prePos+preLength)
		start := prePos + preLength
		if pos == -1 {
			if start < len(nals) {
				nalList = append(nalList, nals[start:])
			} else {
				err = ErrAVC
			}
			return
		}
		if start < pos {
			nalList = append(nalList, nals[start:pos])
		} else {
			err = ErrAVC
		}

		prePos = pos
		preLength = length
	}
}

// 遍历AVCC格式，去掉4字节长度，获取nal包，正常情况下可能返回1个或多个，异常情况下可能一个也没有
//
// 具体见单元测试
//
func IterateNALUAVCC(nals []byte) (nalList [][]byte, err error) {
	if nals == nil {
		err = ErrAVC
		return
	}
	pos := 0
	for {
		if len(nals[pos:]) < 4 {
			err = ErrAVC
			return
		}
		length := int(bele.BEUint32(nals[pos:]))
		pos += 4
		if pos == len(nals) {
			err = ErrAVC
			return
		}
		epos := pos + length
		if epos < len(nals) {
			// 非最后一个
			nalList = append(nalList, nals[pos:epos])
			pos += length
		} else if epos == len(nals) {
			// 最后一个
			nalList = append(nalList, nals[pos:epos])
			return
		} else {
			nalList = append(nalList, nals[pos:])
			err = ErrAVC
			return
		}
	}
}

// TODO(chef)
// func NALUAVCC2AnnexB
// func NALUAnnexB2AVCC
