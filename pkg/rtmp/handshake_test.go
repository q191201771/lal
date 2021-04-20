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
	"testing"

	. "github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/assert"
)

func TestHandshakeSimple(t *testing.T) {
	var err error
	var hc HandshakeClientSimple
	var hs HandshakeServer
	b := &bytes.Buffer{}
	err = hc.WriteC0C1(b)
	assert.Equal(t, nil, err)
	err = hs.ReadC0C1(b)
	assert.Equal(t, nil, err)
	err = hs.WriteS0S1S2(b)
	assert.Equal(t, nil, err)
	err = hc.ReadS0S1S2(b)
	assert.Equal(t, nil, err)
	err = hc.WriteC2(b)
	assert.Equal(t, nil, err)
	err = hs.ReadC2(b)
	assert.Equal(t, nil, err)
}

func TestHandshakeComplex(t *testing.T) {
	var err error
	var hc HandshakeClientComplex
	var hs HandshakeServer
	b := &bytes.Buffer{}
	err = hc.WriteC0C1(b)
	assert.Equal(t, nil, err)
	err = hs.ReadC0C1(b)
	assert.Equal(t, nil, err)
	err = hs.WriteS0S1S2(b)
	assert.Equal(t, nil, err)
	err = hc.ReadS0S1S2(b)
	assert.Equal(t, nil, err)
	err = hc.WriteC2(b)
	assert.Equal(t, nil, err)
	err = hs.ReadC2(b)
	assert.Equal(t, nil, err)
}

func BenchmarkHandshakeSimple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var hc HandshakeClientSimple
		var hs HandshakeServer
		b := &bytes.Buffer{}
		_ = hc.WriteC0C1(b)
		_ = hs.ReadC0C1(b)
		_ = hs.WriteS0S1S2(b)
		_ = hc.ReadS0S1S2(b)
		_ = hc.WriteC2(b)
		_ = hs.ReadC2(b)
	}
}

func BenchmarkHandshakeComplex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var hc HandshakeClientComplex
		var hs HandshakeServer
		b := &bytes.Buffer{}
		_ = hc.WriteC0C1(b)
		_ = hs.ReadC0C1(b)
		_ = hs.WriteS0S1S2(b)
		_ = hc.ReadS0S1S2(b)
		_ = hc.WriteC2(b)
		_ = hs.ReadC2(b)
	}
}
