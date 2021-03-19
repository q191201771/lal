// Copyright 2019, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv

import "os"

type FLVFileWriter struct {
	fp *os.File
}

func (ffw *FLVFileWriter) Open(filename string) (err error) {
	ffw.fp, err = os.Create(filename)
	return
}

func (ffw *FLVFileWriter) WriteRaw(b []byte) (err error) {
	_, err = ffw.fp.Write(b)
	return
}

func (ffw *FLVFileWriter) WriteTag(tag Tag) (err error) {
	_, err = ffw.fp.Write(tag.Raw)
	return
}

func (ffw *FLVFileWriter) Dispose() {
	if ffw.fp != nil {
		_ = ffw.fp.Close()
	}
}
