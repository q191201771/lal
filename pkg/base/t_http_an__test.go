// Copyright 2023, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"github.com/q191201771/naza/pkg/assert"
	"testing"
)

//func ringBufToStr(ringBuf []RecordPerSec) string {
//	var buf strings.Builder
//	for idx, record := range ringBuf {
//		if record.UnixSec == 0 {
//			continue
//		}
//		record_str := fmt.Sprintf(" [%d]:{%d,%d}", idx, record.UnixSec, record.V)
//		buf.WriteString(record_str)
//	}
//	return buf.String()
//}

func TestPeriodRecord(t *testing.T) {
	funCmpRecord := func(index int, record *RecordPerSec, expect *RecordPerSec) {
		assert.Equal(t, record.UnixSec, expect.UnixSec)
		assert.Equal(t, record.V, expect.V)
	}

	records := NewPeriodRecord(16)
	statSess := StatGroup{}

	expected_fps := []RecordPerSec{
		{UnixSec: 1, V: 11},
		{UnixSec: 3, V: 13},
		{UnixSec: 9, V: 19},
		{UnixSec: 16, V: 26},
		{UnixSec: 17, V: 27},
		{UnixSec: 19, V: 29},
		{UnixSec: 25, V: 35},
		{UnixSec: 26, V: 36},
	}
	expected_n := 4
	for _, record := range expected_fps {
		records.Add(record.UnixSec, 1)
		records.Add(record.UnixSec, 2)
		records.Add(record.UnixSec, 3)
		records.Add(record.UnixSec, record.V-6)
	}
	nowUnixSec := int64(26)
	statSess.GetFpsFrom(&records, nowUnixSec)

	assert.Equal(t, expected_n, len(statSess.Fps))
	funCmpRecord(0, &statSess.Fps[0], &expected_fps[6])
	funCmpRecord(1, &statSess.Fps[1], &expected_fps[5])
	funCmpRecord(2, &statSess.Fps[2], &expected_fps[4])
	funCmpRecord(3, &statSess.Fps[3], &expected_fps[3])

	expected_fps_2nd := []RecordPerSec{
		{UnixSec: 0 + 16*2, V: 10 + 16*2},
		{UnixSec: 1 + 16*2, V: 11 + 16*2},
		{UnixSec: 3 + 16*2, V: 3 + 16*2},
		{UnixSec: 9 + 16*2, V: 9 + 16*2},
		{UnixSec: 11 + 16*2, V: 11 + 16*2},
	}
	expected_n = 5
	for _, record := range expected_fps_2nd {
		records.Add(record.UnixSec, record.V)
	}
	nowUnixSec = 11 + 16*2
	statSess.GetFpsFrom(&records, nowUnixSec)

	assert.Equal(t, expected_n, len(statSess.Fps))
	funCmpRecord(0, &statSess.Fps[0], &expected_fps_2nd[3])
	funCmpRecord(1, &statSess.Fps[1], &expected_fps_2nd[2])
	funCmpRecord(2, &statSess.Fps[2], &expected_fps_2nd[1])
	funCmpRecord(3, &statSess.Fps[3], &expected_fps_2nd[0])
	funCmpRecord(4, &statSess.Fps[4], &expected_fps[7])

	expected_fps = []RecordPerSec{
		{UnixSec: 0 + 16*3, V: 10 + 16*3},
		{UnixSec: 1 + 16*3, V: 11 + 16*3},
		{UnixSec: 3 + 16*3, V: 13 + 16*3},
	}
	expected_n = 6
	for _, record := range expected_fps {
		records.Add(record.UnixSec, record.V)
	}
	nowUnixSec = 3 + 16*3 + 1
	statSess.GetFpsFrom(&records, nowUnixSec)

	assert.Equal(t, expected_n, len(statSess.Fps))
	funCmpRecord(0, &statSess.Fps[0], &expected_fps[2])
	funCmpRecord(1, &statSess.Fps[1], &expected_fps[1])
	funCmpRecord(2, &statSess.Fps[2], &expected_fps[0])
	funCmpRecord(3, &statSess.Fps[3], &expected_fps_2nd[4])
	funCmpRecord(3, &statSess.Fps[4], &expected_fps_2nd[3])
}
