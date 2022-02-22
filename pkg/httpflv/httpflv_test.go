// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/naza/pkg/assert"
	"github.com/q191201771/naza/pkg/mock"
	"github.com/q191201771/naza/pkg/nazamd5"

	"github.com/q191201771/lal/pkg/innertest"
)

func TestHttpflv(t *testing.T) {
	innertest.Entry(t)
}

func TestFlvFilePump(t *testing.T) {
	const flvFile = "../../testdata/test.flv"
	if _, err := os.Lstat(flvFile); err != nil {
		httpflv.Log.Warnf("lstat %s error. err=%+v", flvFile, err)
		return
	}

	var (
		//headers []byte
		tagCount  int
		allHeader []byte
		allRaw    []byte
	)

	httpflv.Clock = mock.NewFakeClock()
	defer func() {
		httpflv.Clock = mock.NewStdClock()
	}()

	ffp := httpflv.NewFlvFilePump()
	err := ffp.Pump(flvFile, func(tag httpflv.Tag) bool {
		tagCount++
		allRaw = append(allRaw, tag.Raw...)
		h := fmt.Sprintf("%+v", tag.Header)
		allHeader = append(allHeader, []byte(h)...)
		return true
	})
	assert.Equal(t, nil, err)
	assert.Equal(t, 1746, tagCount)
	assert.Equal(t, "ab7f75d2491711cc9a8d0ccd5d56280b", nazamd5.Md5(allRaw))
	assert.Equal(t, "2a1cd1bd99f725c19bbd45d81d436e59", nazamd5.Md5(allHeader))
}
