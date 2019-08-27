package assert

import "testing"
//import aaa "github.com/stretchr/testify/assert"

func TestEqual(t *testing.T) {
	Equal(t, nil, nil)
	Equal(t, nil, nil, "fxxk.")
	Equal(t, 1, 1)
	Equal(t, "aaa", "aaa")
	var ch chan struct{}
	Equal(t, nil, ch)
	var m map[string]string
	Equal(t, nil, m)
	var p *int
	Equal(t, nil, p)
	var i interface{}
	Equal(t, nil, i)
	var b []byte
	Equal(t, nil, b)

	Equal(t, true, isNil(nil))
	Equal(t, false, isNil("aaa"))
	Equal(t, false, equal([]byte{}, "aaa"))
	Equal(t, true, equal([]byte{}, []byte{}))
	Equal(t, true, equal([]byte{0, 1, 2}, []byte{0, 1, 2}))
}
