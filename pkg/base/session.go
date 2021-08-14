// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"errors"
	"io"
	"strings"
)

var (
	ErrSessionNotStarted = errors.New("lal.base: session has not been started yet")
)

// IsUseClosedConnectionError 当connection处于这些情况时，就不需要再Close了
// TODO(chef): 临时放这
// TODO(chef): 目前暂时没有使用，因为connection支持多次调用Close
//
func IsUseClosedConnectionError(err error) bool {
	if err == io.EOF || (err != nil && strings.Contains(err.Error(), "use of closed network connection")) {
		return true
	}
	return false
}

type IClientSession interface {
	// PushSession:
	// Push()
	// Write()
	// Flush()
	// PullSession:
	// Pull()

	IClientSessionLifecycle
	ISessionUrlContext
	IObject
	ISessionStat
}

type IServerSession interface {
	IServerSessionLifecycle
	ISessionUrlContext
	IObject
	ISessionStat
}

type IClientSessionLifecycle interface {
	// Dispose 主动关闭session时调用
	//
	// 注意，只有Start（具体session的Start类型函数一般命令为Push和Pull）成功后的session才能调用，否则行为未定义
	//
	// Dispose可在任意协程内调用
	//
	// 注意，目前Dispose允许调用多次，但是未来可能不对该表现做保证
	//
	// Dispose后，调用Write函数将返回错误
	//
	// @return 可以通过返回值判断调用Dispose前，session是否已经被关闭了 TODO(chef) 这个返回值没有太大意义，后面可能会删掉
	//
	Dispose() error

	// WaitChan Start成功后，可使用这个channel来接收session结束的消息
	//
	// 注意，只有Start成功后的session才能调用，否则行为未定义
	//
	// 注意，目前WaitChan只会被通知一次，但是未来可能不对该表现做保证，业务方应该只关注第一次通知
	//
	// TODO(chef): 是否应该严格保证：获取到关闭消息后，后续不应该再有该session的回调上来
	//
	// @return 一般关闭有以下几种情况：
	//         - 对端关闭，此时error为EOF
	//         - 本端底层关闭，比如协议非法等，此时error为具体的错误值
	//         - 本端上层主动调用Dispose关闭，此时error为nil
	//
	WaitChan() <-chan error
}

type IServerSessionLifecycle interface {
	// 开启session的事件循环，阻塞直到session结束
	RunLoop() error

	// 主动关闭session时调用
	//
	// 如果是session通知业务方session已关闭（比如`RunLoop`函数返回错误），则不需要调用`Dispose` TODO(chef): review现状
	//
	Dispose() error
}

// 调用约束：对于Client类型的Session，调用Start函数并返回成功后才能调用，否则行为未定义
type ISessionStat interface {
	// 周期性调用该函数，用于计算bitrate
	//
	// @param intervalSec 距离上次调用的时间间隔，单位毫秒
	UpdateStat(intervalSec uint32)

	// 获取session状态
	//
	// @return 注意，结构体中的`Bitrate`的值由最近一次`func UpdateStat`调用计算决定，其他值为当前最新值
	GetStat() StatSession

	// 周期性调用该函数，判断是否有读取、写入数据
	// 注意，判断的依据是，距离上次调用该函数的时间间隔内，是否有读取、写入数据
	// 注意，不活跃，并一定是链路或网络有问题，也可能是业务层没有写入数据
	//
	// @return readAlive  读取是否获取
	// @return writeAlive 写入是否活跃
	IsAlive() (readAlive, writeAlive bool)
}

// 获取和流地址相关的信息
//
// 调用约束：对于Client类型的Session，调用Start函数并返回成功后才能调用，否则行为未定义
type ISessionUrlContext interface {
	Url() string
	AppName() string
	StreamName() string
	RawQuery() string
}

type IObject interface {
	// 对象的全局唯一标识
	UniqueKey() string
}

// TODO chef: rtmp.ClientSession修改为BaseClientSession更好些
