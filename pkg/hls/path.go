// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import (
	"fmt"
	"io/ioutil"
	"strings"
)

// 本文件聚合以下功能：
// - 生成HLS（m3u8文件+ts文件）时，文件命名规则，以及文件存放规则
// - HTTP请求HLS时，request URI和文件路径的映射规则

// HTTP请求URI格式，已经文件路径的映射规则
//
// 假设
// 流名称="test110"
// rootPath="/tmp/lal/hls/"
//
// 则
// http://127.0.0.1:8081/hls/test110/playlist.m3u8  -> /tmp/lal/hls/test110/playlist.m3u8
// http://127.0.0.1:8081/hls/test110/record.m3u8    -> /tmp/lal/hls/test110/record.m3u8
// http://127.0.0.1:8081/hls/test110/timestamp-0.ts -> /tmp/lal/hls/test110/timestamp-0.ts

type requestInfo struct {
	fileName   string
	streamName string
	fileType   string
}

// RequestURI example:
// uri                                              -> fileName       streamName fileType
// http://127.0.0.1:8081/hls/test110/playlist.m3u8  -> playlist.m3u8  test110    m3u8
// http://127.0.0.1:8081/hls/test110/record.m3u8    -> record.m3u8    test110    m3u8
// http://127.0.0.1:8081/hls/test110/timestamp-0.ts -> timestamp-0.ts test110    ts
func parseRequestInfo(uri string) (ri requestInfo) {
	ss := strings.Split(uri, "/")
	if len(ss) < 2 {
		return
	}
	ri.streamName = ss[len(ss)-2]
	ri.fileName = ss[len(ss)-1]

	ss = strings.Split(ri.fileName, ".")
	if len(ss) < 2 {
		return
	}
	ri.fileType = ss[len(ss)-1]

	return
}

func readFileContent(rootOutPath string, ri requestInfo) ([]byte, error) {
	filename := fmt.Sprintf("%s%s/%s", rootOutPath, ri.streamName, ri.fileName)
	return ioutil.ReadFile(filename)
}

func getMuxerOutPath(rootOutPath string, streamName string) string {
	return fmt.Sprintf("%s%s/", rootOutPath, streamName)
}

func getM3U8Filename(outpath string, streamName string) string {
	return fmt.Sprintf("%s%s.m3u8", outpath, "playlist")
}

func getRecordM3U8Filename(outpath string, streamName string) string {
	return fmt.Sprintf("%s%s.m3u8", outpath, "record")
}

func getTSFilenameWithPath(outpath string, filename string) string {
	return fmt.Sprintf("%s%s", outpath, filename)
}

func getTSFilename(streamName string, id int, timestamp int) string {
	return fmt.Sprintf("%d-%d.ts", timestamp, id)
}
