// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package mpegts

import (
	"github.com/q191201771/naza/pkg/nazabits"
)

// Pmt
//
// ----------------------------------------
// Program Map Table
// <iso13818-1.pdf> <2.4.4.8> <page 64/174>
// table_id                 [8b]  *
// section_syntax_indicator [1b]
// 0                        [1b]
// reserved                 [2b]
// section_length           [12b] **
// program_number           [16b] **
// reserved                 [2b]
// version_number           [5b]
// current_next_indicator   [1b]  *
// section_number           [8b]  *
// last_section_number      [8b]  *
// reserved                 [3b]
// PCR_PID                  [13b] **
// reserved                 [4b]
// program_info_length      [12b] **
// -----loop-----
// stream_type              [8b]  *
// reserved                 [3b]
// elementary_PID           [13b] **
// reserved                 [4b]
// ES_info_length_length    [12b] **
// --------------
// CRC32                    [32b] ****
// ----------------------------------------
//
type Pmt struct {
	tid             uint8
	ssi             uint8
	sl              uint16
	pn              uint16
	vn              uint8
	cni             uint8
	sn              uint8
	lsn             uint8
	pp              uint16
	pil             uint16
	ProgramElements []PmtProgramElement
	crc32           uint32
}

type PmtProgramElement struct {
	StreamType uint8
	Pid        uint16
	Length     uint16
}

func ParsePmt(b []byte) (pmt Pmt) {
	br := nazabits.NewBitReader(b)
	pmt.tid, _ = br.ReadBits8(8)
	pmt.ssi, _ = br.ReadBits8(1)
	_, _ = br.ReadBits8(3)
	pmt.sl, _ = br.ReadBits16(12)
	length := pmt.sl - 13
	pmt.pn, _ = br.ReadBits16(16)
	_, _ = br.ReadBits8(2)
	pmt.vn, _ = br.ReadBits8(5)
	pmt.cni, _ = br.ReadBits8(1)
	pmt.sn, _ = br.ReadBits8(8)
	pmt.lsn, _ = br.ReadBits8(8)
	_, _ = br.ReadBits8(3)
	pmt.pp, _ = br.ReadBits16(13)
	_, _ = br.ReadBits8(4)
	pmt.pil, _ = br.ReadBits16(12)
	if pmt.pil != 0 {
		Log.Warn(pmt.pil)
		_, _ = br.ReadBytes(uint(pmt.pil))
	}

	for i := uint16(0); i < length; i += 5 {
		var ppe PmtProgramElement
		ppe.StreamType, _ = br.ReadBits8(8)
		_, _ = br.ReadBits8(3)
		ppe.Pid, _ = br.ReadBits16(13)
		_, _ = br.ReadBits8(4)
		ppe.Length, _ = br.ReadBits16(12)
		if ppe.Length != 0 {
			Log.Warn(ppe.Length)
			_, _ = br.ReadBits32(uint(ppe.Length))
		}
		pmt.ProgramElements = append(pmt.ProgramElements, ppe)
	}

	return
}

func (pmt *Pmt) SearchPid(pid uint16) *PmtProgramElement {
	for _, ppe := range pmt.ProgramElements {
		if ppe.Pid == pid {
			return &ppe
		}
	}
	return nil
}
