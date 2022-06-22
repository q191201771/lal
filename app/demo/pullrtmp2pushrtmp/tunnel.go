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
	"fmt"
	"time"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazabytes"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/unique"
)

// 注意，当前的策略是，当推流有多个地址时，任意一个失败就会退出整个任务

var ErrClosedByCaller = errors.New("tunnel closed by caller")

type Tunnel struct {
	uk          string
	inUrl       string
	outUrlList  []string
	startTime   time.Time
	startEcChan chan ErrorCode
	pullEcChan  chan ErrorCode
	pushEcChan  chan ErrorCode
	closeChan   chan ErrorCode
	waitChan    chan ErrorCode
	rtmpMsgQ    chan base.RtmpMsg

	pullSession     *rtmp.PullSession
	pushSessionList []*rtmp.PushSession
}

type ErrorCode struct {
	code int // -1表示拉流失败或者结束，>=0表示推流失败或者结束，值对应outUrlList的下标
	err  error
}

// @param inUrl      拉流rtmp url地址
// @param outUrlList 推流rtmp url地址列表
//
func NewTunnel(inUrl string, outUrlList []string) *Tunnel {
	var streamName string
	ctx, err := base.ParseRtmpUrl(inUrl)
	if err != nil {
		nazalog.Errorf("parse rtmp url failed. url=%s", inUrl)
		streamName = "invalid"
	} else {
		streamName = ctx.LastItemOfPath
	}
	originUk := unique.GenUniqueKey("TUNNEL")
	uk := fmt.Sprintf("%s-%s", originUk, streamName)

	return &Tunnel{
		uk:          uk,
		inUrl:       inUrl,
		outUrlList:  outUrlList,
		startTime:   time.Now(),
		startEcChan: make(chan ErrorCode, len(outUrlList)+1),
		pullEcChan:  make(chan ErrorCode, 1),
		pushEcChan:  make(chan ErrorCode, len(outUrlList)),
		closeChan:   make(chan ErrorCode, 1),
		waitChan:    make(chan ErrorCode, len(outUrlList)+1),
		rtmpMsgQ:    make(chan base.RtmpMsg, 1024),
	}
}

// @return err 为nil时，表示任务启动成功，拉流和推流通道都已成功建立，并开始转推数据
//             不为nil时，表示任务失败，可以通过`code`得到是拉流还是推流失败
func (t *Tunnel) Start() (ret ErrorCode) {
	const (
		pullTimeoutMs   = 10000
		pushTimeoutMs   = 10000
		statIntervalSec = 5
	)

	nazalog.Infof("[%s] new tunnel. inUrl=%s, outUrlList=%+v", t.uk, t.inUrl, t.outUrlList)

	defer func() {
		if ret.err != nil {
			t.notifyStartEc(ret)
		}

		go func() {
			nazalog.Debugf("[%s] > main event loop.", t.uk)
			debugWriteCount := 0
			maxDebugWriteCount := 5
			ticker := time.NewTicker(statIntervalSec * time.Second)
			defer ticker.Stop()

			// 最后清理所有session
			defer func() {
				nazalog.Debugf("[%s] < main event loop. duration=%+v", t.uk, time.Now().Sub(t.startTime))
				if t.pullSession != nil {
					nazalog.Infof("[%s] dispose pull session. [%s]", t.uk, t.pullSession.UniqueKey())
					t.pullSession.Dispose()
				}

				for _, s := range t.pushSessionList {
					nazalog.Infof("[%s] dispose push session. [%s]", t.uk, s.UniqueKey())
					s.Dispose()
				}
			}()

			if t.pullSession != nil {
				go func() {
					nazalog.Debugf("[%s] > pull event loop. %s", t.uk, t.pullSession.UniqueKey())
					for {
						select {
						case err := <-t.pullSession.WaitChan():
							t.notifyPullEc(ErrorCode{-1, err})
							nazalog.Debugf("[%s] < pull event loop. %s", t.uk, t.pullSession.UniqueKey())
							return
						}
					}
				}()
			}

			// 将多个pushSession wait事件聚合在一起
			for i, pushSession := range t.pushSessionList {
				go func(ii int, s *rtmp.PushSession) {
					nazalog.Debugf("[%s] > push event loop. %s", t.uk, s.UniqueKey())
					for {
						select {
						case err := <-s.WaitChan():
							nazalog.Errorf("[%s] push wait error. [%s] err=%+v", t.uk, s.UniqueKey(), err)
							t.notifyPushEc(ErrorCode{ii, err})
							nazalog.Debugf("[%s] < push event loop. %s", t.uk, s.UniqueKey())
							return
						}
					}
				}(i, pushSession)
			}

			// 主事件监听
			for {
				select {
				case ec := <-t.startEcChan:
					nazalog.Errorf("[%s] exit main event loop, <- startEcChan. err=%s", t.uk, ec.Stringify())
					t.notifyWait(ec)
					return
				case ec := <-t.pullEcChan:
					nazalog.Errorf("[%s] exit main event loop, <- pullEcChan. err=%s", t.uk, ec.Stringify())
					t.notifyWait(ec)
					return
				case ec := <-t.pushEcChan:
					nazalog.Errorf("[%s] exit main event loop, <- pushEcChan. err=%s", t.uk, ec.Stringify())
					t.notifyWait(ec)
					return
				case ec := <-t.closeChan:
					nazalog.Errorf("[%s] exit main event loop, <- closeChan.", t.uk)
					t.notifyWait(ec)
					return
				case m := <-t.rtmpMsgQ:
					currHeader := remux.MakeDefaultRtmpHeader(m.Header)
					chunks := rtmp.Message2Chunks(m.Payload, &currHeader)
					if debugWriteCount < maxDebugWriteCount {
						nazalog.Infof("[%s] write. header=%+v, %+v, %s", t.uk, m.Header, currHeader, hex.Dump(nazabytes.Prefix(m.Payload, 32)))
						debugWriteCount++
					}

					for i, pushSession := range t.pushSessionList {
						err := pushSession.Write(chunks)
						if err != nil {
							nazalog.Errorf("[%s] exit main event loop, write error. err=%+v", t.uk, err)
							t.notifyWait(ErrorCode{i, err})
							return
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
	}()

	// 逐个开启push session
	for i, outUrl := range t.outUrlList {
		pushSession := rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
			option.PushTimeoutMs = pushTimeoutMs
		})
		nazalog.Infof("[%s] start push. [%s] url=%s", t.uk, pushSession.UniqueKey(), outUrl)

		err := pushSession.Push(outUrl)
		// 只有有一个失败就直接退出
		if err != nil {
			nazalog.Errorf("[%s] push error. [%s] err=%+v", t.uk, pushSession.UniqueKey(), err)
			ret = ErrorCode{i, err}
			return
		}
		nazalog.Infof("[%s] push succ. [%s]", t.uk, pushSession.UniqueKey())

		// 加入的都是成功的
		t.pushSessionList = append(t.pushSessionList, pushSession)
	}

	t.pullSession = rtmp.NewPullSession(func(option *rtmp.PullSessionOption) {
		option.PullTimeoutMs = pullTimeoutMs
		//option.ReuseReadMessageBufferFlag = false
	}).WithOnReadRtmpAvMsg(func(msg base.RtmpMsg) {
		m := msg.Clone()
		t.rtmpMsgQ <- m
	})
	nazalog.Infof("[%s] start pull. [%s] url=%s", t.uk, t.pullSession.UniqueKey(), t.inUrl)

	err := t.pullSession.Pull(t.inUrl)
	// pull失败就直接退出
	if err != nil {
		nazalog.Errorf("[%s] pull error. [%s] err=%+v", t.uk, t.pullSession.UniqueKey(), err)
		t.pullSession = nil
		ret = ErrorCode{-1, err}
		return
	}
	nazalog.Infof("[%s] pull succ. [%s]", t.uk, t.pullSession.UniqueKey())

	ret = ErrorCode{0, nil}
	return
}

// `Start`函数调用成功后，可调用`Wait`函数，等待任务结束
//
func (t *Tunnel) Wait() chan ErrorCode {
	return t.waitChan
}

// `Start`函数调用成功后，可调用`Close`函数，主动关闭转推任务
// `Close`函数允许调用多次
//
func (t *Tunnel) Close() {
	t.notifyClose()
}

func (t *Tunnel) notifyClose() {
	select {
	case t.closeChan <- ErrorCode{-1, ErrClosedByCaller}:
		nazalog.Debugf("[%s] notifyClose.", t.uk)
	default:
		nazalog.Debugf("[%s] notifyClose fail, ignore.", t.uk)
	}
}

func (t *Tunnel) notifyWait(ec ErrorCode) {
	select {
	case t.waitChan <- ec:
		nazalog.Debugf("[%s] notifyWait. ec=%s", t.uk, ec.Stringify())
	default:
		nazalog.Warnf("[%s] CHEFNOTICEME notifyWait fail, ignore. ec=%s", t.uk, ec.Stringify())
	}
}

func (t *Tunnel) notifyStartEc(ec ErrorCode) {
	select {
	case t.startEcChan <- ec:
		nazalog.Debugf("[%s] notifyStartEc. ec=%s", t.uk, ec.Stringify())
	default:
		nazalog.Warnf("[%s] CHEFNOTICEME notifyStartEc fail, ignore. ec=%s", t.uk, ec.Stringify())
	}
}

func (t *Tunnel) notifyPushEc(ec ErrorCode) {
	select {
	case t.pushEcChan <- ec:
		nazalog.Debugf("[%s] notifyPushEc. ec=%s", t.uk, ec.Stringify())
	default:
		nazalog.Warnf("[%s] CHEFNOTICEME notifyPushEc fail, ignore. ec=%s", t.uk, ec.Stringify())
	}
}

func (t *Tunnel) notifyPullEc(ec ErrorCode) {
	select {
	case t.pullEcChan <- ec:
		nazalog.Debugf("[%s] notifyPullEc. ec=%s", t.uk, ec.Stringify())
	default:
		nazalog.Warnf("[%s] CHEFNOTICEME notifyPullEc fail, ignore. ec=%s", t.uk, ec.Stringify())
	}
}

func (ec *ErrorCode) Stringify() string {
	return fmt.Sprintf("(%d, %+v)", ec.code, ec.err)
}
