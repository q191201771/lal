package bele

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBEUint16(t *testing.T) {
	vector := []struct {
		input  []byte
		output uint16
	}{
		{input: []byte{0, 0}, output: 0},
		{input: []byte{0, 1}, output: 1},
		{input: []byte{0, 255}, output: 255},
		{input: []byte{1, 0}, output: 256},
		{input: []byte{255, 0}, output: 255 * 256},
		{input: []byte{12, 34}, output: 12*256 + 34},
	}

	for i := 0; i < len(vector); i++ {
		assert.Equal(t, vector[i].output, BEUint16(vector[i].input), "fxxk.")
	}
}

func TestBEUint24(t *testing.T) {
	vector := []struct {
		input  []byte
		output uint32
	}{
		{input: []byte{0, 0, 0}, output: 0},
		{input: []byte{0, 0, 1}, output: 1},
		{input: []byte{0, 1, 0}, output: 256},
		{input: []byte{1, 0, 0}, output: 1 * 256 * 256},
		{input: []byte{12, 34, 56}, output: 12*256*256 + 34*256 + 56},
	}

	for i := 0; i < len(vector); i++ {
		assert.Equal(t, vector[i].output, BEUint24(vector[i].input), "fxxk.")
	}
}

type DummyWriter struct {
}

func (w DummyWriter) Write(b []byte) (int, error) {
	return 0, nil
}

func BenchmarkWriteBEUint24(b *testing.B) {
	var w DummyWriter
	for i := 0; i < b.N; i++ {
		WriteBEUint24(w, uint32(i))
	}
}
