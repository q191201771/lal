// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package datamanager

import (
	"sync"
	"time"

	"github.com/cfeeling/naza/pkg/nazalog"
)

type DataManagerMemory struct {
	serverTimeoutSec    int
	mutex               sync.Mutex
	serverID2pubStreams map[string]map[string]struct{}
	serverID2AliveTS    map[string]int64
}

func NewDataManagerMemory(serverTimeoutSec int) *DataManagerMemory {
	d := &DataManagerMemory{
		serverTimeoutSec:    serverTimeoutSec,
		serverID2pubStreams: make(map[string]map[string]struct{}),
		serverID2AliveTS:    make(map[string]int64),
	}

	// TODO chef: release goroutine
	go func() {
		var count int
		for {
			time.Sleep(1 * time.Second)
			count++
			now := time.Now().Unix()

			d.mutex.Lock()
			// 清除长时间没有update报活的节点
			for serverID, ts := range d.serverID2AliveTS {
				if now > ts && now-ts > int64(d.serverTimeoutSec)*1000 {
					nazalog.Warnf("server timeout. serverID=%s", serverID)
					delete(d.serverID2pubStreams, serverID)
				}
			}

			// 定时打印数据日志
			if count%60 == 0 {
				nazalog.Infof("data info. %+v", d.serverID2pubStreams)
			}

			d.mutex.Unlock()
		}
	}()

	return d
}

func (d *DataManagerMemory) AddPub(streamName, serverID string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	pss, _ := d.serverID2pubStreams[serverID]
	if pss == nil {
		pss = make(map[string]struct{})
	}
	pss[streamName] = struct{}{}
}

func (d *DataManagerMemory) DelPub(streamName, serverID string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	actualServerID, _ := d.queryPub(streamName)
	if actualServerID != serverID {
		return
	}
	delete(d.serverID2pubStreams[serverID], streamName)
}

func (d *DataManagerMemory) QueryPub(streamName string) (serverID string, exist bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.queryPub(streamName)
}

func (d *DataManagerMemory) UpdatePub(serverID string, streamNameList []string) {
	// 3. server超时，去掉所有上面所有的pub

	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.markAlive(serverID)

	// 更新serverID对应的stream列表
	pss := make(map[string]struct{})
	for _, s := range streamNameList {
		pss[s] = struct{}{}
	}
	cpss := d.serverID2pubStreams[serverID]
	d.serverID2pubStreams[serverID] = pss

	// only for log
	for s := range pss {
		if _, exist := cpss[s]; !exist {
			nazalog.Warnf("update pub, add. serverID=%s, streamName=%s", serverID, s)
		}
	}
	for s := range cpss {
		if _, exist := pss[s]; !exist {
			nazalog.Warnf("update pub, del. serverID=%s, streamName=%s", serverID, s)
		}
	}
}

func (d *DataManagerMemory) queryPub(streamName string) (string, bool) {
	for serverID, pss := range d.serverID2pubStreams {
		if _, exist := pss[streamName]; exist {
			return serverID, true
		}
	}
	return "", false
}

func (d *DataManagerMemory) markAlive(serverID string) {
	d.serverID2AliveTS[serverID] = time.Now().Unix()
}
