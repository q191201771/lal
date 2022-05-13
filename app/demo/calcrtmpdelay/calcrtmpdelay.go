// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/remux"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
	"github.com/q191201771/naza/pkg/nazamd5"
)

const outDetailFilename = "delay.txt"

type PullType int

const (
	PullTypeUnknown PullType = iota
	PullTypeRtmp
	PullTypeHttpflv
)

func (pt PullType) Readable() string {
	switch pt {
	case PullTypeUnknown:
		return "unknown"
	case PullTypeRtmp:
		return "rtmp"
	case PullTypeHttpflv:
		return "httpflv"
	}

	// never reach here
	return "fxxk"
}

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})
	defer nazalog.Sync()
	base.LogoutStartInfo()

	var mu sync.Mutex
	tagKey2writeTime := make(map[string]time.Time)
	var delays []int64

	filename, pushUrl, pullUrl, pullType := parseFlag()
	nazalog.Infof("parse flag succ. filename=%s, pushUrl=%s, pullUrl=%s, pullType=%s",
		filename, pushUrl, pullUrl, pullType.Readable())

	pushSession := rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
		option.PushTimeoutMs = 10000
	})
	err := pushSession.Push(pushUrl)
	if err != nil {
		nazalog.Fatalf("push rtmp failed. url=%s, err=%+v", pushUrl, err)
	}
	nazalog.Info("push rtmp succ.")
	defer pushSession.Dispose()

	var rtmpPullSession *rtmp.PullSession
	var httpflvPullSession *httpflv.PullSession

	handleReadPayloadFn := func(payload []byte) {
		tagKey := nazamd5.Md5(payload)
		mu.Lock()
		t, exist := tagKey2writeTime[tagKey]
		if !exist {
			nazalog.Errorf("tag key not exist.")
		} else {
			delay := time.Now().Sub(t).Milliseconds()
			delays = append(delays, delay)
			delete(tagKey2writeTime, tagKey)
		}
		mu.Unlock()
	}

	switch pullType {
	case PullTypeHttpflv:
		httpflvPullSession = httpflv.NewPullSession()
		err = httpflvPullSession.Pull(pullUrl, func(tag httpflv.Tag) {
			handleReadPayloadFn(tag.Payload())
		})
		if err != nil {
			nazalog.Fatalf("pull flv failed. err=%+v", err)
		}
		nazalog.Info("pull flv succ.")
		defer httpflvPullSession.Dispose()
	case PullTypeRtmp:
		rtmpPullSession = rtmp.NewPullSession().WithOnReadRtmpAvMsg(func(msg base.RtmpMsg) {
			handleReadPayloadFn(msg.Payload)
		})
		err = rtmpPullSession.Pull(pullUrl)
		if err != nil {
			nazalog.Fatalf("pull rtmp failed. err=%+v", err)
		}
		nazalog.Info("pull rtmp succ.")
		defer rtmpPullSession.Dispose()
	}

	go func() {
		for {
			time.Sleep(5 * time.Second)
			pushSession.UpdateStat(5)
			var pullBitrate int
			switch pullType {
			case PullTypeRtmp:
				rtmpPullSession.UpdateStat(5)
				pullBitrate = rtmpPullSession.GetStat().Bitrate
			case PullTypeHttpflv:
				httpflvPullSession.UpdateStat(5)
				pullBitrate = httpflvPullSession.GetStat().Bitrate
			}
			nazalog.Debugf("stat bitrate. push=%+v, pull=%+v", pushSession.GetStat().Bitrate, pullBitrate)
		}
	}()

	// 读取flv文件
	flvFilePump := httpflv.NewFlvFilePump(func(option *httpflv.FlvFilePumpOption) {
		option.IsRecursive = false
	})
	err = flvFilePump.Pump(filename, func(tag httpflv.Tag) bool {
		chunks := remux.FlvTag2RtmpChunks(tag)

		mu.Lock()
		tagKey := nazamd5.Md5(tag.Payload())
		if _, exist := tagKey2writeTime[tagKey]; exist {
			nazalog.Errorf("tag key already exist. key=%s", tagKey)
		}
		tagKey2writeTime[tagKey] = time.Now()
		mu.Unlock()

		err = pushSession.Write(chunks)
		if err != nil {
			nazalog.Fatalf("write failed. err=%+v", err)
			return false
		}
		return true
	})
	if err != nil {
		nazalog.Fatalf("pump flv file failed. err=%+v", err)
	}

	_ = pushSession.Flush()
	time.Sleep(300 * time.Millisecond)

	// 信息分析汇总输出
	min := int64(2147483647)
	max := int64(0)
	avg := int64(0)
	sum := int64(0)
	outDetailFp, _ := os.Create(outDetailFilename)
	defer outDetailFp.Close()
	for _, d := range delays {
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
		sum += d
		_, _ = outDetailFp.WriteString(fmt.Sprintf("%d\n", d))
	}
	if len(delays) > 0 {
		avg = sum / int64(len(delays))
	}
	nazalog.Debugf("len(tagKey2writeTime)=%d, delays(len=%d, avg=%d, min=%d, max=%d), detailFilename=%s", len(tagKey2writeTime), len(delays), avg, min, max, outDetailFilename)
}

func parseFlag() (filename, pushUrl, pullUrl string, pullType PullType) {
	f := flag.String("f", "", "specify flv file")
	o := flag.String("o", "", "specify rtmp/httpflv push url")
	i := flag.String("i", "", "specify rtmp/httpflv pull url")
	flag.Parse()
	if strings.HasPrefix(*i, "rtmp") {
		pullType = PullTypeRtmp
	} else if strings.HasSuffix(*i, ".flv") {
		pullType = PullTypeHttpflv
	} else {
		pullType = PullTypeUnknown
	}
	if *f == "" || *i == "" || *o == "" || pullType == PullTypeUnknown {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  %s -f test.flv -o rtmp://127.0.0.1:1935/live/test -i rtmp://127.0.0.1:1935/live/test
  %s -f test.flv -o rtmp://127.0.0.1:1935/live/test -i http://127.0.0.1:8080/live/test.flv
`, os.Args[0], os.Args[0])
		base.OsExitAndWaitPressIfWindows(1)
	}
	filename = *f
	pushUrl = *o
	pullUrl = *i
	return
}
