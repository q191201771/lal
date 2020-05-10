// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import (
	"os"

	"github.com/q191201771/naza/pkg/nazalog"
)

type MPEGTSFrame struct {
	pts uint64
	dts uint64
	pid uint16
	sid uint8
	cc  uint8
	key bool
}

func mpegtsOpenFile(filename string) {
	fp, err := os.Create(filename)
	nazalog.Assert(nil, err)
	_, err = fp.Write(TSHeader)
	nazalog.Assert(nil, err)
}

func mpegtsWriteFrame(frame *MPEGTSFrame, b []byte) {
	nazalog.Debugf("mpegts: write frame. %+v, size=%d", frame, len(b))

	packet := make([]byte, 188)
	packet = packet[0:0]
	wpos := 0
	lpos := 0
	rpos := len(b)

	first := true

	for lpos != rpos {
		wpos = 0
		frame.cc++

		// -----TS Header-----
		// sync_byte
		// transport_error_indicator    0
		// payload_unit_start_indicator if first then 1; else then 0;
		// transport_priority           0
		// PID
		// transport_scrambling_control 0
		// adaptation_field_control     if key then 3; else then 1;
		// continuity_counter
		packet[0] = syncByte // sync_byte

		if first {
			packet[1] |= 0x40 // payload_unit_start_indicator
		}
		packet[1] = uint8(frame.pid >> 8)   // PID
		packet[2] = uint8(frame.pid & 0xFF) //

		packet[3] = 0x10 | (frame.cc & 0x0f) // adaptation_field_control, continuity_counter
		wpos += 4

		if first {
			if frame.key {
				// -----Adaptation-----
				// adaptation_field_length
				// discontinuity_indicator              0
				// random_access_indicator              1
				// elementary_stream_priority_indicator 0
				// PCR_flag                             1
				// OPCR_flag                            0
				// splicing_point_flag                  0
				// transport_private_data_flag          0
				// adaptation_field_extension_flag      0
				// program_clock_reference_base
				// reserved
				// program_clock_reference_extension
				packet[3] |= 0x20                        // adaptation_field_control
				packet[4] = 7                            // size
				packet[5] = 0x50                         // random access + PCR
				mpegtsdWritePCR(packet, frame.dts-delay) // using 6 byte
				wpos += 8
			}

			// -----PES Header-----
			// packet_start_code_prefix
			// stream_id
			// PES_packet_length
			// '10'
			// PES_scrambling_control    0
			// PES_priority              0
			// data_alignment_indicator  0
			// copyright                 0
			// original_or_copy          0
			// PTS_DTS_flags
			// ESCR_flag                 0
			// ES_rate_flag              0
			// DSM_trick_mode_flag       0
			// additional_copy_info_flag 0
			// PES_CRC_flag              0
			// PES_extension_flag        0
			// PES_header_data_length
			packet[wpos] = 0x00        // packet_start_code_prefix
			packet[wpos+1] = 0x00      //
			packet[wpos+2] = 0x01      //
			packet[wpos+3] = frame.sid // stream_id
			wpos += 4

			headerSize := uint8(5)
			flags := uint8(0x80) // PTS
			if frame.dts != frame.pts {
				headerSize += 5
				flags |= 0x40 // DTS
			}

			pesSize := rpos + int(headerSize) + 3
			if pesSize > 0xFFFF {
				pesSize = 0
			}

			packet[wpos] = uint8(pesSize >> 8)     // PES_packet_length
			packet[wpos+1] = uint8(pesSize & 0xFF) //
			packet[wpos+2] = 0x80
			packet[wpos+3] = flags      // PTS and DTS flag
			packet[wpos+4] = headerSize // PES_header_data_length
			wpos += 5

			mpegtsWritePTS(packet[wpos:], flags>>6, frame.pts+delay)
			wpos += 5
			if frame.pts != frame.dts {
				mpegtsWritePTS(packet[wpos:], 1, frame.dts+delay)
				wpos += 5
			}

			first = false
		}

		bodySize := 188 - wpos // 当前TS packet，可写入大小
		inSize := rpos - lpos  // 整个帧剩余待打包大小

		if bodySize <= inSize {
			copy(packet[wpos:], b[lpos:lpos+inSize])
			lpos += bodySize
		} else {
			//stuffSize := bodySize - inSize // 当前TS packet的剩余空闲空间

			// 真实数据挪最后，中间用0xFF填充
			if packet[3]&0x20 != 0 {
				// has adaptation
				//base := 4 + packet[4]
			} else {
				// no adaptation
			}
		}
	}
}

func mpegtsdWritePCR(out []byte, pcr uint64) {
	out[0] = uint8(pcr >> 25)
	out[1] = uint8(pcr >> 17)
	out[2] = uint8(pcr >> 9)
	out[3] = uint8(pcr >> 1)
	out[4] = uint8(pcr<<7) | 0x7e
	out[5] = 0
}

// write PTS or DTS
func mpegtsWritePTS(out []byte, fb uint8, pts uint64) {
	var val uint64
	out[0] = (fb << 4) | (uint8(pts>>30) & 0x07) | 1

	val = (((pts >> 15) & 0x7FFF) << 1) | 1
	out[1] = uint8(val >> 8)
	out[2] = uint8(val)

	val = ((pts & 0x7FFF) << 1) | 1
	out[3] = uint8(val >> 8)
	out[4] = uint8(val)
}
