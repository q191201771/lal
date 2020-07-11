// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv

import (
	"errors"
)

var ErrHTTPFLV = errors.New("lal.httpflv: fxxk")

const (
	TagHeaderSize int = 11

	flvHeaderSize        int = 13
	prevTagSizeFieldSize int = 4
)

const (
	TagTypeMetadata uint8 = 18
	TagTypeVideo    uint8 = 9
	TagTypeAudio    uint8 = 8
)

const (
	frameTypeKey   uint8 = 1
	frameTypeInter uint8 = 2
)

const (
	codecIDAVC  uint8 = 7
	codecIDHEVC uint8 = 12
)

const (
	AVCKeyFrame   = frameTypeKey<<4 | codecIDAVC
	AVCInterFrame = frameTypeInter<<4 | codecIDAVC

	HEVCKeyFrame   = frameTypeKey<<4 | codecIDHEVC
	HEVCInterFrame = frameTypeInter<<4 | codecIDHEVC
)

const (
	AVCPacketTypeSeqHeader uint8 = 0
	AVCPacketTypeNALU      uint8 = 1

	HEVCPacketTypeSeqHeader uint8 = 0
	HEVCPacketTypeNALU      uint8 = 1

	AACPacketTypeSeqHeader uint8 = 0
	AACPacketTypeRaw       uint8 = 1
)

const (
	SoundFormatAAC uint8 = 10
)
