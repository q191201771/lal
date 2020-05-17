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
)

// <iso13818-1.pdf>
// <2.4.3.6 PES packet> <page 49/174>
// <Table E.1 - PES packet header example> <page 142/174>
// <F.0.2 PES packet> <page 144/174>
// packet_start_code_prefix  [24b] *** always 0x00, 0x00, 0x01
// stream_id                 [8b]  *
// PES_packet_length         [16b] **
// '10'                      [2b]
// PES_scrambling_control    [2b]
// PES_priority              [1b]
// data_alignment_indicator  [1b]
// copyright                 [1b]
// original_or_copy          [1b]  *
// PTS_DTS_flags             [2b]
// ESCR_flag                 [1b]
// ES_rate_flag              [1b]
// DSM_trick_mode_flag       [1b]
// additional_copy_info_flag [1b]
// PES_CRC_flag              [1b]
// PES_extension_flag        [1b]  *
// PES_header_data_length    [8b]  *
type PES struct {
	pscp uint32
	sid  uint8
	ppl  uint16
	pad1 uint8
	pdf  uint8
	pad2 uint8
	phdl uint8
}

func ParsePES(b []byte) (pes PES, length int) {
	br := nazabits.NewBitReader(b)
	pes.pscp = br.ReadBits32(24)
	pes.sid = br.ReadBits8(8)
	pes.ppl = br.ReadBits16(16)

	pes.pad1 = br.ReadBits8(8)
	pes.pdf = br.ReadBits8(2)
	pes.pad2 = br.ReadBits8(6)
	pes.phdl = br.ReadBits8(8)

	br.ReadBytes(uint(pes.phdl))
	length = 9 + int(pes.phdl)

	return
}
