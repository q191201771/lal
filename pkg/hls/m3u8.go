// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
)

// @param content     需写入文件的内容
// @param filename    m3u8文件名
// @param filenameBak m3u8临时文件名
//
func writeM3U8File(content []byte, filename string, filenameBak string) error {
	var fp *os.File
	var err error
	if fp, err = os.Create(filenameBak); err != nil {
		return err
	}
	if _, err = fp.Write(content); err != nil {
		return err
	}
	if err = fp.Close(); err != nil {
		return err
	}
	if err = os.Rename(filenameBak, filename); err != nil {
		return err
	}
	return nil
}

// 如果当前duration比原m3u8文件的`EXT-X-TARGETDURATION`大，则更新`EXT-X-TARGETDURATION`的值
//
// @param content      原m3u8文件的内容
// @param currDuration 当前duration
//
// @return 处理后的m3u8文件内容
func updateTargetDurationInM3U8(content []byte, currDuration int) ([]byte, error) {
	l := bytes.Index(content, []byte("#EXT-X-TARGETDURATION:"))
	if l == -1 {
		return content, ErrHLS
	}
	r := bytes.Index(content[l:], []byte{'\n'})
	if r == -1 {
		return content, ErrHLS
	}
	oldDurationStr := bytes.TrimPrefix(content[l:l+r], []byte("#EXT-X-TARGETDURATION:"))
	oldDuration, err := strconv.Atoi(string(oldDurationStr))
	if err != nil {
		return content, err
	}
	if currDuration > oldDuration {
		tmpContent := make([]byte, l)
		copy(tmpContent, content[:l])
		tmpContent = append(tmpContent, []byte(fmt.Sprintf("#EXT-X-TARGETDURATION:%d", currDuration))...)
		tmpContent = append(tmpContent, content[l+r:]...)
		content = tmpContent
	}
	return content, nil
}
