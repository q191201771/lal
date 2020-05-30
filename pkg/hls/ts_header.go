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

// ------------------------------------------------
// <iso13818-1.pdf> <2.4.3.2> <page 36/174>
// sync_byte                    [8b]  * always 0x47
// transport_error_indicator    [1b]
// payload_unit_start_indicator [1b]
// transport_priority           [1b]
// PID                          [13b] **
// transport_scrambling_control [2b]
// adaptation_field_control     [2b]
// continuity_counter           [4b]  *
// ------------------------------------------------
type TSPacketHeader struct {
	Sync             uint8
	Err              uint8
	PayloadUnitStart uint8
	Prio             uint8
	Pid              uint16
	Scra             uint8
	Adaptation       uint8
	CC               uint8
}

// ----------------------------------------------------------
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
// ----------------------------------------------------------
type TSPacketAdaptation struct {
	Length uint8
}

// 解析4字节TS Packet header
func ParseTSPacketHeader(b []byte) (h TSPacketHeader) {
	br := nazabits.NewBitReader(b)
	h.Sync = br.ReadBits8(8)
	nazalog.Assert(uint8(0x47), h.Sync)
	h.Err = br.ReadBits8(1)
	h.PayloadUnitStart = br.ReadBits8(1)
	h.Prio = br.ReadBits8(1)
	h.Pid = br.ReadBits16(13)
	h.Scra = br.ReadBits8(2)
	h.Adaptation = br.ReadBits8(2)
	h.CC = br.ReadBits8(4)
	return
}

// TODO chef
func ParseTSPacketAdaptation(b []byte) (f TSPacketAdaptation) {
	br := nazabits.NewBitReader(b)
	f.Length = br.ReadBits8(8)
	return
}
