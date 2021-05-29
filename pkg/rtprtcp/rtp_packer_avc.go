// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtprtcp

import "github.com/q191201771/lal/pkg/avc"

const (
	fuaHeaderSize = 2
)

type RTPPackerAVC struct {
}

// @param nalu: AVCC格式
//
// @return 返回RTP(only body)的数组
//         内存块为独立申请，函数返回后，内部不再持有该内存块
//
func (*RTPPackerAVC) Pack(nalu []byte, maxSize int) (ret [][]byte) {
	if nalu == nil || maxSize <= 0 {
		return
	}

	nals, err := avc.SplitNALUAVCC(nalu)
	if err != nil {
		return
	}

	for _, nal := range nals {
		nalType := nal[0] & 0x1F
		nri := nal[0] & 0x60

		if nalType == avc.NALUTypeAUD {
			continue
		}

		// single
		if len(nal) <= maxSize-fuaHeaderSize {
			item := make([]byte, len(nal))
			copy(item, nal)
			ret = append(ret, item)
			continue
		}

		// FU-A
		var length int
		bpos := 0
		epos := len(nal)
		for {
			if epos-bpos > maxSize-fuaHeaderSize {
				// 前面的包
				length = maxSize
				item := make([]byte, maxSize)
				// fuIndicator
				item[0] = NALUTypeAVCFUA
				item[0] |= nri
				// fuHeader
				item[1] = nalType
				if bpos == 0 {
					item[1] |= 0x80 // start
				}
				//
				copy(item[fuaHeaderSize:], nal[bpos:bpos+maxSize-fuaHeaderSize])
				bpos += maxSize - fuaHeaderSize
				continue
			}

			// 最后一包
			length = epos - bpos + fuaHeaderSize
			item := make([]byte, length)
			// fuIndicator
			item[0] = NALUTypeAVCFUA
			item[0] |= nri
			// fuHeader
			item[1] = nalType | 0x40 // end
			copy(item[fuaHeaderSize:], nal[bpos:])
			break
		}
	}
	return
}
