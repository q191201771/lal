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

// TODO chef: 这个文件需要和session.go一起重构

type MPEGTSFrame struct {
	pts uint64
	dts uint64
	pid uint16
	sid uint8
	cc  uint8
	key bool
}

func mpegtsOpenFile(filename string) *os.File {
	fp, err := os.Create(filename)
	nazalog.Assert(nil, err)
	mpegtsWriteFile(fp, FixedFragmentHeader)
	return fp
}

func mpegtsWriteFrame(fp *os.File, frame *MPEGTSFrame, b []byte) {
	//nazalog.Debugf("mpegts: pid=%d, sid=%d, pts=%d, dts=%d, key=%b, size=%d", frame.pid, frame.sid, frame.pts, frame.dts, frame.key, len(b))

	wpos := 0      // 当前packet的写入位置
	lpos := 0      // 当前帧的处理位置
	rpos := len(b) // 当前帧大小

	first := true // 是否为帧的首个packet的标准

	for lpos != rpos {
		packet := make([]byte, 188)
		wpos = 0
		frame.cc++

		// 每个packet都需要添加TS Header
		// -----TS Header-----
		// sync_byte
		// transport_error_indicator    0
		// payload_unit_start_indicator
		// transport_priority           0
		// PID
		// transport_scrambling_control 0
		// adaptation_field_control
		// continuity_counter
		packet[0] = syncByte // sync_byte

		if first {
			packet[1] |= 0x40 // payload_unit_start_indicator
		}
		packet[1] |= uint8(frame.pid >> 8)  // PID
		packet[2] = uint8(frame.pid & 0xFF) //

		// adaptation_field_control 先设置成无Adaptation
		// continuity_counter
		packet[3] = 0x10 | (frame.cc & 0x0f)
		wpos += 4

		if first {
			if frame.key {
				// 关键帧的首个packet需要添加Adaptation
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
				packet[3] |= 0x20                            // adaptation_field_control 设置Adaptation
				packet[4] = 7                                // adaptation_field_length
				packet[5] = 0x50                             // random_access_indicator + PCR_flag
				mpegtsdWritePCR(packet[6:], frame.dts-delay) // using 6 byte
				wpos += 8
			}

			// 帧的首个packet需要添加PES Header
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

			// 计算PES Header中一些字段的值
			// PTS相关
			headerSize := uint8(5)
			flags := uint8(0x80)
			// DTS相关
			if frame.dts != frame.pts {
				headerSize += 5
				flags |= 0x40
			}

			pesSize := rpos + int(headerSize) + 3 // PES Header剩余3字节 + PTS/PTS长度 + 整个帧的长度
			if pesSize > 0xFFFF {
				nazalog.Warnf("pes size too large. pesSize=%d", pesSize)
				pesSize = 0
			}

			packet[wpos] = uint8(pesSize >> 8)     // PES_packet_length
			packet[wpos+1] = uint8(pesSize & 0xFF) //
			packet[wpos+2] = 0x80                  // 除了reserve的'10'，其他字段都是0
			packet[wpos+3] = flags                 // PTS/DTS flag
			packet[wpos+4] = headerSize            // PES_header_data_length
			wpos += 5

			// 写入PTS的值
			mpegtsWritePTS(packet[wpos:], flags>>6, frame.pts+delay)
			wpos += 5
			// 写入DTS的值
			if frame.pts != frame.dts {
				mpegtsWritePTS(packet[wpos:], 1, frame.dts+delay)
				wpos += 5
				//nazalog.Debugf("%d %d", (frame.pts)/90, (frame.dts)/90)
			}

			first = false
		}

		// 把帧的内容切割放入packet中
		bodySize := 188 - wpos // 当前TS packet，可写入大小
		inSize := rpos - lpos  // 整个帧剩余待打包大小

		if bodySize <= inSize {
			// 当前packet写不完这个帧，或者刚好够写完

			copy(packet[wpos:], b[lpos:lpos+inSize])
			lpos += bodySize
		} else {
			// 当前packet可以写完这个帧，并且还有空闲空间
			// 此时，真实数据挪最后，中间用0xFF填充到Adaptation中
			// 注意，此时有两种情况
			// 1. 原本有Adaptation
			// 2. 原本没有Adaptation

			stuffSize := bodySize - inSize // 当前TS packet的剩余空闲空间

			if packet[3]&0x20 != 0 {
				// has Adaptation

				base := int(4 + packet[4]) // TS Header + Adaptation
				if wpos > base {
					// 比如有PES Header

					copy(packet[base+stuffSize:], packet[base:wpos])
				}
				wpos = base + stuffSize

				packet[4] += uint8(stuffSize) // adaptation_field_length
				for i := 0; i < stuffSize; i++ {
					packet[base+i] = 0xFF
				}
			} else {
				// no Adaptation

				packet[3] |= 0x20 // 设置Adaptation

				base := 4
				if wpos > base {
					copy(packet[base+stuffSize:], packet[base:wpos])
				}
				wpos += stuffSize

				packet[4] = uint8(stuffSize - 1) // adaptation_field_length
				if stuffSize >= 2 {
					// TODO chef 这里是参考nginx rtmp module的实现，为什么这个字节写0而不是0xFF
					packet[5] = 0
					for i := 0; i < stuffSize-2; i++ {
						packet[6+i] = 0xFF
					}
				}
			}

			// 真实数据放在packet尾部
			copy(packet[wpos:], b[lpos:lpos+inSize])
			lpos = rpos
		}

		mpegtsWriteFile(fp, packet)
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

func mpegtsWriteFile(fp *os.File, b []byte) {
	_, _ = fp.Write(b)
}

func mpegtsCloseFile(fp *os.File) {
	_ = fp.Close()
}
