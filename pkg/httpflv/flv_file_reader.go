// Copyright 2019, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
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
	fp               *os.File
	hasReadFLVHeader bool
}

func (ffr *FLVFileReader) Open(filename string) (err error) {
	ffr.fp, err = os.Open(filename)
	return
}

func (ffr *FLVFileReader) ReadFLVHeader() ([]byte, error) {
	ffr.hasReadFLVHeader = true

	flvHeader := make([]byte, flvHeaderSize)
	_, err := ffr.fp.Read(flvHeader)
	return flvHeader, err
}

func (ffr *FLVFileReader) ReadTag() (Tag, error) {
	// lazy read flv header
	if !ffr.hasReadFLVHeader {
		_, _ = ffr.ReadFLVHeader()
		ffr.hasReadFLVHeader = true
	}

	return readTag(ffr.fp)
}

func (ffr *FLVFileReader) Dispose() {
	if ffr.fp != nil {
		_ = ffr.fp.Close()
	}
}
