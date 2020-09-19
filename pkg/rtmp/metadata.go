// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import (
	"bytes"

	"github.com/q191201771/lal/pkg/base"
)

// TODO chef: 见ServerSession::doDataMessageAMF0
func ParseMetadata(b []byte) (ObjectPairArray, error) {
	_, l, err := AMF0.ReadString(b)
	if err != nil {
		return nil, err
	}
	opa, _, err := AMF0.ReadObjectOrArray(b[l:])
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
// - encoder
// - server
// - author
// - version
//
// @param width        如果为-1，则metadata中不写入width
// @param height       如果为-1，则metadata中不写入height
// @param audiocodecid AAC 10
// @param videocodecid AVC 7
//
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
	opa = append(opa, ObjectPair{
		Key:   "audiocodecid",
		Value: audiocodecid,
	})
	opa = append(opa, ObjectPair{
		Key:   "videocodecid",
		Value: videocodecid,
	})

	opa = append(opa, ObjectPair{
		Key:   "version",
		Value: base.LALRTMPBuildMetadataEncoder,
	})

	if err := AMF0.WriteObject(buf, opa); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
