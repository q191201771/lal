// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import (
	"github.com/q191201771/naza/pkg/nazabits"
	"github.com/q191201771/naza/pkg/nazalog"
)

// <iso13818-1.pdf> <2.4.3.2> <page 36/174>
// sync_byte                    [8b]  * always 0x47
// transport_error_indicator    [1b]
// payload_unit_start_indicator [1b]    如果为1，读完头需要跳过一个字节
// transport_priority           [1b]
// PID                          [13b] **
// transport_scrambling_control [2b]
// adaptation_field_control     [2b]
// continuity_counter           [4b] *
type TSPacketHeader struct {
	sb   uint8
	tei  uint8
	pusi uint8
	tp   uint8
	pid  uint16
	tsc  uint8
	afc  uint8
	cc   uint8
}

// <iso13818-1.pdf> <Table 2-6> <page 40/174>
// adaptation_field_length              [8b] * 不包括自己这1字节
// discontinuity_indicator              [1b]
// random_access_indicator              [1b]
// elementary_stream_priority_indicator [1b]
// PCR_flag                             [1b]
// OPCR_flag                            [1b]
// splicing_point_flag                  [1b]
// transport_private_data_flag          [1b]
// adaptation_field_extension_flag      [1b] *
// -----if PCR_flag == 1-----
// program_clock_reference_base         [33b]
// reserved                             [6b]
// program_clock_reference_extension    [9b] ******
type TSAdaptationField struct {
	afl uint8
}

// 解析4字节TS header
func ParseTSPacketHeader(b []byte) (h TSPacketHeader) {
	br := nazabits.NewBitReader(b)
	h.sb = br.ReadBits8(8)
	nazalog.Assert(uint8(0x47), h.sb)
	h.tei = br.ReadBits8(1)
	h.pusi = br.ReadBits8(1)
	h.tp = br.ReadBits8(1)
	h.pid = br.ReadBits16(13)
	h.tsc = br.ReadBits8(2)
	h.afc = br.ReadBits8(2)
	h.cc = br.ReadBits8(4)
	return
}

// TODO chef
func ParseTSAdaptationField(b []byte) (f TSAdaptationField) {
	br := nazabits.NewBitReader(b)
	f.afl = br.ReadBits8(8)
	return
}
