package assert

import "testing"
//import aaa "github.com/stretchr/testify/assert"

func TestEqual(t *testing.T) {
	Equal(t, nil, nil, "fxxk.")
	Equal(t, 1, 1, "fxxk.")
	Equal(t, "aaa", "aaa", "fxxk.")
	var ch chan struct{}
	Equal(t, nil, ch, "fxxk.")
	var m map[string]string
	Equal(t, nil, m, "fxxk.")
	var p *int
	Equal(t, nil, p, "fxxk.")
	var i interface{}
	Equal(t, nil, i, "fxxk.")
	var b []byte
	Equal(t, nil, b, "fxxk.")

	Equal(t, true, isNil(nil), "fxxk.")
	Equal(t, false, isNil("aaa"), "fxxk.")
	Equal(t, false, equal([]byte{}, "aaa"), "fxxk.")
	Equal(t, true, equal([]byte{}, []byte{}), "fxxk.")
	Equal(t, true, equal([]byte{0, 1, 2}, []byte{0, 1, 2}), "fxxk.")
}
