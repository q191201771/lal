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

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/naza/pkg/assert"
)

func TestDefaultPathStrategy_GetRequestInfo(t *testing.T) {
	dps := &hls.DefaultPathStrategy{}
	rootOutPath := "/tmp/lal/hls/"

	golden := map[string]hls.RequestInfo{
		"http://127.0.0.1:8080/hls/test110.m3u8": {
			StreamName:       "test110",
			FileNameWithPath: "/tmp/lal/hls/test110/playlist.m3u8",
		},
		"http://127.0.0.1:8080/hls/test110/playlist.m3u8": {
			StreamName:       "test110",
			FileNameWithPath: "/tmp/lal/hls/test110/playlist.m3u8",
		},
		"http://127.0.0.1:8080/hls/test110/record.m3u8": {
			StreamName:       "test110",
			FileNameWithPath: "/tmp/lal/hls/test110/record.m3u8",
		},
		"http://127.0.0.1:8080/hls/test110/test110-1620540712084-0.ts": {
			StreamName:       "test110",
			FileNameWithPath: "/tmp/lal/hls/test110/test110-1620540712084-0.ts",
		},
		"http://127.0.0.1:8080/hls/test110-1620540712084-0.ts": {
			StreamName:       "test110",
			FileNameWithPath: "/tmp/lal/hls/test110/test110-1620540712084-0.ts",
		},
	}

	for k, v := range golden {
		ctx, err := base.ParseUrl(k, -1)
		hls.Log.Assert(nil, err)
		out := dps.GetRequestInfo(ctx, rootOutPath)
		assert.Equal(t, v, out)
	}
}
