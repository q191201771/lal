// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp_test

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/q191201771/naza/pkg/nazalog"

	. "github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/assert"
	"github.com/q191201771/naza/pkg/fake"
)

func TestAmf0_WriteNumber_ReadNumber(t *testing.T) {
	cases := []float64{
		0,
		1,
		0xff,
		1.2,
	}

	for _, item := range cases {
		out := &bytes.Buffer{}
		err := Amf0.WriteNumber(out, item)
		assert.Equal(t, nil, err)
		v, l, err := Amf0.ReadNumber(out.Bytes())
		assert.Equal(t, item, v)
		assert.Equal(t, l, 9)
		assert.Equal(t, nil, err)
	}
}

func TestAmf0_WriteString_ReadString(t *testing.T) {
	cases := []string{
		"a",
		"ab",
		"111",
		"~!@#$%^&*()_+",
	}
	for _, item := range cases {
		out := &bytes.Buffer{}
		err := Amf0.WriteString(out, item)
		assert.Equal(t, nil, err)
		v, l, err := Amf0.ReadString(out.Bytes())
		assert.Equal(t, item, v)
		assert.Equal(t, l, len(item)+3)
		assert.Equal(t, nil, err)
	}

	longStr := strings.Repeat("1", 65536)
	out := &bytes.Buffer{}
	err := Amf0.WriteString(out, longStr)
	assert.Equal(t, nil, err)
	v, l, err := Amf0.ReadString(out.Bytes())
	assert.Equal(t, longStr, v)
	assert.Equal(t, l, len(longStr)+5)
	assert.Equal(t, nil, err)
}

func TestAmf0_WriteObject_ReadObject(t *testing.T) {
	out := &bytes.Buffer{}
	objs := []ObjectPair{
		{Key: "air", Value: 3},
		{Key: "ban", Value: "cat"},
		{Key: "dog", Value: true},
	}
	err := Amf0.WriteObject(out, objs)
	assert.Equal(t, nil, err)
	v, _, err := Amf0.ReadObject(out.Bytes())
	assert.Equal(t, nil, err)
	assert.Equal(t, 3, len(v))
	assert.Equal(t, float64(3), v.Find("air"))
	assert.Equal(t, "cat", v.Find("ban"))
	assert.Equal(t, true, v.Find("dog"))
}

func TestAmf0_WriteNull_readNull(t *testing.T) {
	out := &bytes.Buffer{}
	err := Amf0.WriteNull(out)
	assert.Equal(t, nil, err)
	l, err := Amf0.ReadNull(out.Bytes())
	assert.Equal(t, 1, l)
	assert.Equal(t, nil, err)
}

func TestAmf0_WriteBoolean_ReadBoolean(t *testing.T) {
	cases := []bool{true, false}

	for i := range cases {
		out := &bytes.Buffer{}
		err := Amf0.WriteBoolean(out, cases[i])
		assert.Equal(t, nil, err)
		v, l, err := Amf0.ReadBoolean(out.Bytes())
		assert.Equal(t, cases[i], v)
		assert.Equal(t, 2, l)
		assert.Equal(t, nil, err)
	}
}

func TestAmf0_ReadArray(t *testing.T) {
	gold := []byte{0x08, 0x00, 0x00, 0x00, 0x10, 0x00, 0x08, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x05, 0x77, 0x69, 0x64, 0x74, 0x68, 0x00, 0x40, 0x88, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06, 0x68, 0x65, 0x69, 0x67, 0x68, 0x74, 0x00, 0x40, 0x74, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0d, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x64, 0x61, 0x74, 0x61, 0x72, 0x61, 0x74, 0x65, 0x00, 0x40, 0x69, 0xe8, 0x50, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09, 0x66, 0x72, 0x61, 0x6d, 0x65, 0x72, 0x61, 0x74, 0x65, 0x00, 0x40, 0x39, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0c, 0x76, 0x69, 0x64, 0x65, 0x6f, 0x63, 0x6f, 0x64, 0x65, 0x63, 0x69, 0x64, 0x00, 0x40, 0x1c, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0d, 0x61, 0x75, 0x64, 0x69, 0x6f, 0x64, 0x61, 0x74, 0x61, 0x72, 0x61, 0x74, 0x65, 0x00, 0x40, 0x3d, 0x54, 0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0f, 0x61, 0x75, 0x64, 0x69, 0x6f, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x72, 0x61, 0x74, 0x65, 0x00, 0x40, 0xe5, 0x88, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0f, 0x61, 0x75, 0x64, 0x69, 0x6f, 0x73, 0x61, 0x6d, 0x70, 0x6c, 0x65, 0x73, 0x69, 0x7a, 0x65, 0x00, 0x40, 0x30, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x06, 0x73, 0x74, 0x65, 0x72, 0x65, 0x6f, 0x01, 0x01, 0x00, 0x0c, 0x61, 0x75, 0x64, 0x69, 0x6f, 0x63, 0x6f, 0x64, 0x65, 0x63, 0x69, 0x64, 0x00, 0x40, 0x24, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0b, 0x6d, 0x61, 0x6a, 0x6f, 0x72, 0x5f, 0x62, 0x72, 0x61, 0x6e, 0x64, 0x02, 0x00, 0x04, 0x69, 0x73, 0x6f, 0x6d, 0x00, 0x0d, 0x6d, 0x69, 0x6e, 0x6f, 0x72, 0x5f, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x02, 0x00, 0x03, 0x35, 0x31, 0x32, 0x00, 0x11, 0x63, 0x6f, 0x6d, 0x70, 0x61, 0x74, 0x69, 0x62, 0x6c, 0x65, 0x5f, 0x62, 0x72, 0x61, 0x6e, 0x64, 0x73, 0x02, 0x00, 0x10, 0x69, 0x73, 0x6f, 0x6d, 0x69, 0x73, 0x6f, 0x32, 0x61, 0x76, 0x63, 0x31, 0x6d, 0x70, 0x34, 0x31, 0x00, 0x07, 0x65, 0x6e, 0x63, 0x6f, 0x64, 0x65, 0x72, 0x02, 0x00, 0x0d, 0x4c, 0x61, 0x76, 0x66, 0x35, 0x37, 0x2e, 0x38, 0x33, 0x2e, 0x31, 0x30, 0x30, 0x00, 0x08, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x69, 0x7a, 0x65, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x09}

	ops, l, err := Amf0.ReadArray(gold)
	assert.Equal(t, nil, err)
	assert.Equal(t, 16, len(ops))
	assert.Equal(t, 359, len(gold))
	assert.Equal(t, 359, l)
	nazalog.Debug(ops)
}

func TestAmf0_ReadCase1(t *testing.T) {
	// ZLMediaKit connect result的object中存在null type
	// https://github.com/q191201771/lal/issues/102
	//
	gold := "030000000000b614000000000200075f726573756c74003ff000000000000003000c6361706162696c697469657300403f0000000000000006666d7356657202000d464d532f332c302c312c313233000009030004636f646502001d4e6574436f6e6e656374696f6e2e436f6e6e6563742e53756363657373000b6465736372697074696f6e020015436f6e6e656374696f6e207375636365656465642e00056c6576656c020006737461747573000e6f626a656374456e636f64696e6705000009"
	goldbytes, err := hex.DecodeString(gold)
	assert.Equal(t, nil, err)
	index := 12

	s, l, err := Amf0.ReadString(goldbytes[index:])
	assert.Equal(t, nil, err)
	assert.Equal(t, "_result", s)
	index += l

	n, l, err := Amf0.ReadNumber(goldbytes[index:])
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(1), n)
	index += l

	o, l, err := Amf0.ReadObject(goldbytes[index:])
	assert.Equal(t, nil, err)
	i, err := o.FindNumber("capabilities")
	assert.Equal(t, nil, err)
	assert.Equal(t, 31, i)
	s, err = o.FindString("fmsVer")
	assert.Equal(t, nil, err)
	assert.Equal(t, "FMS/3,0,1,123", s)
	index += l

	o, l, err = Amf0.ReadObject(goldbytes[index:])
	assert.Equal(t, nil, err)
	s, err = o.FindString("code")
	assert.Equal(t, nil, err)
	assert.Equal(t, "NetConnection.Connect.Success", s)
	s, err = o.FindString("description")
	assert.Equal(t, nil, err)
	assert.Equal(t, "Connection succeeded.", s)
	s, err = o.FindString("level")
	assert.Equal(t, nil, err)
	assert.Equal(t, "status", s)
	index += l

	assert.Equal(t, len(goldbytes), index)
}

func TestAmf0Corner(t *testing.T) {
	var (
		mw   *fake.Writer
		err  error
		b    []byte
		str  string
		l    int
		num  float64
		flag bool
		objs []ObjectPair
	)

	mw = fake.NewWriter(fake.WriterTypeReturnError)
	err = Amf0.WriteNumber(mw, 0)
	assert.IsNotNil(t, err)

	mw = fake.NewWriter(fake.WriterTypeReturnError)
	err = Amf0.WriteBoolean(mw, true)
	assert.IsNotNil(t, err)

	// WriteString 调用 三次写
	mw = fake.NewWriter(fake.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]fake.WriterType{0: fake.WriterTypeReturnError})
	err = Amf0.WriteString(mw, "0")
	assert.IsNotNil(t, err)
	mw = fake.NewWriter(fake.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]fake.WriterType{1: fake.WriterTypeReturnError})
	err = Amf0.WriteString(mw, "1")
	assert.IsNotNil(t, err)
	mw = fake.NewWriter(fake.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]fake.WriterType{2: fake.WriterTypeReturnError})
	err = Amf0.WriteString(mw, "2")
	assert.IsNotNil(t, err)
	longStr := strings.Repeat("1", 65536)
	mw = fake.NewWriter(fake.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]fake.WriterType{0: fake.WriterTypeReturnError})
	err = Amf0.WriteString(mw, longStr)
	assert.IsNotNil(t, err)
	mw = fake.NewWriter(fake.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]fake.WriterType{1: fake.WriterTypeReturnError})
	err = Amf0.WriteString(mw, longStr)
	assert.IsNotNil(t, err)
	mw = fake.NewWriter(fake.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]fake.WriterType{2: fake.WriterTypeReturnError})
	err = Amf0.WriteString(mw, longStr)
	assert.IsNotNil(t, err)

	objs = []ObjectPair{
		{Key: "air", Value: 3},
		{Key: "ban", Value: "cat"},
		{Key: "dog", Value: true},
	}
	for i := uint32(0); i < 14; i++ {
		mw = fake.NewWriter(fake.WriterTypeDoNothing)
		mw.SetSpecificType(map[uint32]fake.WriterType{i: fake.WriterTypeReturnError})
		err = Amf0.WriteObject(mw, objs)
		assert.IsNotNil(t, err)
	}

	b = nil
	str, l, err = Amf0.ReadStringWithoutType(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAmfTooShort, err)
	b = []byte{1, 1}
	str, l, err = Amf0.ReadStringWithoutType(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAmfTooShort, err)

	b = nil
	str, l, err = Amf0.ReadLongStringWithoutType(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAmfTooShort, err)
	b = []byte{1, 1, 1, 1}
	str, l, err = Amf0.ReadLongStringWithoutType(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAmfTooShort, err)

	b = nil
	str, l, err = Amf0.ReadString(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAmfTooShort, err)
	b = []byte{1}
	str, l, err = Amf0.ReadString(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAmfInvalidType, err)

	b = nil
	num, l, err = Amf0.ReadNumber(b)
	assert.Equal(t, int(num), 0)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAmfTooShort, err)
	str = strings.Repeat("1", 16)
	b = []byte(str)
	num, l, err = Amf0.ReadNumber(b)
	assert.Equal(t, int(num), 0)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAmfInvalidType, err)

	b = nil
	flag, l, err = Amf0.ReadBoolean(b)
	assert.Equal(t, flag, false)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAmfTooShort, err)
	b = []byte{0, 0}
	flag, l, err = Amf0.ReadBoolean(b)
	assert.Equal(t, flag, false)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAmfInvalidType, err)

	b = nil
	l, err = Amf0.ReadNull(b)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAmfTooShort, err)
	b = []byte{0}
	l, err = Amf0.ReadNull(b)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAmfInvalidType, err)

	b = nil
	objs, l, err = Amf0.ReadObject(b)
	assert.Equal(t, nil, objs)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAmfTooShort, err)
	b = []byte{0}
	objs, l, err = Amf0.ReadObject(b)
	assert.Equal(t, nil, objs)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAmfInvalidType, err)

	defer func() {
		recover()
	}()
	objs = []ObjectPair{
		{Key: "key", Value: []byte{1}},
	}
	_ = Amf0.WriteObject(mw, objs)
}

func BenchmarkAmf0_ReadObject(b *testing.B) {
	out := &bytes.Buffer{}
	objs := []ObjectPair{
		{Key: "air", Value: 3},
		{Key: "ban", Value: "cat"},
		{Key: "dog", Value: true},
	}
	_ = Amf0.WriteObject(out, objs)
	for i := 0; i < b.N; i++ {
		_, _, _ = Amf0.ReadObject(out.Bytes())
	}
}

func BenchmarkAmf0_WriteObject(b *testing.B) {
	out := &bytes.Buffer{}
	objs := []ObjectPair{
		{Key: "air", Value: 3},
		{Key: "ban", Value: "cat"},
		{Key: "dog", Value: true},
	}
	for i := 0; i < b.N; i++ {
		_ = Amf0.WriteObject(out, objs)
	}
}
