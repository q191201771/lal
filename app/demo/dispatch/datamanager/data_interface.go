// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package datamanager

type DataManger interface {
	AddPub(streamName, serverID string)
	DelPub(streamName, serverID string)
	QueryPub(streamName string) (serverID string, exist bool)
}

type DataManagerType int

const (
	DMTMemory DataManagerType = iota
)

func NewDataManager(t DataManagerType) DataManger {
	switch t {
	case DMTMemory:
		return NewDataManagerMemory()
	default:
		panic("invalid data manager type")
	}
}
