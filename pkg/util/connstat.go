package util

import (
	"sync/atomic"
	"time"
)

// 高性能场景下实现长连接流数据读写超时功能
// 不使用Go的库函数SetDeadline
// 也不在每次读写数据时使用time now获取读写时间
// 允许出现两秒左右的误差

type ConnStat struct {
	readTimeout  int64
	writeTimeout int64

	totalReadByte  uint64
	totalWriteByte uint64

	prevTotalReadByte  uint64
	prevTotalWriteByte uint64

	lastReadActiveTick  int64
	lastWriteActiveTick int64
}

// 单位秒，设置为0则检查时用于不超时
func (cs *ConnStat) Start(readTimeout int64, writeTimeout int64) {
	cs.readTimeout = readTimeout
	cs.writeTimeout = writeTimeout

	now := time.Now().Unix()
	cs.lastReadActiveTick = now
	cs.lastWriteActiveTick = now
}

func (cs *ConnStat) Read(n int) {
	atomic.AddUint64(&cs.totalReadByte, uint64(n))
}

func (cs *ConnStat) Write(n int) {
	atomic.AddUint64(&cs.totalWriteByte, uint64(n))
}

// 检查时传入当前时间戳。检查频率应该小于超时阈值。频率越低，则越精确
func (cs *ConnStat) Check(now int64) (isReadTimeout bool, isWriteTimeout bool) {
	if cs.readTimeout == 0 { // 没有设置，则不用检查
		isReadTimeout = false
	} else {
		trb := atomic.LoadUint64(&cs.totalReadByte)
		if trb == 0 { // 历史从来没有收到过数据
			isReadTimeout = (now - cs.lastReadActiveTick) > cs.readTimeout
		} else {
			if trb-cs.prevTotalReadByte > 0 { // 距离上次检查有收到过数据
				isReadTimeout = false
				cs.lastReadActiveTick = now
			} else {
				isReadTimeout = (now - cs.lastReadActiveTick) > cs.readTimeout
			}
		}
		cs.prevTotalReadByte = trb
	}

	if cs.writeTimeout == 0 {
		isWriteTimeout = false
	} else {
		twb := atomic.LoadUint64(&cs.totalWriteByte)
		if twb == 0 {
			isWriteTimeout = (now - cs.lastWriteActiveTick) > cs.writeTimeout
		} else {
			if twb-cs.prevTotalWriteByte > 0 {
				isWriteTimeout = false
				cs.lastWriteActiveTick = now
			} else {
				isWriteTimeout = (now - cs.lastWriteActiveTick) > cs.writeTimeout
			}
		}
		cs.prevTotalWriteByte = twb
	}

	return
}
