// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package remux

import (
	"testing"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/assert"
)

func TestGopCache_Feed(t *testing.T) {

	i1 := base.RtmpMsg{
		Header:  base.RtmpHeader{Csid: 0, MsgLen: 0, MsgTypeId: 9, MsgStreamId: 10, TimestampAbs: 0},
		Payload: []byte{23, 1},
	}
	p1 := base.RtmpMsg{
		Header:  base.RtmpHeader{Csid: 0, MsgLen: 0, MsgTypeId: 9, MsgStreamId: 10, TimestampAbs: 0},
		Payload: []byte{6, 0, 1},
	}
	i2 := base.RtmpMsg{
		Header:  base.RtmpHeader{Csid: 0, MsgLen: 0, MsgTypeId: 9, MsgStreamId: 10, TimestampAbs: 0},
		Payload: []byte{23, 1},
	}
	p2 := base.RtmpMsg{
		Header:  base.RtmpHeader{Csid: 0, MsgLen: 0, MsgTypeId: 9, MsgStreamId: 10, TimestampAbs: 0},
		Payload: []byte{6, 0, 2},
	}
	i3 := base.RtmpMsg{
		Header:  base.RtmpHeader{Csid: 0, MsgLen: 0, MsgTypeId: 9, MsgStreamId: 10, TimestampAbs: 0},
		Payload: []byte{23, 1},
	}
	p3 := base.RtmpMsg{
		Header:  base.RtmpHeader{Csid: 0, MsgLen: 0, MsgTypeId: 9, MsgStreamId: 10, TimestampAbs: 0},
		Payload: []byte{6, 0, 3},
	}
	i4 := base.RtmpMsg{
		Header:  base.RtmpHeader{Csid: 0, MsgLen: 0, MsgTypeId: 9, MsgStreamId: 10, TimestampAbs: 0},
		Payload: []byte{23, 1},
	}
	p4 := base.RtmpMsg{
		Header:  base.RtmpHeader{Csid: 0, MsgLen: 0, MsgTypeId: 9, MsgStreamId: 10, TimestampAbs: 0},
		Payload: []byte{6, 0, 4},
	}
	i1f := func() []byte { return []byte{1, 1} }
	p1f := func() []byte { return []byte{0, 1} }
	i2f := func() []byte { return []byte{1, 2} }
	p2f := func() []byte { return []byte{0, 2} }
	i3f := func() []byte { return []byte{1, 3} }
	p3f := func() []byte { return []byte{0, 3} }
	i4f := func() []byte { return []byte{1, 4} }
	p4f := func() []byte { return []byte{0, 4} }

	nc := NewGopCache("rtmp", "test", 3)
	assert.Equal(t, 0, nc.GetGopCount())
	assert.Equal(t, nil, nc.GetGopDataAt(0))
	assert.Equal(t, nil, nc.GetGopDataAt(1))
	assert.Equal(t, nil, nc.GetGopDataAt(2))
	assert.Equal(t, nil, nc.GetGopDataAt(3))

	nc.Feed(i1, i1f())
	assert.Equal(t, 1, nc.GetGopCount())
	assert.Equal(t, [][]byte{{1, 1}}, nc.GetGopDataAt(0))
	assert.Equal(t, nil, nc.GetGopDataAt(1))
	assert.Equal(t, nil, nc.GetGopDataAt(2))
	assert.Equal(t, nil, nc.GetGopDataAt(3))
	nc.Feed(p1, p1f())
	assert.Equal(t, 1, nc.GetGopCount())
	assert.Equal(t, [][]byte{{1, 1}, {0, 1}}, nc.GetGopDataAt(0))
	assert.Equal(t, nil, nc.GetGopDataAt(1))
	assert.Equal(t, nil, nc.GetGopDataAt(2))
	assert.Equal(t, nil, nc.GetGopDataAt(3))

	nc.Feed(i2, i2f())
	assert.Equal(t, 2, nc.GetGopCount())
	assert.Equal(t, [][]byte{{1, 1}, {0, 1}}, nc.GetGopDataAt(0))
	assert.Equal(t, [][]byte{{1, 2}}, nc.GetGopDataAt(1))
	assert.Equal(t, nil, nc.GetGopDataAt(2))
	assert.Equal(t, nil, nc.GetGopDataAt(3))
	nc.Feed(p2, p2f())
	assert.Equal(t, 2, nc.GetGopCount())
	assert.Equal(t, [][]byte{{1, 1}, {0, 1}}, nc.GetGopDataAt(0))
	assert.Equal(t, [][]byte{{1, 2}, {0, 2}}, nc.GetGopDataAt(1))
	assert.Equal(t, nil, nc.GetGopDataAt(2))
	assert.Equal(t, nil, nc.GetGopDataAt(3))

	nc.Feed(i3, i3f())
	assert.Equal(t, 3, nc.GetGopCount())
	assert.Equal(t, [][]byte{{1, 1}, {0, 1}}, nc.GetGopDataAt(0))
	assert.Equal(t, [][]byte{{1, 2}, {0, 2}}, nc.GetGopDataAt(1))
	assert.Equal(t, [][]byte{{1, 3}}, nc.GetGopDataAt(2))
	assert.Equal(t, nil, nc.GetGopDataAt(3))
	nc.Feed(p3, p3f())
	assert.Equal(t, 3, nc.GetGopCount())
	assert.Equal(t, [][]byte{{1, 1}, {0, 1}}, nc.GetGopDataAt(0))
	assert.Equal(t, [][]byte{{1, 2}, {0, 2}}, nc.GetGopDataAt(1))
	assert.Equal(t, [][]byte{{1, 3}, {0, 3}}, nc.GetGopDataAt(2))
	assert.Equal(t, nil, nc.GetGopDataAt(3))

	nc.Feed(i4, i4f())
	assert.Equal(t, 3, nc.GetGopCount())
	assert.Equal(t, [][]byte{{1, 2}, {0, 2}}, nc.GetGopDataAt(0))
	assert.Equal(t, [][]byte{{1, 3}, {0, 3}}, nc.GetGopDataAt(1))
	assert.Equal(t, [][]byte{{1, 4}}, nc.GetGopDataAt(2))
	assert.Equal(t, nil, nc.GetGopDataAt(3))
	nc.Feed(p4, p4f())
	assert.Equal(t, 3, nc.GetGopCount())
	assert.Equal(t, [][]byte{{1, 2}, {0, 2}}, nc.GetGopDataAt(0))
	assert.Equal(t, [][]byte{{1, 3}, {0, 3}}, nc.GetGopDataAt(1))
	assert.Equal(t, [][]byte{{1, 4}, {0, 4}}, nc.GetGopDataAt(2))
	assert.Equal(t, nil, nc.GetGopDataAt(3))
}
