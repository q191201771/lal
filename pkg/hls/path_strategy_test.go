// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls_test

import (
	"testing"

	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/naza/pkg/assert"
)

func TestDefaultPathStrategy_GetRequestInfo(t *testing.T) {
	dps := &hls.DefaultPathStrategy{}
	rootOutPath := "/tmp/lal/hls/"

	golden := map[string]hls.RequestInfo{
		"/hls/test110.m3u8": {
			FileName:         "test110.m3u8",
			FileType:         "m3u8",
			StreamName:       "test110",
			FileNameWithPath: "/tmp/lal/hls/test110/playlist.m3u8",
		},
		"/hls/test110/playlist.m3u8": {
			FileName:         "playlist.m3u8",
			FileType:         "m3u8",
			StreamName:       "test110",
			FileNameWithPath: "/tmp/lal/hls/test110/playlist.m3u8",
		},
		"/hls/test110/record.m3u8": {
			FileName:         "record.m3u8",
			FileType:         "m3u8",
			StreamName:       "test110",
			FileNameWithPath: "/tmp/lal/hls/test110/record.m3u8",
		},
		"/hls/test110/test110-1620540712084-0.ts": {
			FileName:         "test110-1620540712084-0.ts",
			FileType:         "ts",
			StreamName:       "test110",
			FileNameWithPath: "/tmp/lal/hls/test110/test110-1620540712084-0.ts",
		},
		"/hls/test110-1620540712084-0.ts": {
			FileName:         "test110-1620540712084-0.ts",
			FileType:         "ts",
			StreamName:       "test110",
			FileNameWithPath: "/tmp/lal/hls/test110/test110-1620540712084-0.ts",
		},
	}

	for k, v := range golden {
		out := dps.GetRequestInfo(k, rootOutPath)
		assert.Equal(t, v, out)
	}
}
