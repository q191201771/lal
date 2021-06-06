// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import "errors"

var (
	ErrSessionNotStarted = errors.New("lal.base: session has not been started yet")
)

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

// 调用约束：对于Client类型的Session，调用Start函数并返回成功后才能调用，否则行为未定义
type IClientSessionLifecycle interface {
	// 关闭session
	// 业务方想主动关闭session时调用
	// 注意，Start成功后的session，必须显示调用Dispose释放资源（即使是被动接收到了WaitChan信号）
	Dispose() error

	// Start成功后，可使用这个channel来接收session结束的信号
	WaitChan() <-chan error
}

type IServerSessionLifecycle interface {
	// 开启session的事件循环，阻塞直到session结束
	RunLoop() error

	// 主动关闭session时调用
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
