// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls_test

import (
	"runtime"
	"testing"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/naza/pkg/assert"
)

func TestDefaultPathStrategy_GetRequestInfo(t *testing.T) {
	testCases := []struct {
		name                    string
		url                     string
		wantStreamName          string
		wantFileNameWithPath    string
		wantWinFileNameWithPath string
	}{
		{
			name:                    "1 hls/[].m3u8 格式测试",
			url:                     "http://127.0.0.1:8080/hls/test1.m3u8",
			wantStreamName:          "test1",
			wantFileNameWithPath:    "/tmp/lal/hls/test1/playlist.m3u8",
			wantWinFileNameWithPath: "\\tmp\\lal\\hls\\test1\\playlist.m3u8",
		},
		{
			name:                    "2 hls/[]/playlist.m3u8 格式测试",
			url:                     "http://127.0.0.1:8080/hls/test2.m3u8",
			wantStreamName:          "test2",
			wantFileNameWithPath:    "/tmp/lal/hls/test2/playlist.m3u8",
			wantWinFileNameWithPath: "\\tmp\\lal\\hls\\test2\\playlist.m3u8",
		},
		{
			name:                    "3 hls/[]/record.m3u8 格式测试",
			url:                     "http://127.0.0.1:8080/hls/test3/record.m3u8",
			wantStreamName:          "test3",
			wantFileNameWithPath:    "/tmp/lal/hls/test3/record.m3u8",
			wantWinFileNameWithPath: "\\tmp\\lal\\hls\\test3\\record.m3u8",
		},
		{
			name:                    "4 hls/[]/[]-timestamp-seq.ts 格式测试",
			url:                     "http://127.0.0.1:8080/hls/test4/test4-1620540712084-0.ts",
			wantStreamName:          "test4",
			wantFileNameWithPath:    "/tmp/lal/hls/test4/test4-1620540712084-0.ts",
			wantWinFileNameWithPath: "\\tmp\\lal\\hls\\test4\\test4-1620540712084-0.ts",
		},
		{
			name:                    "5 hls/[]-timestamp-seq.ts 格式测试",
			url:                     "http://127.0.0.1:8080/hls/test5-1620540712084-0.ts",
			wantStreamName:          "test5",
			wantFileNameWithPath:    "/tmp/lal/hls/test5/test5-1620540712084-0.ts",
			wantWinFileNameWithPath: "\\tmp\\lal\\hls\\test5\\test5-1620540712084-0.ts",
		},
		{
			name:                    "6 hls/[]/[]-timestamp-seq.ts 名称带-符号格式测试",
			url:                     "http://127.0.0.1:8080/hls/test6/test6-0-1620540712084-0.ts",
			wantStreamName:          "test6-0",
			wantFileNameWithPath:    "/tmp/lal/hls/test6-0/test6-0-1620540712084-0.ts",
			wantWinFileNameWithPath: "\\tmp\\lal\\hls\\test6-0\\test6-0-1620540712084-0.ts",
		},
		{
			name:                    "7 hls/[]-timestamp-seq.ts 名称带-符号格式测试",
			url:                     "http://127.0.0.1:8080/hls/test7-0-1620540712084-0.ts",
			wantStreamName:          "test7-0",
			wantFileNameWithPath:    "/tmp/lal/hls/test7-0/test7-0-1620540712084-0.ts",
			wantWinFileNameWithPath: "\\tmp\\lal\\hls\\test7-0\\test7-0-1620540712084-0.ts",
		},
		{
			name:                    "8 hls/[]-timestamp-seq.ts 名称带中文测试",
			url:                     "http://127.0.0.1:8080/hls/中文测试-1620540712084-0.ts",
			wantStreamName:          "中文测试",
			wantFileNameWithPath:    "/tmp/lal/hls/中文测试/中文测试-1620540712084-0.ts",
			wantWinFileNameWithPath: "\\tmp\\lal\\hls\\中文测试\\中文测试-1620540712084-0.ts",
		},
		{
			name:                    "9 hls/[]-timestamp-seq.ts 名称带中文和-符号测试",
			url:                     "http://127.0.0.1:8080/hls/中文测试-zh-0-1620540712084-0.ts",
			wantStreamName:          "中文测试-zh-0",
			wantFileNameWithPath:    "/tmp/lal/hls/中文测试-zh-0/中文测试-zh-0-1620540712084-0.ts",
			wantWinFileNameWithPath: "\\tmp\\lal\\hls\\中文测试-zh-0\\中文测试-zh-0-1620540712084-0.ts",
		},
		{
			name:                    "10 hls/[]-timestamp.ts 非标准格式",
			url:                     "http://127.0.0.1:8080/hls/中文测试-1620540712084.ts",
			wantStreamName:          "中文测试-1620540712084.ts",
			wantFileNameWithPath:    "/tmp/lal/hls/中文测试-1620540712084.ts/中文测试-1620540712084.ts",
			wantWinFileNameWithPath: "\\tmp\\lal\\hls\\中文测试-1620540712084.ts\\中文测试-1620540712084.ts",
		},
	}

	dps := &hls.DefaultPathStrategy{}
	rootOutPath := "/tmp/lal/hls/"
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, err := base.ParseUrl(tc.url, -1)
			hls.Log.Assert(nil, err)
			out := dps.GetRequestInfo(ctx, rootOutPath)
			tmp := hls.RequestInfo{
				StreamName:       tc.wantStreamName,
				FileNameWithPath: tc.wantFileNameWithPath,
			}
			if runtime.GOOS == "windows" {
				tmp.FileNameWithPath = tc.wantWinFileNameWithPath
			}
			assert.Equal(t, tmp, out)
		})
	}
}
