// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import (
	"bytes"

	"github.com/cfeeling/lal/pkg/base"
)

func ParseMetadata(b []byte) (ObjectPairArray, error) {
	pos := 0
	v, l, err := AMF0.ReadString(b[pos:])
	if err != nil {
		return nil, err
	}
	pos += l
	if v == "@setDataFrame" {
		_, l, err = AMF0.ReadString(b[pos:])
		if err != nil {
			return nil, err
		}
		pos += l
	}
	opa, _, err := AMF0.ReadObjectOrArray(b[pos:])
	return opa, err
}

// spec-video_file_format_spec_v10.pdf
// onMetaData
// - duration        DOUBLE, seconds
// - width           DOUBLE
// - height          DOUBLE
// - videodatarate   DOUBLE
// - framerate       DOUBLE
// - videocodecid    DOUBLE
// - audiosamplerate DOUBLE
// - audiosamplesize DOUBLE
// - stereo          BOOL
// - audiocodecid    DOUBLE
// - filesize        DOUBLE, bytes
//
// 目前包含的字段：
// - width
// - height
// - audiocodecid
// - videocodecid
// - version
//
// width        如果为-1，则metadata中不写入该字段
// height       如果为-1，则metadata中不写入该字段
// audiocodecid 如果为-1，则metadata中不写入该字段
//                     AAC 10
// videocodecid 如果为-1，则metadata中不写入该字段
//                     H264 7
//                     H265 12
// @return 返回的内存块为新申请的独立内存块
func BuildMetadata(width int, height int, audiocodecid int, videocodecid int) ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := AMF0.WriteString(buf, "onMetaData"); err != nil {
		return nil, err
	}

	var opa ObjectPairArray
	if width != -1 {
		opa = append(opa, ObjectPair{
			Key:   "width",
			Value: width,
		})
	}
	if height != -1 {
		opa = append(opa, ObjectPair{
			Key:   "height",
			Value: height,
		})
	}
	if audiocodecid != -1 {
		opa = append(opa, ObjectPair{
			Key:   "audiocodecid",
			Value: audiocodecid,
		})
	}
	if videocodecid != -1 {
		opa = append(opa, ObjectPair{
			Key:   "videocodecid",
			Value: videocodecid,
		})
	}
	opa = append(opa, ObjectPair{
		Key:   "version",
		Value: base.LALRTMPBuildMetadataEncoder,
	})

	if err := AMF0.WriteObject(buf, opa); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
