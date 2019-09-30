package rtmp_test

import (
	"bytes"
	. "github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/assert"
	"github.com/q191201771/naza/pkg/mockwriter"
	"strings"
	"testing"
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
		err := AMF0.WriteNumber(out, item)
		assert.Equal(t, nil, err)
		v, l, err := AMF0.ReadNumber(out.Bytes())
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
		err := AMF0.WriteString(out, item)
		assert.Equal(t, nil, err)
		v, l, err := AMF0.ReadString(out.Bytes())
		assert.Equal(t, item, v)
		assert.Equal(t, l, len(item)+3)
		assert.Equal(t, nil, err)
	}

	longStr := strings.Repeat("1", 65536)
	out := &bytes.Buffer{}
	err := AMF0.WriteString(out, longStr)
	assert.Equal(t, nil, err)
	v, l, err := AMF0.ReadString(out.Bytes())
	assert.Equal(t, longStr, v)
	assert.Equal(t, l, len(longStr)+5)
	assert.Equal(t, nil, err)
}

func TestAmf0_WriteObject_ReadObject(t *testing.T) {
	out := &bytes.Buffer{}
	objs := []ObjectPair{
		{"air", 3},
		{"ban", "cat"},
		{"dog", true},
	}
	err := AMF0.WriteObject(out, objs)
	assert.Equal(t, nil, err)
	v, _, err := AMF0.ReadObject(out.Bytes())
	assert.Equal(t, nil, err)
	assert.Equal(t, 3, len(v))
	assert.Equal(t, float64(3), v["air"])
	assert.Equal(t, "cat", v["ban"])
	assert.Equal(t, true, v["dog"])
}

func TestAmf0_WriteNull_readNull(t *testing.T) {
	out := &bytes.Buffer{}
	err := AMF0.WriteNull(out)
	assert.Equal(t, nil, err)
	l, err := AMF0.ReadNull(out.Bytes())
	assert.Equal(t, 1, l)
	assert.Equal(t, nil, err)
}

func TestAmf0_WriteBoolean_ReadBoolean(t *testing.T) {
	cases := []bool{true, false}

	for i := range cases {
		out := &bytes.Buffer{}
		err := AMF0.WriteBoolean(out, cases[i])
		assert.Equal(t, nil, err)
		v, l, err := AMF0.ReadBoolean(out.Bytes())
		assert.Equal(t, cases[i], v)
		assert.Equal(t, 2, l)
		assert.Equal(t, nil, err)
	}
}

func TestAMF0Corner(t *testing.T) {
	var (
		mw     *mockwriter.MockWriter
		err    error
		b      []byte
		str    string
		l      int
		num    float64
		flag   bool
		objs   []ObjectPair
		objMap map[string]interface{}
	)

	mw = mockwriter.NewMockWriter(mockwriter.WriterTypeReturnError)
	err = AMF0.WriteNumber(mw, 0)
	assert.IsNotNil(t, err)

	mw = mockwriter.NewMockWriter(mockwriter.WriterTypeReturnError)
	err = AMF0.WriteBoolean(mw, true)
	assert.IsNotNil(t, err)

	// WriteString 调用 三次写
	mw = mockwriter.NewMockWriter(mockwriter.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]mockwriter.WriterType{0: mockwriter.WriterTypeReturnError})
	err = AMF0.WriteString(mw, "0")
	assert.IsNotNil(t, err)
	mw = mockwriter.NewMockWriter(mockwriter.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]mockwriter.WriterType{1: mockwriter.WriterTypeReturnError})
	err = AMF0.WriteString(mw, "1")
	assert.IsNotNil(t, err)
	mw = mockwriter.NewMockWriter(mockwriter.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]mockwriter.WriterType{2: mockwriter.WriterTypeReturnError})
	err = AMF0.WriteString(mw, "2")
	assert.IsNotNil(t, err)
	longStr := strings.Repeat("1", 65536)
	mw = mockwriter.NewMockWriter(mockwriter.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]mockwriter.WriterType{0: mockwriter.WriterTypeReturnError})
	err = AMF0.WriteString(mw, longStr)
	assert.IsNotNil(t, err)
	mw = mockwriter.NewMockWriter(mockwriter.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]mockwriter.WriterType{1: mockwriter.WriterTypeReturnError})
	err = AMF0.WriteString(mw, longStr)
	assert.IsNotNil(t, err)
	mw = mockwriter.NewMockWriter(mockwriter.WriterTypeDoNothing)
	mw.SetSpecificType(map[uint32]mockwriter.WriterType{2: mockwriter.WriterTypeReturnError})
	err = AMF0.WriteString(mw, longStr)
	assert.IsNotNil(t, err)

	objs = []ObjectPair{
		{"air", 3},
		{"ban", "cat"},
		{"dog", true},
	}
	for i := uint32(0); i < 14; i++ {
		mw = mockwriter.NewMockWriter(mockwriter.WriterTypeDoNothing)
		mw.SetSpecificType(map[uint32]mockwriter.WriterType{i: mockwriter.WriterTypeReturnError})
		err = AMF0.WriteObject(mw, objs)
		assert.IsNotNil(t, err)
	}

	b = nil
	str, l, err = AMF0.ReadStringWithoutType(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAMFTooShort, err)
	b = []byte{1, 1}
	str, l, err = AMF0.ReadStringWithoutType(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAMFTooShort, err)

	b = nil
	str, l, err = AMF0.ReadLongStringWithoutType(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAMFTooShort, err)
	b = []byte{1, 1, 1, 1}
	str, l, err = AMF0.ReadLongStringWithoutType(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAMFTooShort, err)

	b = nil
	str, l, err = AMF0.ReadString(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAMFTooShort, err)
	b = []byte{1}
	str, l, err = AMF0.ReadString(b)
	assert.Equal(t, str, "")
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAMFInvalidType, err)

	b = nil
	num, l, err = AMF0.ReadNumber(b)
	assert.Equal(t, int(num), 0)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAMFTooShort, err)
	str = strings.Repeat("1", 16)
	b = []byte(str)
	num, l, err = AMF0.ReadNumber(b)
	assert.Equal(t, int(num), 0)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAMFInvalidType, err)

	b = nil
	flag, l, err = AMF0.ReadBoolean(b)
	assert.Equal(t, flag, false)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAMFTooShort, err)
	b = []byte{0, 0}
	flag, l, err = AMF0.ReadBoolean(b)
	assert.Equal(t, flag, false)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAMFInvalidType, err)

	b = nil
	l, err = AMF0.ReadNull(b)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAMFTooShort, err)
	b = []byte{0}
	l, err = AMF0.ReadNull(b)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAMFInvalidType, err)

	b = nil
	objMap, l, err = AMF0.ReadObject(b)
	assert.Equal(t, nil, objMap)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAMFTooShort, err)
	b = []byte{0}
	objMap, l, err = AMF0.ReadObject(b)
	assert.Equal(t, nil, objMap)
	assert.Equal(t, 0, l)
	assert.Equal(t, ErrAMFInvalidType, err)

	defer func() {
		recover()
	}()
	objs = []ObjectPair{
		{"key", []byte{1}},
	}
	_ = AMF0.WriteObject(mw, objs)
}

func BenchmarkAmf0_ReadObject(b *testing.B) {
	out := &bytes.Buffer{}
	objs := []ObjectPair{
		{"air", 3},
		{"ban", "cat"},
		{"dog", true},
	}
	_ = AMF0.WriteObject(out, objs)
	for i := 0; i < b.N; i++ {
		_, _, _ = AMF0.ReadObject(out.Bytes())
	}
}

func BenchmarkAmf0_WriteObject(b *testing.B) {
	out := &bytes.Buffer{}
	objs := []ObjectPair{
		{"air", 3},
		{"ban", "cat"},
		{"dog", true},
	}
	for i := 0; i < b.N; i++ {
		_ = AMF0.WriteObject(out, objs)
	}
}
