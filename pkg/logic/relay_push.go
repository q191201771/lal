// Copyright 2020, Chef.  All rights reserved.
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

	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"

	"github.com/q191201771/lal/pkg/rtmp"
)

// TODO chef: 结合Group和Session做一次重构

type RelayPushStatus uint

const (
	RelayPushStatusStart RelayPushStatus = iota
	RelayPushStatusStop
	RelayPushStatusDispose
)

type RelayPush struct {
	IsFresh bool

	url string
	uk  string

	notifyChan chan struct{}

	mutex   sync.Mutex
	t       RelayPushStatus
	session *rtmp.PushSession
}

func NewRelayPush(url string) *RelayPush {
	uk := unique.GenUniqueKey("RELAYPUSH")
	nazalog.Infof("lifecycle new relaypush. [%s] url=%s", uk, url)
	rp := &RelayPush{
		url:        url,
		uk:         uk,
		notifyChan: make(chan struct{}, 1),
		IsFresh:    true,
	}
	go rp.runLoop()
	return rp
}

func (rp *RelayPush) Start() {
	rp.notify(RelayPushStatusStart)
}

func (rp *RelayPush) Stop() {
	rp.notify(RelayPushStatusStop)
}

func (rp *RelayPush) Dispose() {
	nazalog.Infof("lifecycle dispose relaypush. [%s]", rp.uk)
	rp.notify(RelayPushStatusDispose)
}

func (rp *RelayPush) Connected() bool {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	return rp.connected()
}

func (rp *RelayPush) AsyncWrite(msg []byte) error {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	if !rp.connected() {
		return ErrLogic
	}
	return rp.session.AsyncWrite(msg)
}

func (rp *RelayPush) connected() bool {
	return rp.session != nil && rp.session.Status() == rtmp.PushSessionStatusConnected
}

func (rp *RelayPush) runLoop() {
	ticker := time.NewTicker(time.Duration(relayPushCheckIntervalMS) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-rp.notifyChan:
			if rp.do() {
				return
			}
		case <-ticker.C:
			if rp.do() {
				return
			}
		}
	}
}

func (rp *RelayPush) do() (dispose bool) {
	rp.mutex.Lock()
	defer rp.mutex.Unlock()

	switch rp.t {
	case RelayPushStatusStart:
		if rp.session == nil {
			rp.buildNewSession()
		}
		if rp.session.Status() == rtmp.PushSessionStatusError {
			nazalog.Infof("relay push error. [%s] [%s]", rp.uk, rp.session.UniqueKey())
			rp.buildNewSession()
		}
		if rp.session.Status() == rtmp.PushSessionStatusInit {
			go func(s *rtmp.PushSession) {
				nazalog.Infof("start relay push. [%s]", rp.uk)
				err := s.Push(rp.url)
				if err == nil {
					nazalog.Infof("relay push succ. [%s]", rp.uk)
				} else {
					nazalog.Warnf("relay push fail. [%s] err=%+v", rp.uk, err)
				}
			}(rp.session)
		}
		dispose = false
	case RelayPushStatusStop:
		if rp.session != nil && rp.session.Status() == rtmp.PushSessionStatusConnected {
			rp.session.Dispose()
			rp.session = nil
		}
		dispose = false
	case RelayPushStatusDispose:
		if rp.session != nil && rp.session.Status() == rtmp.PushSessionStatusConnected {
			rp.session.Dispose()
			rp.session = nil
		}
		dispose = true
	}
	return
}

func (rp *RelayPush) buildNewSession() {
	rp.session = rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
		option.ConnectTimeoutMS = relayPushConnectTimeoutMS
		option.PushTimeoutMS = relayPushTimeoutMS
		option.WriteAVTimeoutMS = relayPushWriteAVTimeoutMS
	})
	rp.IsFresh = true
}

func (rp *RelayPush) notify(t RelayPushStatus) {
	rp.t = t
	select {
	case rp.notifyChan <- struct{}{}:
	default:
	}
}
