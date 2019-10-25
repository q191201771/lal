// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv

import (
	"os"
)

type FLVFileReader struct {
	fp *os.File
}

func (ffr *FLVFileReader) Open(filename string) (err error) {
	ffr.fp, err = os.Open(filename)
	return
}

func (ffr *FLVFileReader) ReadFLVHeader() ([]byte, error) {
	flvHeader := make([]byte, flvHeaderSize)
	_, err := ffr.fp.Read(flvHeader)
	return flvHeader, err
}

// TODO chef: 返回 Tag 类型，对比 bench
func (ffr *FLVFileReader) ReadTag() (*Tag, error) {
	return readTag(ffr.fp)
}

func (ffr *FLVFileReader) Dispose() {
	if ffr.fp != nil {
		_ = ffr.fp.Close()
	}
}
