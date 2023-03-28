// Copyright 2023, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package mpegts

import (
	"github.com/q191201771/naza/pkg/bele"
	"github.com/q191201771/naza/pkg/nazabits"
)

// PsiId
const (
	TsPsiIdPas            = 0x00 // program_association_section
	TsPsiIdCas            = 0x01 // conditional_access_section (CA_section)
	TsPsiIdPms            = 0x02 // TS_program_map_section
	TsPsiIdDs             = 0x03 // TS_description_section
	TsPsiIdSds            = 0x04 // ISO_IEC_14496_scene_description_section
	TsPsiIdOds            = 0x05 // ISO_IEC_14496_object_descriptor_section
	TsPsiIdIso138181Start = 0x06 // ITU-T Rec. H.222.0 | ISO/IEC 13818-1 reserved
	TsPsiIdIso138181End   = 0x37
	TsPsiIdIso138186Start = 0x38 // Defined in ISO/IEC 13818-6
	TsPsiIdIso138186End   = 0x3F
	TsPsiIdUserStart      = 0x40 // User private
	TsPsiIdUserEnd        = 0xFE
	TsPsiIdForbidden      = 0xFF // forbidden
)

type PsiSection struct {
	pointerFileld uint8
	sectionData   PsiSectionData
}

type PsiSectionData struct {
	header  PsiTableHeader
	section PsiTableSyntaxSection
	patData PatSpecificData
	pmtData PmtSpecificData
}

type PsiTableHeader struct {
	tableId                uint8
	sectionSyntaxIndicator uint8
	sectionLength          uint16
}

type PsiTableSyntaxSection struct {
	tableIdExtension     uint16
	versionNumber        uint8
	currentNextIndicator uint8
	sectionNumber        uint8
	lastSectionNumber    uint8
	tableData            []byte
	crc32                uint32
}

type PatSpecificData struct {
	pes []PatProgramElement
}

type PmtSpecificData struct {
	pcrPid            uint16
	programInfoLength uint16
	pes               []PmtProgramElement
}

func NewPsi() *PsiSection {
	return &PsiSection{
		pointerFileld: 0x00,
	}
}

func (psi *PsiSection) Pack() (int, []byte) {
	psiSection := make([]byte, 1+3+psi.calcPsiSectionLength())
	bw := nazabits.NewBitWriter(psiSection)

	bw.WriteBits8(8, psi.pointerFileld)
	psi.writePsiTableHeader(&bw)
	psi.writePsiTableSyntaxSection(&bw)

	crc := CalcCrc32(0xffffffff, psiSection[1:len(psiSection)-4])
	bele.LePutUint32(psiSection[4+psi.calcPsiSectionLength()-4:], crc)

	return int(1 + 3 + psi.calcPsiSectionLength()), psiSection
}

func (psi *PsiSection) writePsiTableHeader(bw *nazabits.BitWriter) {
	bw.WriteBits8(8, psi.sectionData.header.tableId)
	bw.WriteBit(psi.sectionData.header.sectionSyntaxIndicator)
	bw.WriteBit(0)
	bw.WriteBits8(2, 0xff)

	psi.sectionData.header.sectionLength = psi.calcPsiSectionLength()
	bw.WriteBits16(12, psi.sectionData.header.sectionLength)

	return
}

func (psi *PsiSection) writePsiTableSyntaxSection(bw *nazabits.BitWriter) {
	psi.writePsiTableSyntaxSectionHeader(bw)
	psi.writePsiTableSyntaxSectionData(bw)

	return
}

func (psi *PsiSection) writePsiTableSyntaxSectionHeader(bw *nazabits.BitWriter) {
	bw.WriteBits16(16, psi.sectionData.section.tableIdExtension)
	bw.WriteBits8(2, 0xff)
	bw.WriteBits8(5, psi.sectionData.section.versionNumber)
	bw.WriteBit(psi.sectionData.section.currentNextIndicator)
	bw.WriteBits8(8, psi.sectionData.section.sectionNumber)
	bw.WriteBits8(8, psi.sectionData.section.lastSectionNumber)
	return
}

func (psi *PsiSection) writePsiTableSyntaxSectionData(bw *nazabits.BitWriter) {
	switch psi.sectionData.header.tableId {
	case TsPsiIdPas:
		psi.writePatSection(bw)
	case TsPsiIdPms:
		psi.writePmtSection(bw)
	}

	return
}

func (psi *PsiSection) calcPsiSectionLength() (length uint16) {
	if psi.sectionData.header.tableId == TsPsiIdPas || psi.sectionData.header.tableId == TsPsiIdPms {
		// Table ID extension(16 bits)+Reserved bits(2 bits)+Version number(5 bits)+Current next Indicator(1 bit)+Section number(8 bits)+Last section number(8 bits)
		length += 5
	}

	switch psi.sectionData.header.tableId {
	case TsPsiIdPas:
		length += psi.calaPatSectionLength()
	case TsPsiIdPms:
		length += psi.calaPmtSectionLength()
	}

	length += 4 //crc32

	return
}

func (psi *PsiSection) calaPatSectionLength() (length uint16) {
	length = uint16(4 * len(psi.sectionData.patData.pes))
	return
}

func (psi *PsiSection) calaPmtSectionLength() (length uint16) {
	// 暂不考虑Program descriptors
	// Reserved bits(3 bits)+PCR PID(13 bits)+Reserved bits(4 bits)+Program info length(12 bits)
	length = 4
	length += uint16(5 * len(psi.sectionData.pmtData.pes))
	return
}

func (psi *PsiSection) writePatSection(bw *nazabits.BitWriter) {
	for _, pe := range psi.sectionData.patData.pes {
		bw.WriteBits16(16, pe.pn)
		bw.WriteBits8(3, 0xff)
		bw.WriteBits16(13, pe.pmpid)
	}

	return
}

func (psi *PsiSection) writePmtSection(bw *nazabits.BitWriter) {
	bw.WriteBits8(3, 0xff)
	bw.WriteBits16(13, psi.sectionData.pmtData.pcrPid)
	bw.WriteBits8(4, 0xff)
	bw.WriteBits16(12, psi.sectionData.pmtData.programInfoLength)

	for _, pe := range psi.sectionData.pmtData.pes {
		bw.WriteBits8(8, pe.StreamType)
		bw.WriteBits8(3, 0xff)
		bw.WriteBits16(13, pe.Pid)
		bw.WriteBits8(4, 0xff)
		bw.WriteBits16(12, 0)
	}
	return
}
