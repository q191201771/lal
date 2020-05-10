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

// Program association section
// <iso13818-1.pdf> <2.4.4.3> <page 61/174>
// table_id                 [8b] *
// section_syntax_indicator [1b]
// '0'                      [1b]
// reserved                 [2b]
// section_length           [12b] **
// transport_stream_id      [16b] **
// reserved                 [2b]
// version_number           [5b]
// current_next_indicator   [1b]  *
// section_number           [8b]  *
// last_section_number      [8b]  *
// -----loop-----
// program_number           [16b] **
// reserved                 [3b]
// program_map_PID          [13b] ** if program_number == 0 then network_PID else then program_map_PID
// --------------
// CRC_32                   [32b] ****
type PAT struct {
	tid   uint8
	ssi   uint8
	sl    uint16
	tsi   uint16
	vn    uint8
	cni   uint8
	sn    uint8
	lsn   uint8
	ppes  []PATProgramElement
	crc32 uint32
}

type PATProgramElement struct {
	pn    uint16
	pmpid uint16
}

func ParsePAT(b []byte) (pat PAT) {
	br := nazabits.NewBitReader(b)
	pat.tid = br.ReadBits8(8)
	pat.ssi = br.ReadBits8(1)
	br.ReadBits8(3)
	pat.sl = br.ReadBits16(12)
	pat.tsi = br.ReadBits16(16)
	br.ReadBits8(2)
	pat.vn = br.ReadBits8(5)
	pat.cni = br.ReadBits8(1)
	pat.sn = br.ReadBits8(8)
	pat.lsn = br.ReadBits8(8)

	len := pat.sl - 9

	for i := uint16(0); i < len; i += 4 {
		var ppe PATProgramElement
		ppe.pn = br.ReadBits16(16)
		br.ReadBits8(3)
		// TODO chef if pn == 0
		ppe.pmpid = br.ReadBits16(13)
		pat.ppes = append(pat.ppes, ppe)
	}
	pat.crc32 = br.ReadBits32(32)
	return
}

func (pat *PAT) searchPID(pid uint16) bool {
	for _, ppe := range pat.ppes {
		if pid == ppe.pmpid {
			return true
		}
	}
	return false
}
