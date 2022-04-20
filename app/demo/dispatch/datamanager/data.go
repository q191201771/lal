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
	"time"

	"github.com/q191201771/naza/pkg/nazalog"
)

type DataManagerMemory struct {
	serverTimeoutSec    int
	mutex               sync.Mutex
	serverId2pubStreams map[string]map[string]struct{}
	serverId2AliveTs    map[string]int64
}

func NewDataManagerMemory(serverTimeoutSec int) *DataManagerMemory {
	d := &DataManagerMemory{
		serverTimeoutSec:    serverTimeoutSec,
		serverId2pubStreams: make(map[string]map[string]struct{}),
		serverId2AliveTs:    make(map[string]int64),
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
			for serverId, ts := range d.serverId2AliveTs {
				if now > ts && now-ts > int64(d.serverTimeoutSec) {
					nazalog.Warnf("server timeout. serverId=%s", serverId)
					delete(d.serverId2pubStreams, serverId)
				}
			}

			// 定时打印数据日志
			if count%60 == 0 {
				nazalog.Infof("data info. %+v", d.serverId2pubStreams)
			}

			d.mutex.Unlock()
		}
	}()

	return d
}

func (d *DataManagerMemory) AddPub(streamName, serverId string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	pss, _ := d.serverId2pubStreams[serverId]
	if pss == nil {
		pss = make(map[string]struct{})
	}
	pss[streamName] = struct{}{}
}

func (d *DataManagerMemory) DelPub(streamName, serverId string) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	actualServerId, _ := d.queryPub(streamName)
	if actualServerId != serverId {
		return
	}
	delete(d.serverId2pubStreams[serverId], streamName)
}

func (d *DataManagerMemory) QueryPub(streamName string) (serverId string, exist bool) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.queryPub(streamName)
}

func (d *DataManagerMemory) UpdatePub(serverId string, streamNameList []string) {
	// 3. server超时，去掉所有上面所有的pub

	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.markAlive(serverId)

	// 更新serverId对应的stream列表
	pss := make(map[string]struct{})
	for _, s := range streamNameList {
		pss[s] = struct{}{}
	}
	cpss := d.serverId2pubStreams[serverId]
	d.serverId2pubStreams[serverId] = pss

	// only for log
	for s := range pss {
		if _, exist := cpss[s]; !exist {
			nazalog.Warnf("update pub, add. serverId=%s, streamName=%s", serverId, s)
		}
	}
	for s := range cpss {
		if _, exist := pss[s]; !exist {
			nazalog.Warnf("update pub, del. serverId=%s, streamName=%s", serverId, s)
		}
	}
}

func (d *DataManagerMemory) queryPub(streamName string) (string, bool) {
	for serverId, pss := range d.serverId2pubStreams {
		if _, exist := pss[streamName]; exist {
			return serverId, true
		}
	}
	return "", false
}

func (d *DataManagerMemory) markAlive(serverId string) {
	d.serverId2AliveTs[serverId] = time.Now().Unix()
}
