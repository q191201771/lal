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
)

type FragmentOP struct {
	fp     *os.File
	packet []byte //WriteFrame中缓存每个TS包数据
}

type mpegTSFrame struct {
	pts uint64
	dts uint64
	pid uint16
	sid uint8
	cc  uint8
	key bool // 关键帧
}

func (f *FragmentOP) OpenFile(filename string) (err error) {
	f.fp, err = os.Create(filename)
	if err != nil {
		return
	}
	f.writeFile(FixedFragmentHeader)
	//TS包固定188-byte
	f.packet = make([]byte, 188)
	return nil
}

func (f *FragmentOP) WriteFrame(frame *mpegTSFrame, b []byte) {
	//nazalog.Debugf("mpegts: pid=%d, sid=%d, pts=%d, dts=%d, key=%b, size=%d", frame.pid, frame.sid, frame.pts, frame.dts, frame.key, len(b))

	wpos := 0      // 当前packet的写入位置
	lpos := 0      // 当前帧的处理位置
	rpos := len(b) // 当前帧大小

	first := true // 是否为帧的首个packet的标准

	for lpos != rpos {
		wpos = 0
		frame.cc++

		// 每个packet都需要添加TS Header
		// -----TS Header----------------
		// sync_byte
		// transport_error_indicator    0
		// payload_unit_start_indicator
		// transport_priority           0
		// PID
		// transport_scrambling_control 0
		// adaptation_field_control
		// continuity_counter
		// ------------------------------
		f.packet[0] = syncByte // sync_byte
		f.packet[1] = 0x0
		if first {
			f.packet[1] = 0x40 // payload_unit_start_indicator
		}
		f.packet[1] |= uint8((frame.pid >> 8) & 0x1F) //PID高5位
		f.packet[2] = uint8(frame.pid & 0xFF)         //PID低8位

		// adaptation_field_control 先设置成无Adaptation
		// continuity_counter
		f.packet[3] = 0x10 | (frame.cc & 0x0f)
		wpos += 4

		if first {
			if frame.key {
				// 关键帧的首个packet需要添加Adaptation
				// -----Adaptation-----------------------
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
				// --------------------------------------
				f.packet[3] |= 0x20                            // adaptation_field_control 设置Adaptation
				f.packet[4] = 7                                // adaptation_field_length
				f.packet[5] = 0x50                             // random_access_indicator + PCR_flag
				mpegtsdWritePCR(f.packet[6:], frame.dts-delay) // using 6 byte
				wpos += 8
			}

			// 帧的首个packet需要添加PES Header
			// -----PES Header------------
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
			// ---------------------------
			f.packet[wpos] = 0x00        // packet_start_code_prefix 24-bits
			f.packet[wpos+1] = 0x00      //
			f.packet[wpos+2] = 0x01      //
			f.packet[wpos+3] = frame.sid // stream_id
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
				pesSize = 0
			}

			f.packet[wpos] = uint8(pesSize >> 8)     // PES_packet_length
			f.packet[wpos+1] = uint8(pesSize & 0xFF) //
			f.packet[wpos+2] = 0x80                  // 除了reserve的'10'，其他字段都是0
			f.packet[wpos+3] = flags                 // PTS/DTS flag
			f.packet[wpos+4] = headerSize            // PES_header_data_length: PTS+DTS数据长度
			wpos += 5

			// 写入PTS的值
			mpegtsWritePTS(f.packet[wpos:], flags>>6, frame.pts+delay)
			wpos += 5
			// 写入DTS的值
			if frame.pts != frame.dts {
				mpegtsWritePTS(f.packet[wpos:], 1, frame.dts+delay)
				wpos += 5
			}

			first = false
		}

		// 把帧的内容切割放入packet中
		bodySize := 188 - wpos // 当前TS packet，可写入大小
		inSize := rpos - lpos  // 整个帧剩余待打包大小

		if bodySize <= inSize {
			// 当前packet写不完这个帧，或者刚好够写完
			copy(f.packet[wpos:], b[lpos:lpos+inSize])
			lpos += bodySize
		} else {
			// 当前packet可以写完这个帧，并且还有空闲空间
			// 此时，真实数据挪最后，中间用0xFF填充到Adaptation中
			// 注意，此时有两种情况
			// 1. 原本有Adaptation
			// 2. 原本没有Adaptation

			stuffSize := bodySize - inSize // 当前TS packet的剩余空闲空间

			if f.packet[3]&0x20 != 0 {
				// has Adaptation

				base := int(4 + f.packet[4]) // TS Header + Adaptation
				if wpos > base {
					// 比如有PES Header

					copy(f.packet[base+stuffSize:], f.packet[base:wpos])
				}
				wpos = base + stuffSize

				f.packet[4] += uint8(stuffSize) // adaptation_field_length
				for i := 0; i < stuffSize; i++ {
					f.packet[base+i] = 0xFF
				}
			} else {
				// no Adaptation

				f.packet[3] |= 0x20

				base := 4
				if wpos > base {
					copy(f.packet[base+stuffSize:], f.packet[base:wpos])
				}
				wpos += stuffSize

				f.packet[4] = uint8(stuffSize - 1) // adaptation_field_length
				if stuffSize >= 2 {
					// TODO chef 这里是参考nginx rtmp module的实现，为什么这个字节写0而不是0xFF
					f.packet[5] = 0
					for i := 0; i < stuffSize-2; i++ {
						f.packet[6+i] = 0xFF
					}
				}
			}

			// 真实数据放在packet尾部
			copy(f.packet[wpos:], b[lpos:lpos+inSize])
			lpos = rpos
		}

		f.writeFile(f.packet)
	}
}

func (f *FragmentOP) CloseFile() {
	_ = f.fp.Close()
}

func (f *FragmentOP) writeFile(b []byte) {
	_, _ = f.fp.Write(b)
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
