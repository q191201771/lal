// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package datamanager

import (
	"sync"

	"github.com/q191201771/naza/pkg/nazalog"
)

type DataManagerMemory struct {
	mutex              sync.Mutex
	pubStream2ServerID map[string]string
}

func NewDataManagerMemory() *DataManagerMemory {
	return &DataManagerMemory{
		pubStream2ServerID: make(map[string]string),
	}
}

func (d *DataManagerMemory) AddPub(streamName, serverID string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	nazalog.Infof("add pub. streamName=%s, serverID=%s", streamName, serverID)
	d.pubStream2ServerID[streamName] = serverID
}

func (d *DataManagerMemory) DelPub(streamName, serverID string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	// 清除用户推流对应的节点信息
	cacheServerID, exist := d.pubStream2ServerID[streamName]
	if !exist || serverID != cacheServerID {
		nazalog.Errorf("del pub but server id dismatch. streamName=%s, serverID=%s, cache id=%s", streamName, serverID, cacheServerID)
		return
	}
	delete(d.pubStream2ServerID, streamName)
}

func (d *DataManagerMemory) QueryPub(streamName string) (serverID string, exist bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	serverID, exist = d.pubStream2ServerID[streamName]
	return
}
