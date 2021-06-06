// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package datamanager

// 本demo的数据存储在内存中（只实现了DataManagerMemory），所以存在单点风险（指的是dispatch永久性发生故障，短暂故障或重启是ok的），
// 生产环境可以把数据存储在redis、mysql等数据库中（实现DataManager interface即可）。

type DataManger interface {
	AddPub(streamName, serverId string)
	DelPub(streamName, serverId string)
	QueryPub(streamName string) (serverId string, exist bool)

	// 1. 全量校正。比如自身服务重启了，lal节点重启了，或其他原因Add、Del消息丢失了
	// 2. 心跳保活
	UpdatePub(serverId string, streamNameList []string)
}

type DataManagerType int

const (
	DmtMemory DataManagerType = iota
)

// @param serverTimeoutSec 超过该时间间隔没有Update，则清空对应节点的所有信息
func NewDataManager(t DataManagerType, serverTimeoutSec int) DataManger {
	switch t {
	case DmtMemory:
		return NewDataManagerMemory(serverTimeoutSec)
	default:
		panic("invalid data manager type")
	}
}
