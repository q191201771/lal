// Copyright 2023, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base_test

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/nazalog"
	"io"
	"testing"
)

func TestDumpFile_WriteWithType(t *testing.T) {
	df := base.NewDumpFile()
	err := df.OpenToWrite("/tmp/test.laldump")
	nazalog.Assert(nil, err)
	err = df.WriteWithType([]byte("hello"), base.DumpTypePsRtpData)
	nazalog.Assert(nil, err)
	err = df.Close()
	nazalog.Assert(nil, err)
}

func TestDumpFile_OpenToRead(t *testing.T) {
	df := base.NewDumpFile()
	err := df.OpenToRead("/tmp/test.laldump")
	nazalog.Assert(nil, err)
	for {
		m, err := df.ReadOneMessage()
		if err == io.EOF {
			break
		}
		nazalog.Assert(nil, err)
		nazalog.Debugf("%+v", m)
	}
}
