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

type RtpPackerPayloadAvcType int

const (
	RtpPackerPayloadAvcTypeNalu RtpPackerPayloadAvcType = iota + 1
	RtpPackerPayloadAvcTypeAvcc
	RtpPackerPayloadAvcTypeAnnexb
)

type RtpPackerPayloadAvcOption struct {
	Typ RtpPackerPayloadAvcType
}

var defaultRtpPackerPayloadAvcOption = RtpPackerPayloadAvcOption{
	Typ: RtpPackerPayloadAvcTypeNalu,
}

type RtpPackerPayloadAvc struct {
	option RtpPackerPayloadAvcOption
}

type ModRtpPackerPayloadAvcOption func(option *RtpPackerPayloadAvcOption)

func NewRtpPackerPayloadAvc(modOptions ...ModRtpPackerPayloadAvcOption) *RtpPackerPayloadAvc {
	option := defaultRtpPackerPayloadAvcOption
	for _, fn := range modOptions {
		fn(&option)
	}
	return &RtpPackerPayloadAvc{
		option: option,
	}
}

// @param in: AVCC格式
//
// @return out: 内存块为独立新申请；函数返回后，内部不再持有该内存块
//
func (r *RtpPackerPayloadAvc) Pack(in []byte, maxSize int) (out [][]byte) {
	if in == nil || maxSize <= 0 {
		return
	}

	var splitFn func([]byte) ([][]byte, error)

	switch r.option.Typ {
	case RtpPackerPayloadAvcTypeNalu:
		return r.PackNal(in, maxSize)
	case RtpPackerPayloadAvcTypeAvcc:
		splitFn = avc.SplitNaluAvcc
	case RtpPackerPayloadAvcTypeAnnexb:
		splitFn = avc.SplitNaluAnnexb
	}

	nals, err := splitFn(in)
	if err != nil {
		return
	}

	for _, nal := range nals {
		nalType := nal[0] & 0x1F
		if nalType == avc.NaluTypeAud {
			continue
		}

		out = append(out, r.PackNal(nal, maxSize)...)
	}
	return
}

func (r *RtpPackerPayloadAvc) PackNal(nal []byte, maxSize int) (out [][]byte) {
	nalType := nal[0] & 0x1F
	nri := nal[0] & 0x60

	// single
	if len(nal) <= maxSize-fuaHeaderSize {
		item := make([]byte, len(nal))
		copy(item, nal)
		out = append(out, item)
		return
	}

	// FU-A
	var length int
	// 注意，跳过输入的nal type那个字节，使用FU-A自己的两个字节的头，避免重复
	bpos := 1

	epos := len(nal)
	for {
		if epos-bpos > maxSize-fuaHeaderSize {
			// 前面的包
			length = maxSize
			item := make([]byte, maxSize)
			// fuIndicator
			item[0] = NaluTypeAvcFua
			item[0] |= nri
			// fuHeader
			item[1] = nalType
			// 当前帧切割后的首个RTP包
			if bpos == 1 {
				item[1] |= 0x80 // start
			}
			//
			copy(item[fuaHeaderSize:], nal[bpos:bpos+maxSize-fuaHeaderSize])
			out = append(out, item)
			bpos += maxSize - fuaHeaderSize
			continue
		}

		// 最后一包
		length = epos - bpos + fuaHeaderSize
		item := make([]byte, length)
		// fuIndicator
		item[0] = NaluTypeAvcFua
		item[0] |= nri
		// fuHeader
		item[1] = nalType | 0x40 // end
		copy(item[fuaHeaderSize:], nal[bpos:])
		out = append(out, item)
		break
	}
	return
}
