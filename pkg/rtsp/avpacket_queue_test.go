// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/assert"
)

var (
	diffA = uint32(23)
	diffV = uint32(40)
)

var golden = []base.AvPacket{
	v(0), // 注意一个小细节，音频视频相等时，视频先输出
	a(0),
	a(23),
	v(40),
	a(46),
	a(69),
	v(80),
	a(92),
	a(115),
	v(120),
}

func TestAvPacketQueue(t *testing.T) {
	var in []base.AvPacket

	// case. 只有音频，且数量小于队列容量
	oneCase(t, []base.AvPacket{
		a(1),
	}, nil)

	// case. 只有音频，且数量大于队列容量
	in = nil
	for i := uint32(0); i <= maxQueueSize; i++ {
		in = append(in, a(i*diffA))
	}
	oneCase(t, in, in[:len(in)-1])

	// case. 只有视频，且数量大于队列容量，只是为了测试覆盖率
	in = nil
	for i := uint32(0); i <= maxQueueSize; i++ {
		in = append(in, v(i*diffV))
	}
	oneCase(t, in, in[:len(in)-1])

	// case. 最正常的数据
	oneCase(t, golden, golden[:len(golden)-1])

	// case. 音频和视频之间不完全有序
	oneCase(t, []base.AvPacket{
		a(0),
		a(23),
		a(46),
		a(69),
		v(0),
		v(40),
		v(80),
		v(120),
		a(92),
		a(115),
	}, golden[:len(golden)-1])

	// case. 起始非0，且起始不对齐
	in = nil
	for _, pkt := range golden {
		pkt2 := pkt
		if pkt2.PayloadType == base.AvPacketPtAac {
			pkt2.Timestamp += 100
		} else {
			pkt2.Timestamp += 10000
		}
		in = append(in, pkt2)
	}
	oneCase(t, in, golden[:len(golden)-1])

	// case. 起始非0，且起始不对齐，且乱序
	oneCase(t, []base.AvPacket{
		a(0 + 99),
		a(23 + 99),
		a(46 + 99),
		a(69 + 99),
		v(0 + 1234),
		v(40 + 1234),
		v(80 + 1234),
		v(120 + 1234),
		a(92 + 99),
		a(115 + 99),
	}, golden[:len(golden)-1])

	// case.
	oneCase(t, []base.AvPacket{
		a(4294967293),
		v(4294967294),
		a(4294967295),
	}, []base.AvPacket{
		v(0),
	})

	// case. 翻转1
	// i:[ A(4294967226)  V(66666)  A(4294967249)  V(66706)  A(4294967272)  A(4294967295)  V(66746)  A(22)  A(45)  V(66786)  A(68)  V(66826) ]
	// o:[ V(0)  A(0)  A(23)  V(40)  A(46)  A(69)  V(80)  V(0)  A(0)  A(23)  V(40) ]
	// q:[ A(46) ]
	ab := uint32(4294967295 - diffA*3) // max 4294967295
	vb := uint32(66666)
	in = []base.AvPacket{
		a(ab),           // 0: 0
		v(vb),           // 0
		a(ab + diffA),   // 23
		v(vb + diffV),   // 40
		a(ab + diffA*2), // 46
		a(ab + diffA*3), // 69
		v(vb + diffV*2), // 80
		a(ab + diffA*4), // 92 rotate
		a(ab + diffA*5), // 115 -> 23
		v(vb + diffV*3), // 120 -> 0
		a(ab + diffA*6), // 138
		v(vb + diffV*4), // 160
	}
	expects := [][]base.AvPacket{
		nil,
		{v(0)},
		{v(0)},
		{v(0), a(0), a(diffA)},
		{v(0), a(0), a(diffA), v(diffV)},
		{v(0), a(0), a(diffA), v(diffV)},
		{v(0), a(0), a(diffA), v(diffV), a(diffA * 2), a(diffA * 3)},
		{v(0), a(0), a(diffA), v(diffV), a(diffA * 2), a(diffA * 3), v(diffV * 2)},
		{v(0), a(0), a(diffA), v(diffV), a(diffA * 2), a(diffA * 3), v(diffV * 2)},
		{v(0), a(0), a(diffA), v(diffV), a(diffA * 2), a(diffA * 3), v(diffV * 2), v(0)},
		{v(0), a(0), a(diffA), v(diffV), a(diffA * 2), a(diffA * 3), v(diffV * 2), v(0)},
		{v(0), a(0), a(diffA), v(diffV), a(diffA * 2), a(diffA * 3), v(diffV * 2), v(0), a(0), a(diffA), v(diffV)},
	}
	for i := 0; i < len(in); i++ {
		out, q := oneCase(t, in[:i+1], expects[i])
		Log.Infof("-----%d", i)
		Log.Infof("i:%s", packetsReadable(in[:i+1]))
		Log.Infof("o:%s", packetsReadable(out))
		Log.Infof("e:%s", packetsReadable(expects[i]))
		Log.Infof("q:%s", packetsReadable(peekQueuePackets(q)))
	}

	// case. 翻转2
	// i:[ V(4294967215)  A(12345)  A(12368)  V(4294967255)  A(12391)  A(12414)  V(4294967295)  A(12437)  A(12460)  V(39)  A(12483)  V(79)  A(12506)  A(12529) ]
	// o:[ V(0)  A(0)  A(23)  V(40)  A(46)  A(69)  V(80)  A(92)  A(115)  V(0)  A(0)  A(23)  V(40) ]
	// q:[ A(46) ]
	ab = uint32(12345)
	vb = uint32(4294967295 - diffV*2) // max 4294967295
	in = []base.AvPacket{
		v(vb),           // 0
		a(ab),           // 0
		a(ab + diffA),   // 23
		v(vb + diffV),   // 40
		a(ab + diffA*2), // 46
		a(ab + diffA*3), // 69
		v(vb + diffV*2), // 80
		a(ab + diffA*4), // 92
		a(ab + diffA*5), // 115
		v(vb + diffV*3), // 120 rotate
		a(ab + diffA*6), // 138 -> 0
		v(vb + diffV*4), // 160 -> 40
		a(ab + diffA*7), // 161
		a(ab + diffA*8), // 184
	}
	expects = [][]base.AvPacket{
		nil,
		{v(0)},
		{v(0)},
		{v(0), a(0), a(diffA)},
		{v(0), a(0), a(diffA), v(diffV)},
		{v(0), a(0), a(diffA), v(diffV)},
		{v(0), a(0), a(diffA), v(diffV), a(diffA * 2), a(diffA * 3)},
		{v(0), a(0), a(diffA), v(diffV), a(diffA * 2), a(diffA * 3), v(diffV * 2)},
		{v(0), a(0), a(diffA), v(diffV), a(diffA * 2), a(diffA * 3), v(diffV * 2)},
		{v(0), a(0), a(diffA), v(diffV), a(diffA * 2), a(diffA * 3), v(diffV * 2), a(diffA * 4), a(diffA * 5)},
		{v(0), a(0), a(diffA), v(diffV), a(diffA * 2), a(diffA * 3), v(diffV * 2), a(diffA * 4), a(diffA * 5), v(0)},
		{v(0), a(0), a(diffA), v(diffV), a(diffA * 2), a(diffA * 3), v(diffV * 2), a(diffA * 4), a(diffA * 5), v(0), a(0)},
		{v(0), a(0), a(diffA), v(diffV), a(diffA * 2), a(diffA * 3), v(diffV * 2), a(diffA * 4), a(diffA * 5), v(0), a(0), a(diffA)},
		{v(0), a(0), a(diffA), v(diffV), a(diffA * 2), a(diffA * 3), v(diffV * 2), a(diffA * 4), a(diffA * 5), v(0), a(0), a(diffA), v(diffV)},
	}
	for i := 0; i < len(in); i++ {
		oneCase(t, in[:i+1], expects[i])
	}
}

func a(t uint32) base.AvPacket {
	return base.AvPacket{
		PayloadType: base.AvPacketPtAac,
		Timestamp:   int64(t),
	}
}

func v(t uint32) base.AvPacket {
	return base.AvPacket{
		PayloadType: base.AvPacketPtAvc,
		Timestamp:   int64(t),
	}
}

func oneCase(t *testing.T, in []base.AvPacket, expected []base.AvPacket) (out []base.AvPacket, q *AvPacketQueue) {
	out, q = calc(in)
	assert.Equal(t, expected, out)
	return out, q
}

func calc(in []base.AvPacket) (out []base.AvPacket, q *AvPacketQueue) {
	q = NewAvPacketQueue(func(pkt base.AvPacket) {
		out = append(out, pkt)
	})
	for _, pkt := range in {
		q.Feed(pkt)
	}
	return out, q
}

func packetsReadable(pkts []base.AvPacket) string {
	var buf bytes.Buffer
	buf.WriteString("[")
	for _, pkt := range pkts {
		if pkt.PayloadType == base.AvPacketPtAac {
			buf.WriteString(fmt.Sprintf(" A(%d) ", pkt.Timestamp))
		} else {
			buf.WriteString(fmt.Sprintf(" V(%d) ", pkt.Timestamp))
		}
	}
	buf.WriteString("]")
	return buf.String()
}

func peekQueuePackets(q *AvPacketQueue) []base.AvPacket {
	var out []base.AvPacket
	for i := 0; i < q.audioQueue.Size(); i++ {
		pkt, _ := q.audioQueue.At(i)
		ppkt := pkt.(base.AvPacket)
		out = append(out, ppkt)
	}
	for i := 0; i < q.videoQueue.Size(); i++ {
		pkt, _ := q.videoQueue.At(i)
		ppkt := pkt.(base.AvPacket)
		out = append(out, ppkt)
	}
	return out
}
