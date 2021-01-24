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

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazastring"
	"github.com/q191201771/naza/pkg/unique"
)

// 拉取一路rtmp流，并使用rtmp转推出去，可1对n转推
// 阻塞直至拉流或者（任意一路）推流失败或结束
//
// @param inURL  拉流rtmp url地址
// @param outURL 推流rtmp url地址列表
//
func PullRTMP2PushRTMP(inURL string, outURLList []string) error {
	const (
		pullTimeoutMS = 10000
		pushTimeoutMS = 10000
	)

	tunnelUK := unique.GenUniqueKey("TUNNEL")
	nazalog.Infof("[%s] new tunnel. inURL=%s, outURLList=%+v", tunnelUK, inURL, outURLList)

	errChan := make(chan error, len(outURLList)+1)
	rtmpMsgQ := make(chan base.RTMPMsg, 1024)

	var pullSession *rtmp.PullSession
	var pushSessionList []*rtmp.PushSession

	defer func() {
		if pullSession != nil {
			nazalog.Infof("[%s] dispose pull session. [%s]", tunnelUK, pullSession.UniqueKey())
			pullSession.Dispose()
		}
		for _, s := range pushSessionList {
			nazalog.Infof("[%s] dispose push session. [%s]", tunnelUK, s.UniqueKey())
			s.Dispose()
		}
	}()

	for _, outURL := range outURLList {
		pushSession := rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
			option.PushTimeoutMS = pushTimeoutMS
		})
		nazalog.Infof("[%s] start push. [%s] url=%s", tunnelUK, pushSession.UniqueKey(), outURL)

		err := pushSession.Push(outURL)
		if err != nil {
			nazalog.Errorf("[%s] push error. [%s] err=%+v", tunnelUK, pushSession.UniqueKey(), err)
			return err
		}
		nazalog.Infof("[%s] push succ. [%s]", tunnelUK, pushSession.UniqueKey())

		pushSessionList = append(pushSessionList, pushSession)
		go func(u string, s *rtmp.PushSession) {
			err := <-s.Wait()
			nazalog.Errorf("[%s] push wait error. [%s] err=%+v", tunnelUK, s.UniqueKey(), err)
			errChan <- err
		}(outURL, pushSession)
	}

	pullSession = rtmp.NewPullSession(func(option *rtmp.PullSessionOption) {
		option.PullTimeoutMS = pullTimeoutMS
	})
	nazalog.Infof("[%s] start pull. [%s] url=%s", tunnelUK, pullSession.UniqueKey(), inURL)

	err := pullSession.Pull(inURL, func(msg base.RTMPMsg) {
		m := msg.Clone()
		rtmpMsgQ <- m
	})
	if err != nil {
		nazalog.Errorf("[%s] pull error. [%s] err=%+v", tunnelUK, pullSession.UniqueKey(), err)
		return err
	}
	nazalog.Infof("[%s] pull succ. [%s]", tunnelUK, pullSession.UniqueKey())

	go func(u string, s *rtmp.PullSession) {
		err := <-s.Wait()
		nazalog.Errorf("[%s] pull wait error. [%s] err=%+v", tunnelUK, s.UniqueKey(), err)
		errChan <- err
	}(inURL, pullSession)

	debugWriteCount := 0
	maxDebugWriteCount := 5
	for {
		select {
		case err := <-errChan:
			nazalog.Errorf("[%s] errChan. err=%+v", tunnelUK, err)
			return err
		case msg := <-rtmpMsgQ:
			currHeader := remux.MakeDefaultRTMPHeader(msg.Header)
			chunks := rtmp.Message2Chunks(msg.Payload, &currHeader)
			if debugWriteCount < maxDebugWriteCount {
				nazalog.Infof("[%s] write. header=%+v, %+v, %s", tunnelUK, msg.Header, currHeader, hex.Dump(nazastring.SubSliceSafety(msg.Payload, 32)))
				debugWriteCount++
			}
			for _, pushSession := range pushSessionList {
				err := pushSession.AsyncWrite(chunks)
				if err != nil {
					nazalog.Errorf("[%s] write error. err=%+v", tunnelUK, err)
					return err
				}
			}
		}
	}
}
