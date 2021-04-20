// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"testing"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/assert"
)

func TestGOPCache_Feed(t *testing.T) {

	i1 := base.RTMPMsg{
		Header:  base.RTMPHeader{CSID: 0, MsgLen: 0, MsgTypeID: 9, MsgStreamID: 10, TimestampAbs: 0},
		Payload: []byte{23, 1},
	}
	p1 := base.RTMPMsg{
		Header:  base.RTMPHeader{CSID: 0, MsgLen: 0, MsgTypeID: 9, MsgStreamID: 10, TimestampAbs: 0},
		Payload: []byte{6, 0, 1},
	}
	i2 := base.RTMPMsg{
		Header:  base.RTMPHeader{CSID: 0, MsgLen: 0, MsgTypeID: 9, MsgStreamID: 10, TimestampAbs: 0},
		Payload: []byte{23, 1},
	}
	p2 := base.RTMPMsg{
		Header:  base.RTMPHeader{CSID: 0, MsgLen: 0, MsgTypeID: 9, MsgStreamID: 10, TimestampAbs: 0},
		Payload: []byte{6, 0, 2},
	}
	i3 := base.RTMPMsg{
		Header:  base.RTMPHeader{CSID: 0, MsgLen: 0, MsgTypeID: 9, MsgStreamID: 10, TimestampAbs: 0},
		Payload: []byte{23, 1},
	}
	p3 := base.RTMPMsg{
		Header:  base.RTMPHeader{CSID: 0, MsgLen: 0, MsgTypeID: 9, MsgStreamID: 10, TimestampAbs: 0},
		Payload: []byte{6, 0, 3},
	}
	i4 := base.RTMPMsg{
		Header:  base.RTMPHeader{CSID: 0, MsgLen: 0, MsgTypeID: 9, MsgStreamID: 10, TimestampAbs: 0},
		Payload: []byte{23, 1},
	}
	p4 := base.RTMPMsg{
		Header:  base.RTMPHeader{CSID: 0, MsgLen: 0, MsgTypeID: 9, MsgStreamID: 10, TimestampAbs: 0},
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

	nc := NewGOPCache("rtmp", "test", 3)
	assert.Equal(t, 0, nc.GetGOPCount())
	assert.Equal(t, nil, nc.GetGOPDataAt(0))
	assert.Equal(t, nil, nc.GetGOPDataAt(1))
	assert.Equal(t, nil, nc.GetGOPDataAt(2))
	assert.Equal(t, nil, nc.GetGOPDataAt(3))

	nc.Feed(i1, i1f)
	assert.Equal(t, 1, nc.GetGOPCount())
	assert.Equal(t, [][]byte{{1, 1}}, nc.GetGOPDataAt(0))
	assert.Equal(t, nil, nc.GetGOPDataAt(1))
	assert.Equal(t, nil, nc.GetGOPDataAt(2))
	assert.Equal(t, nil, nc.GetGOPDataAt(3))
	nc.Feed(p1, p1f)
	assert.Equal(t, 1, nc.GetGOPCount())
	assert.Equal(t, [][]byte{{1, 1}, {0, 1}}, nc.GetGOPDataAt(0))
	assert.Equal(t, nil, nc.GetGOPDataAt(1))
	assert.Equal(t, nil, nc.GetGOPDataAt(2))
	assert.Equal(t, nil, nc.GetGOPDataAt(3))

	nc.Feed(i2, i2f)
	assert.Equal(t, 2, nc.GetGOPCount())
	assert.Equal(t, [][]byte{{1, 1}, {0, 1}}, nc.GetGOPDataAt(0))
	assert.Equal(t, [][]byte{{1, 2}}, nc.GetGOPDataAt(1))
	assert.Equal(t, nil, nc.GetGOPDataAt(2))
	assert.Equal(t, nil, nc.GetGOPDataAt(3))
	nc.Feed(p2, p2f)
	assert.Equal(t, 2, nc.GetGOPCount())
	assert.Equal(t, [][]byte{{1, 1}, {0, 1}}, nc.GetGOPDataAt(0))
	assert.Equal(t, [][]byte{{1, 2}, {0, 2}}, nc.GetGOPDataAt(1))
	assert.Equal(t, nil, nc.GetGOPDataAt(2))
	assert.Equal(t, nil, nc.GetGOPDataAt(3))

	nc.Feed(i3, i3f)
	assert.Equal(t, 3, nc.GetGOPCount())
	assert.Equal(t, [][]byte{{1, 1}, {0, 1}}, nc.GetGOPDataAt(0))
	assert.Equal(t, [][]byte{{1, 2}, {0, 2}}, nc.GetGOPDataAt(1))
	assert.Equal(t, [][]byte{{1, 3}}, nc.GetGOPDataAt(2))
	assert.Equal(t, nil, nc.GetGOPDataAt(3))
	nc.Feed(p3, p3f)
	assert.Equal(t, 3, nc.GetGOPCount())
	assert.Equal(t, [][]byte{{1, 1}, {0, 1}}, nc.GetGOPDataAt(0))
	assert.Equal(t, [][]byte{{1, 2}, {0, 2}}, nc.GetGOPDataAt(1))
	assert.Equal(t, [][]byte{{1, 3}, {0, 3}}, nc.GetGOPDataAt(2))
	assert.Equal(t, nil, nc.GetGOPDataAt(3))

	nc.Feed(i4, i4f)
	assert.Equal(t, 3, nc.GetGOPCount())
	assert.Equal(t, [][]byte{{1, 2}, {0, 2}}, nc.GetGOPDataAt(0))
	assert.Equal(t, [][]byte{{1, 3}, {0, 3}}, nc.GetGOPDataAt(1))
	assert.Equal(t, [][]byte{{1, 4}}, nc.GetGOPDataAt(2))
	assert.Equal(t, nil, nc.GetGOPDataAt(3))
	nc.Feed(p4, p4f)
	assert.Equal(t, 3, nc.GetGOPCount())
	assert.Equal(t, [][]byte{{1, 2}, {0, 2}}, nc.GetGOPDataAt(0))
	assert.Equal(t, [][]byte{{1, 3}, {0, 3}}, nc.GetGOPDataAt(1))
	assert.Equal(t, [][]byte{{1, 4}, {0, 4}}, nc.GetGOPDataAt(2))
	assert.Equal(t, nil, nc.GetGOPDataAt(3))
}
