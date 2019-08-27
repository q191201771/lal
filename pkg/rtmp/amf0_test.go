package rtmp

import (
	"bytes"
	"github.com/q191201771/lal/pkg/util/assert"
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
		assert.Equal(t, nil, err, "fxxk.")
		v, l, err := AMF0.ReadNumber(out.Bytes())
		assert.Equal(t, item, v, "fxxk.")
		assert.Equal(t, l, 9, "fxxk.")
		assert.Equal(t, nil, err, "fxxk.")
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
		assert.Equal(t, nil, err, "fxxk.")
		v, l, err := AMF0.ReadString(out.Bytes())
		assert.Equal(t, item, v, "fxxk.")
		assert.Equal(t, l, len(item)+3, "fxxk.")
		assert.Equal(t, nil, err, "fxxk.")
	}

	longStr := strings.Repeat("1", 65536)
	out := &bytes.Buffer{}
	err := AMF0.WriteString(out, longStr)
	assert.Equal(t, nil, err, "fxxk.")
	v, l, err := AMF0.ReadString(out.Bytes())
	assert.Equal(t, longStr, v, "fxxk.")
	assert.Equal(t, l, len(longStr)+5, "fxxk.")
	assert.Equal(t, nil, err, "fxxk.")
}

func TestAmf0_WriteObject_ReadObject(t *testing.T) {
	out := &bytes.Buffer{}
	objs := []ObjectPair{
		{"air", 3},
		{"ban", "cat"},
	}
	err := AMF0.WriteObject(out, objs)
	assert.Equal(t, nil, err, "fxxk.")
	v, _, err := AMF0.ReadObject(out.Bytes())
	assert.Equal(t, nil, err, "fxxk.")
	assert.Equal(t, 2, len(v), "fxxk.")
	assert.Equal(t, float64(3), v["air"], "fxxk.")
	assert.Equal(t, "cat", v["ban"], "fxxk.")
}

func TestAmf0_WriteNull_readNull(t *testing.T) {
	out := &bytes.Buffer{}
	err := AMF0.WriteNull(out)
	assert.Equal(t, nil, err, "fxxk.")
	l, err := AMF0.ReadNull(out.Bytes())
	assert.Equal(t, 1, l, "fxxk.")
	assert.Equal(t, nil, err, "fxxk.")
}

// TODO chef: ReadStringWithoutType ReadLongStringWithoutType ReadBoolean
