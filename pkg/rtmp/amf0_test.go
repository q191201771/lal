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
	}
	err := AMF0.WriteObject(out, objs)
	assert.Equal(t, nil, err)
	v, _, err := AMF0.ReadObject(out.Bytes())
	assert.Equal(t, nil, err)
	assert.Equal(t, 2, len(v))
	assert.Equal(t, float64(3), v["air"])
	assert.Equal(t, "cat", v["ban"])
}

func TestAmf0_WriteNull_readNull(t *testing.T) {
	out := &bytes.Buffer{}
	err := AMF0.WriteNull(out)
	assert.Equal(t, nil, err)
	l, err := AMF0.ReadNull(out.Bytes())
	assert.Equal(t, 1, l)
	assert.Equal(t, nil, err)
}

// TODO chef: ReadStringWithoutType ReadLongStringWithoutType ReadBoolean
