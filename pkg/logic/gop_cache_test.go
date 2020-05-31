package logic

import (
	`github.com/q191201771/naza/pkg/assert`
	`testing`
)

func TestGopCircularQueueCap0(t *testing.T) {
	var gcqCap int = 0
	
	gcq := NewGopCircularQueue(gcqCap)
	
	//fl
	assert.Equal(t, true, gcq.Empty())
	assert.Equal(t, true, gcq.Full())
	assert.Equal(t, 0, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, nil, gcq.At(0))
	assert.Equal(t, nil, gcq.At(0))
	assert.Equal(t, nil, gcq.Front())
	assert.Equal(t, nil, gcq.Back())
	assert.Equal(t, nil, gcq.Dequeue())
	
	
	//fl
	tGOP := &GOP{
		data:nil,
	}
	gcq.Enqueue(tGOP)
	//fg
	assert.Equal(t, true, gcq.Empty())
	assert.Equal(t, true, gcq.Full())
	assert.Equal(t, 0, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, nil, gcq.At(0))
	assert.Equal(t, nil, gcq.At(0))
	assert.Equal(t, nil, gcq.Front())
	assert.Equal(t, nil, gcq.Back())
	
	
	//弹出一个后
	assert.Equal(t, nil, gcq.Dequeue())
	//fg
	assert.Equal(t, true, gcq.Empty())
	assert.Equal(t, true, gcq.Full())
	assert.Equal(t, 0, gcq.Len())
	assert.Equal(t, 0, gcq.Cap())
	assert.Equal(t, nil, gcq.At(0))
	assert.Equal(t, nil,  gcq.At(2))
	assert.Equal(t, nil, gcq.Front())
	assert.Equal(t, nil, gcq.Back())
	assert.Equal(t, nil, gcq.Dequeue())
	
	//放入2个元素后
	gcq.Enqueue(tGOP)
	gcq.Enqueue(tGOP)
	assert.Equal(t, true, gcq.Empty())
	assert.Equal(t, true, gcq.Full())
	assert.Equal(t, 0, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, nil, gcq.At(0))
	assert.Equal(t, nil, gcq.At(0))
	assert.Equal(t, nil, gcq.Front())
	assert.Equal(t, nil, gcq.Back())
	assert.Equal(t, nil, gcq.Dequeue())
}

func TestGopCircularQueue(t *testing.T) {
	var gcqCap int = 3
	
	//元素为个数为0
	//fl,_, _, _
	gcq := NewGopCircularQueue(gcqCap)
	assert.Equal(t, true, gcq.Empty())
	assert.Equal(t, false, gcq.Full())
	assert.Equal(t, 0, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	//gop :=
	assert.Equal(t, nil, gcq.At(0))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, nil, gcq.Front())
	assert.Equal(t, nil, gcq.Back())
	assert.Equal(t, nil, gcq.Dequeue())
	
	//fl,_, _, _
	tGOP := &GOP{
		data:[][]byte{[]byte("t")},
	}
	gcq.Enqueue(tGOP)
	//放入一个元素后
	//tf,l, _, _
	assert.Equal(t, false, gcq.Empty())
	assert.Equal(t, false, gcq.Full())
	assert.Equal(t, 1, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, tGOP,  gcq.At(0))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, tGOP, gcq.Front())
	assert.Equal(t, tGOP, gcq.Back())
	
	//在弹出一个元素后
	//tf,l, _,_
	assert.Equal(t, tGOP, gcq.Dequeue())
	//_,fl, _,_
	assert.Equal(t, true, gcq.Empty())
	assert.Equal(t, false, gcq.Full())
	assert.Equal(t, 0, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, nil, gcq.At(0))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, nil, gcq.Front())
	assert.Equal(t, nil, gcq.Back())
	assert.Equal(t, nil, gcq.Dequeue())
	
	//_,fl, _,_
	GOP0 := &GOP{
		data:[][]byte{[]byte("0")},
	}
	gcq.Enqueue(GOP0)
	//_,f0,l_,_
	GOP1 := &GOP{
		data:[][]byte{[]byte("1")},
	}
	gcq.Enqueue(GOP1)
	//_,f0, 1, l
	//验证遍历情况
	assert.Equal(t, false, gcq.Empty())
	assert.Equal(t, false, gcq.Full())
	assert.Equal(t, 2, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, GOP0, gcq.At(0))
	assert.Equal(t, GOP1, gcq.At(1))
	assert.Equal(t, nil, gcq.At(2))
	assert.Equal(t, GOP0, gcq.Front())
	assert.Equal(t, GOP1, gcq.Back())
	
	GOP2 := &GOP{
		data:[][]byte{[]byte("2")},
	}
	gcq.Enqueue(GOP2)
	//l,f0, 1, 2
	assert.Equal(t, false, gcq.Empty())
	assert.Equal(t, true, gcq.Full())
	assert.Equal(t, 3, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, GOP0, gcq.At(0))
	assert.Equal(t, GOP1, gcq.At(1))
	assert.Equal(t, GOP2, gcq.At(2))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, GOP0, gcq.Front())
	assert.Equal(t, GOP2, gcq.Back())
	
	//
	GOP3 := &GOP{
		data:[][]byte{[]byte("3")},
	}
	gcq.Enqueue(GOP3)
	//3, l, f1, 2
	assert.Equal(t, false, gcq.Empty())
	assert.Equal(t, true, gcq.Full())
	assert.Equal(t, 3, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, GOP1, gcq.At(0))
	assert.Equal(t, GOP2, gcq.At(1))
	assert.Equal(t, GOP3, gcq.At(2))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, GOP1, gcq.Front())
	assert.Equal(t, GOP3, gcq.Back())
	
	
	
	//3, l, f1, 2
	assert.Equal(t, GOP1, gcq.Dequeue())
	//出队后
	//3, l, _, f2
	assert.Equal(t, false, gcq.Empty())
	assert.Equal(t, false, gcq.Full())
	assert.Equal(t, 2, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, GOP2, gcq.At(0))
	assert.Equal(t, GOP3, gcq.At(1))
	assert.Equal(t, nil, gcq.At(2))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, nil, gcq.At(4))
	assert.Equal(t, GOP2, gcq.Front())
	assert.Equal(t, GOP3, gcq.Back())
	
	
	
	//3, l, _, f2
	GOP4 := &GOP{
		data:[][]byte{[]byte("4")},
	}
	gcq.Enqueue(GOP4)
	//入队后
	//3, 4, l, f2
	assert.Equal(t, false, gcq.Empty())
	assert.Equal(t, true, gcq.Full())
	assert.Equal(t, 3, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, GOP2, gcq.At(0))
	assert.Equal(t, GOP3, gcq.At(1))
	assert.Equal(t, GOP4, gcq.At(2))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, nil, gcq.At(4))
	assert.Equal(t, GOP2, gcq.Front())
	assert.Equal(t, GOP4, gcq.Back())
	
	//3, 4, l, 2f
	assert.Equal(t, GOP2, gcq.Dequeue())
	//3f, 4, l, _
	
	assert.Equal(t, false, gcq.Empty())
	assert.Equal(t, false, gcq.Full())
	assert.Equal(t, 2, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, GOP3,  gcq.At(0))
	assert.Equal(t, GOP4, gcq.At(1))
	assert.Equal(t, nil, gcq.At(2))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, nil, gcq.At(4))
	assert.Equal(t, GOP3, gcq.Front())
	assert.Equal(t, GOP4, gcq.Back())
	
	//3f, 4, l, _
	GOP5 := &GOP{
		data:[][]byte{[]byte("5")},
	}
	gcq.Enqueue(GOP5)
	//3f, 4, 5, l
	assert.Equal(t, false, gcq.Empty())
	assert.Equal(t, true, gcq.Full())
	assert.Equal(t, 3, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, GOP3, gcq.At(0))
	assert.Equal(t, GOP4,  gcq.At(1))
	assert.Equal(t, GOP5, gcq.At(2))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, nil, gcq.At(4))
	assert.Equal(t, GOP3, gcq.Front())
	assert.Equal(t, GOP5, gcq.Back())
	
	
	
	//3f, 4, 5, l
	assert.Equal(t, GOP3, gcq.Dequeue())
	//_, 4f, 5, l
	assert.Equal(t, false, gcq.Empty())
	assert.Equal(t, false, gcq.Full())
	assert.Equal(t, 2, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, GOP4, gcq.At(0))
	assert.Equal(t, GOP5, gcq.At(1))
	assert.Equal(t, nil,  gcq.At(2))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, nil, gcq.At(4))
	assert.Equal(t, GOP4, gcq.Front())
	assert.Equal(t, GOP5, gcq.Back())
	
	//_, 4f, 5, l
	assert.Equal(t, GOP4, gcq.Dequeue())
	//_, _, 5f, l
	assert.Equal(t, false, gcq.Empty())
	assert.Equal(t, false, gcq.Full())
	assert.Equal(t, 1, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, GOP5, gcq.At(0))
	assert.Equal(t, nil, gcq.At(1))
	assert.Equal(t, nil, gcq.At(2))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, nil,  gcq.At(4))
	assert.Equal(t, GOP5, gcq.Front())
	assert.Equal(t, GOP5, gcq.Back())
	
	//_, _, 5f, l
	assert.Equal(t, GOP5, gcq.Dequeue())
	//_, _, _, fl
	assert.Equal(t, true, gcq.Empty())
	assert.Equal(t, false, gcq.Full())
	assert.Equal(t, 0, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, nil, gcq.At(0))
	assert.Equal(t, nil, gcq.At(1))
	assert.Equal(t, nil, gcq.At(2))
	assert.Equal(t, nil,  gcq.At(3))
	assert.Equal(t, nil, gcq.At(4))
	assert.Equal(t, nil, gcq.Front())
	assert.Equal(t, nil, gcq.Back())
	
	
	//_, _, _, fl
	GOP6 := &GOP{
		data:[][]byte{[]byte("6")},
	}
	gcq.Enqueue(GOP6)
	//l, _, _, f6
	GOP7 := &GOP{
		data:[][]byte{[]byte("7")},
	}
	gcq.Enqueue(GOP7)
	//7, l, _, f6
	GOP8 := &GOP{
		data:[][]byte{[]byte("8")},
	}
	gcq.Enqueue(GOP8)
	//7, 8, l, f6
	assert.Equal(t, false, gcq.Empty())
	assert.Equal(t, true, gcq.Full())
	assert.Equal(t, 3, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, GOP6, gcq.At(0))
	assert.Equal(t, GOP7, gcq.At(1))
	assert.Equal(t, GOP8, gcq.At(2))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, nil, gcq.At(4))
	assert.Equal(t, GOP6, gcq.Front())
	assert.Equal(t, GOP8, gcq.Back())
	
	
	GOP9 := &GOP{
		data:[][]byte{[]byte("9")},
	}
	gcq.Enqueue(GOP9)
	//f7, 8, 9, l
	assert.Equal(t, false, gcq.Empty())
	assert.Equal(t, true, gcq.Full())
	assert.Equal(t, 3, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, GOP7, gcq.At(0))
	assert.Equal(t, GOP8, gcq.At(1))
	assert.Equal(t, GOP9, gcq.At(2))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, nil, gcq.At(4))
	assert.Equal(t, GOP7, gcq.Front())
	assert.Equal(t, GOP9,  gcq.Back())
	
	//f7, 8, 9, l
	assert.Equal(t, GOP7, gcq.Dequeue())
	//_, f8, 9, l
	assert.Equal(t, GOP8, gcq.Dequeue())
	//_, _, f9, l
	assert.Equal(t, GOP9, gcq.Dequeue())
	//_, _, _, fl
	assert.Equal(t, true, gcq.Empty())
	assert.Equal(t, false, gcq.Full())
	assert.Equal(t, 0, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, nil, gcq.At(0))
	assert.Equal(t, nil, gcq.At(1))
	assert.Equal(t, nil, gcq.At(2))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, nil, gcq.At(4))
	assert.Equal(t, nil,  gcq.Front())
	assert.Equal(t, nil, gcq.Back())
	
	
	//_, _, _, fl
	assert.Equal(t, nil, gcq.Dequeue())
	//_, _, _, fl
	assert.Equal(t, true, gcq.Empty())
	assert.Equal(t, false, gcq.Full())
	assert.Equal(t, 0, gcq.Len())
	assert.Equal(t, gcqCap, gcq.Cap())
	assert.Equal(t, nil, gcq.At(0))
	assert.Equal(t, nil,  gcq.At(1))
	assert.Equal(t, nil, gcq.At(2))
	assert.Equal(t, nil, gcq.At(3))
	assert.Equal(t, nil, gcq.At(4))
	assert.Equal(t, nil, gcq.Front())
	assert.Equal(t, nil, gcq.Back())
}
