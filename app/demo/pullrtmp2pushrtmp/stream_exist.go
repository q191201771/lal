// Copyright 2021, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"context"
	"time"

	"github.com/cfeeling/lal/pkg/base"
	"github.com/cfeeling/lal/pkg/rtmp"
)

// 检查远端rtmp流是否能正常拉取
func StreamExist(url string) error {
	const (
		timeoutMS = 10000
	)

	errChan := make(chan error, 1)

	ctx, cancel := context.WithTimeout(context.Background(), timeoutMS*time.Millisecond)
	defer cancel()

	s := rtmp.NewPullSession()
	defer s.Dispose()

	go func() {
		// 有的场景只有音频没有视频，所以我们不检查视频
		var hasNotify bool
		var readMetadata bool
		var readAudio bool
		err := s.Pull(url, func(msg base.RTMPMsg) {
			if hasNotify {
				return
			}

			switch msg.Header.MsgTypeID {
			case base.RTMPTypeIDMetadata:
				readMetadata = true
			case base.RTMPTypeIDAudio:
				readAudio = true
			}
			if readMetadata && readAudio {
				hasNotify = true
				errChan <- nil
			}
		})
		if err != nil {
			errChan <- err
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}
