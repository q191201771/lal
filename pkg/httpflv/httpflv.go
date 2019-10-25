// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv

import (
	"errors"
	"strings"
)

type Writer interface {
	// TODO chef: return error
	WriteTag(tag *Tag)
}

var ErrHTTPFLV = errors.New("lal.httpflv: fxxk")

const (
	flvHeaderSize    = 13
	prevTagFieldSize = 4
)

var FLVHeader = []byte{0x46, 0x4c, 0x56, 0x01, 0x05, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00, 0x00}

type LineReader interface {
	ReadLine() (line []byte, isPrefix bool, err error)
}

// return 1st line and other headers with kv format
func parseHTTPHeader(r LineReader) (n int, firstLine string, headers map[string]string, err error) {
	headers = make(map[string]string)

	var line []byte
	var isPrefix bool
	line, isPrefix, err = r.ReadLine()
	if err != nil {
		return
	}
	if len(line) == 0 || isPrefix {
		err = ErrHTTPFLV
		return
	}
	firstLine = string(line)
	n += len(line)

	for {
		line, isPrefix, err = r.ReadLine()
		if len(line) == 0 { // 读到一个空的 \r\n 表示http头全部读取完毕了
			break
		}
		if isPrefix {
			err = ErrHTTPFLV
			return
		}
		if err != nil {
			return
		}
		l := string(line)
		n += len(l)
		pos := strings.Index(l, ":")
		if pos == -1 {
			err = ErrHTTPFLV
			return
		}
		headers[strings.Trim(l[0:pos], " ")] = strings.Trim(l[pos+1:], " ")
	}
	return
}
