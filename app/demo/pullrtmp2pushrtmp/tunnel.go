// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"encoding/hex"
	"errors"
	"time"

	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/naza/pkg/nazastring"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"
)

var ErrClosedByCaller = errors.New("tunnel closed by caller")

type Tunnel struct {
	uk         string
	inURL      string
	outURLList []string
	pushECChan chan ErrorCode
	closeChan  chan struct{}
	waitChan   chan ErrorCode
	rtmpMsgQ   chan base.RTMPMsg

	pullSession     *rtmp.PullSession
	pushSessionList []*rtmp.PushSession
}

type ErrorCode struct {
	code int // -1表示拉流失败或者结束，>=0表示推流失败或者结束，值对应outURLList的下标
	err  error
}

// @param inURL      拉流rtmp url地址
// @param outURLList 推流rtmp url地址列表
//
func NewTunnel(inURL string, outURLList []string) *Tunnel {
	return &Tunnel{
		uk:         unique.GenUniqueKey("TUNNEL"),
		inURL:      inURL,
		outURLList: outURLList,
		pushECChan: make(chan ErrorCode, len(outURLList)),
		closeChan:  make(chan struct{}, 1),
		waitChan:   make(chan ErrorCode, len(outURLList)+1),
		rtmpMsgQ:   make(chan base.RTMPMsg, 1024),
	}
}

// @return err 为nil时，表示任务启动成功，拉流和推流通道都已成功建立，并开始转推数据
//             不为nil时，表示任务失败，可以通过`code`得到是拉流还是推流失败
func (t *Tunnel) Start() ErrorCode {
	const (
		pullTimeoutMS   = 10000
		pushTimeoutMS   = 10000
		statIntervalSec = 5
	)

	nazalog.Infof("[%s] new tunnel. inURL=%s, outURLList=%+v", t.uk, t.inURL, t.outURLList)

	for i, outURL := range t.outURLList {
		pushSession := rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
			option.PushTimeoutMS = pushTimeoutMS
		})
		nazalog.Infof("[%s] start push. [%s] url=%s", t.uk, pushSession.UniqueKey(), outURL)

		err := pushSession.Push(outURL)
		if err != nil {
			nazalog.Errorf("[%s] push error. [%s] err=%+v", t.uk, pushSession.UniqueKey(), err)
			return ErrorCode{i, err}
		}
		nazalog.Infof("[%s] push succ. [%s]", t.uk, pushSession.UniqueKey())

		t.pushSessionList = append(t.pushSessionList, pushSession)

		go func(ii int, u string, s *rtmp.PushSession) {
			for {
				select {
				case err := <-s.Wait():
					nazalog.Errorf("[%s] push wait error. [%s] err=%+v", t.uk, s.UniqueKey(), err)
					t.pushECChan <- ErrorCode{ii, err}
					return
				}
			}
		}(i, outURL, pushSession)
	}

	t.pullSession = rtmp.NewPullSession(func(option *rtmp.PullSessionOption) {
		option.PullTimeoutMS = pullTimeoutMS
	})
	nazalog.Infof("[%s] start pull. [%s] url=%s", t.uk, t.pullSession.UniqueKey(), t.inURL)

	err := t.pullSession.Pull(t.inURL, func(msg base.RTMPMsg) {
		m := msg.Clone()
		t.rtmpMsgQ <- m
	})
	if err != nil {
		nazalog.Errorf("[%s] pull error. [%s] err=%+v", t.uk, t.pullSession.UniqueKey(), err)
		return ErrorCode{-1, err}
	}
	nazalog.Infof("[%s] pull succ. [%s]", t.uk, t.pullSession.UniqueKey())

	go func() {
		debugWriteCount := 0
		maxDebugWriteCount := 5
		ticker := time.NewTicker(statIntervalSec * time.Second)
		defer ticker.Stop()

		defer func() {
			if t.pullSession != nil {
				nazalog.Infof("[%s] dispose pull session. [%s]", t.uk, t.pullSession.UniqueKey())
				t.pullSession.Dispose()
			}

			for _, s := range t.pushSessionList {
				nazalog.Infof("[%s] dispose push session. [%s]", t.uk, s.UniqueKey())
				s.Dispose()
			}
		}()

		for {
			select {
			case err := <-t.pullSession.Wait():
				nazalog.Errorf("[%s] <- pull wait. [%s] err=%+v", t.uk, t.pullSession.UniqueKey(), err)
				t.waitChan <- ErrorCode{-1, err}
				return
			case ec := <-t.pushECChan:
				nazalog.Errorf("[%s] <- pushECChan. err=%+v", t.uk, ec)
				t.waitChan <- ec
				return
			case <-t.closeChan:
				nazalog.Errorf("[%s] <- closeChan.", t.uk)
				t.waitChan <- ErrorCode{-1, ErrClosedByCaller}
				return
			case m := <-t.rtmpMsgQ:
				currHeader := remux.MakeDefaultRTMPHeader(m.Header)
				chunks := rtmp.Message2Chunks(m.Payload, &currHeader)
				if debugWriteCount < maxDebugWriteCount {
					nazalog.Infof("[%s] write. header=%+v, %+v, %s", t.uk, m.Header, currHeader, hex.Dump(nazastring.SubSliceSafety(m.Payload, 32)))
					debugWriteCount++
				}

				for i, pushSession := range t.pushSessionList {
					err := pushSession.AsyncWrite(chunks)
					if err != nil {
						nazalog.Errorf("[%s] write error. err=%+v", t.uk, err)
						t.waitChan <- ErrorCode{i, err}
					}
				}
			case <-ticker.C:
				t.pullSession.UpdateStat(statIntervalSec)
				nazalog.Debugf("[%s] tick pull session stat. [%s] streamName=%s, stat=%+v",
					t.uk, t.pullSession.UniqueKey(), t.pullSession.StreamName(), t.pullSession.GetStat())
				for _, s := range t.pushSessionList {
					s.UpdateStat(statIntervalSec)
					nazalog.Debugf("[%s] tick push session stat. [%s] streamName=%s, stat=%+v",
						t.uk, s.UniqueKey(), s.StreamName(), s.GetStat())
				}
			}
		}
	}()

	return ErrorCode{0, nil}
}

// `Start`函数调用成功后，可调用`Wait`函数，等待任务结束
// `Start`函数调用失败后，请不要调用`Wait`函数，否则行为未定义
//
func (t *Tunnel) Wait() chan ErrorCode {
	return t.waitChan
}

// `Start`函数调用成功后，可调用`Close`函数，主动关闭转推任务
// `Start`函数调用失败后，请不要调用`Close`函数，否则行为未定义
// `Close`函数允许调用多次
//
func (t *Tunnel) Close() {
	t.closeChan <- struct{}{}

}
