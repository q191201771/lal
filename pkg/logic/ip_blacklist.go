// Copyright 2024, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"sync"
	"time"
)

type IpBlacklist struct {
	mu  sync.Mutex
	ips map[string]int64 // TODO(chef): 优化性能 202405
}

func (l *IpBlacklist) Add(ip string, durationSec int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.ips == nil {
		l.ips = make(map[string]int64)
	}

	until := time.Now().Unix() + int64(durationSec)
	l.ips[ip] = until
}

func (l *IpBlacklist) Has(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.eraseStale()

	_, ok := l.ips[ip]
	return ok
}

func (l *IpBlacklist) eraseStale() {
	now := time.Now().Unix()

	stales := make(map[string]struct{})

	for ip, until := range l.ips {
		if until < now {
			stales[ip] = struct{}{}
		}
	}

	for ip := range stales {
		Log.Debugf("erase ip from blacklist. ip=%s", ip)
		delete(l.ips, ip)
	}
}
