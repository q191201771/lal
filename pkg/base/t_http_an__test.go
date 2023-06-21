// Copyright 2023, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"fmt"
	"strings"
	"testing"
)

func ringBufToStr(ringBuf []RecordPerSec) string {
	var buf strings.Builder
	for idx, record := range ringBuf {
		if record.UnixSec == 0 {
			continue
		}
		record_str := fmt.Sprintf(" [%d]:{%d,%d}", idx, record.UnixSec, record.V)
		buf.WriteString(record_str)
	}
	return buf.String()
}

func TestPeriodRecord(t *testing.T) {
	records := NewPeriodRecord(16)

	expected_fps := []RecordPerSec{
		{UnixSec: 1, V: 11},
		{UnixSec: 3, V: 13},
		{UnixSec: 9, V: 19},
		{UnixSec: 16, V: 26},
		{UnixSec: 17, V: 27}, // fpsRingBuf len 16, so replace 1
		{UnixSec: 19, V: 29}, // replace 3
		{UnixSec: 25, V: 35}, // replace 9
		{UnixSec: 26, V: 36},
	}
	expected_n := 5

	for _, record := range expected_fps {
		records.Add(record.UnixSec, 1)
		records.Add(record.UnixSec, 2)
		records.Add(record.UnixSec, 3)
		records.Add(record.UnixSec, record.V-6)
		t.Logf("records.nRecord:%d", records.nRecord)
	}

	if records.nRecord != expected_n {
		t.Fatalf("nFpsRecord not match, got:%d expect:%d, %s",
			records.nRecord, expected_n, ringBufToStr(records.ringBuf))
	}

	nowUnixSec := int64(26)
	expected_n = 4 // 26 not complete
	statSess := StatGroup{}
	statSess.GetFpsFrom(&records, nowUnixSec)
	// { 16, 17, 19, 25 }
	// 26 not complete
	if len(statSess.Fps) != expected_n {
		t.Fatalf("len(statSess.Fps) not match, got:%d expect:%d, %s",
			len(statSess.Fps), expected_n, ringBufToStr(statSess.Fps))
	}

	funCmpRecord := func(index int, record *RecordPerSec, expect *RecordPerSec) {
		if record.UnixSec != expect.UnixSec {
			t.Fatalf("index:%d UnixSec not match, got:%d expect:%d",
				index, record.UnixSec, expect.UnixSec)
		}
		if record.V != expect.V {
			t.Fatalf("index:%d V not match, got:%d expect:%d", index, record.V, expect.V)
		}
	}

	funCmpRecord(0, &statSess.Fps[0], &expected_fps[3])
	funCmpRecord(1, &statSess.Fps[1], &expected_fps[4])
	funCmpRecord(2, &statSess.Fps[2], &expected_fps[5])
	funCmpRecord(3, &statSess.Fps[3], &expected_fps[6])
	t.Log(ringBufToStr(statSess.Fps))

	t.Log("2nd period record test")

	expected_fps_2nd := []RecordPerSec{
		{UnixSec: 0 + 16*2, V: 10 + 16*2},
		{UnixSec: 1 + 16*2, V: 11 + 16*2},
		{UnixSec: 3 + 16*2, V: 3 + 16*2},
		{UnixSec: 9 + 16*2, V: 9 + 16*2},
		{UnixSec: 11 + 16*2, V: 11 + 16*2},
	}
	nowUnixSec = 11 + 16*2
	expected_n = 5
	// records = { 26 }
	for _, record := range expected_fps_2nd {
		records.Add(record.UnixSec, record.V)
	}
	statSess.GetFpsFrom(&records, nowUnixSec)

	// {32, 33, 35, 41, 26}
	// 11 + 16*2 not complete
	if len(statSess.Fps) != expected_n {
		t.Fatalf("len(statSess.Fps) not match, got:%d expect:%d, %s",
			len(statSess.Fps), expected_n, ringBufToStr(statSess.Fps))
	}
	funCmpRecord(0, &statSess.Fps[0], &expected_fps_2nd[0])
	funCmpRecord(1, &statSess.Fps[1], &expected_fps_2nd[1])
	funCmpRecord(2, &statSess.Fps[2], &expected_fps_2nd[2])
	funCmpRecord(3, &statSess.Fps[3], &expected_fps_2nd[3])
	funCmpRecord(4, &statSess.Fps[4], &expected_fps[7])

	t.Log(ringBufToStr(statSess.Fps))

	t.Log("3rd period record test")

	expected_fps = []RecordPerSec{
		{UnixSec: 0 + 16*3, V: 10 + 16*3},
		{UnixSec: 1 + 16*3, V: 11 + 16*3},
		{UnixSec: 3 + 16*3, V: 13 + 16*3},
	}
	nowUnixSec = 3 + 16*3 + 1
	expected_n = 4

	// records = { 11 + 16*2 }
	for _, record := range expected_fps {
		records.Add(record.UnixSec, record.V)
	}
	statSess.GetFpsFrom(&records, nowUnixSec)

	if len(statSess.Fps) != expected_n {
		t.Fatalf("len(statSess.Fps) not match, got:%d expect:%d, %s",
			len(statSess.Fps), expected_n, ringBufToStr(statSess.Fps))
	}

	funCmpRecord(0, &statSess.Fps[0], &expected_fps[0])
	funCmpRecord(1, &statSess.Fps[1], &expected_fps[1])
	funCmpRecord(2, &statSess.Fps[2], &expected_fps[2])
	funCmpRecord(3, &statSess.Fps[3], &expected_fps_2nd[4])
	t.Log(ringBufToStr(statSess.Fps))
}
