// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import (
	"os"

	"github.com/q191201771/lal/pkg/mpegts"
)

type Fragment struct {
	fp *os.File
}

func (f *Fragment) OpenFile(filename string) (err error) {
	f.fp, err = os.Create(filename)
	if err != nil {
		return
	}
	f.writeFile(mpegts.FixedFragmentHeader)
	return nil
}

func (f *Fragment) WriteFrame(frame *mpegts.Frame) {
	mpegts.PackTSPacket(frame, func(packet []byte, cc uint8) {
		f.writeFile(packet)
	})
}

func (f *Fragment) CloseFile() {
	_ = f.fp.Close()
}

func (f *Fragment) writeFile(b []byte) {
	_, _ = f.fp.Write(b)
}
