// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv

import "os"

// TODO chef: 结构体重命名为FileWriter，文件名重命名为file_writer.go。所有写流文件的（flv,hls,ts）统一重构

type FLVFileWriter struct {
	fp *os.File
}

func (ffw *FLVFileWriter) Open(filename string) (err error) {
	ffw.fp, err = os.Create(filename)
	return
}

func (ffw *FLVFileWriter) WriteRaw(b []byte) (err error) {
	if ffw.fp == nil {
		return ErrHTTPFLV
	}
	_, err = ffw.fp.Write(b)
	return
}

func (ffw *FLVFileWriter) WriteFLVHeader() (err error) {
	if ffw.fp == nil {
		return ErrHTTPFLV
	}
	_, err = ffw.fp.Write(FLVHeader)
	return
}

func (ffw *FLVFileWriter) WriteTag(tag Tag) (err error) {
	if ffw.fp == nil {
		return ErrHTTPFLV
	}
	_, err = ffw.fp.Write(tag.Raw)
	return
}

func (ffw *FLVFileWriter) Dispose() error {
	if ffw.fp == nil {
		return ErrHTTPFLV
	}
	return ffw.fp.Close()
}

func (ffw *FLVFileWriter) Name() string {
	if ffw.fp == nil {
		return ""
	}
	return ffw.fp.Name()
}
