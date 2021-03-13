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

const detailFilename = "delay.txt"

type PullType int

const (
	PullTypeUnknown PullType = iota
	PullTypeRTMP
	PullTypeHTTPFLV
)

func main() {
	tagKey2writeTime := make(map[string]time.Time)
	var delays []int64
	var mu sync.Mutex

	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})
	filename, pushURL, pullURL, pullType := parseFlag()

	tags, err := httpflv.ReadAllTagsFromFLVFile(filename)
	nazalog.Assert(nil, err)
	nazalog.Infof("read tags from flv file succ. len of tags=%d", len(tags))

	pushSession := rtmp.NewPushSession()
	err = pushSession.Push(pushURL)
	nazalog.Assert(nil, err)
	nazalog.Info("push succ.")
	//defer pushSession.Dispose()

	var rtmpPullSession *rtmp.PullSession
	var httpflvPullSession *httpflv.PullSession

	handleReadPayloadFn := func(payload []byte) {
		tagKey := nazamd5.MD5(payload)
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
	case PullTypeHTTPFLV:
		httpflvPullSession = httpflv.NewPullSession()
		err = httpflvPullSession.Pull(pullURL, func(tag httpflv.Tag) {
			handleReadPayloadFn(tag.Payload())
		})
		nazalog.Assert(nil, err)
	case PullTypeRTMP:
		rtmpPullSession = rtmp.NewPullSession()
		err = rtmpPullSession.Pull(pullURL, func(msg base.RTMPMsg) {
			handleReadPayloadFn(msg.Payload)
		})
		nazalog.Assert(nil, err)
		//defer pullSession.Dispose()
	}

	nazalog.Info("pull succ.")

	go func() {
		for {
			time.Sleep(5 * time.Second)
			pushSession.UpdateStat(1)
			var pullBitrate int
			switch pullType {
			case PullTypeRTMP:
				rtmpPullSession.UpdateStat(1)
				pullBitrate = rtmpPullSession.GetStat().Bitrate
			case PullTypeHTTPFLV:
				httpflvPullSession.UpdateStat(1)
				pullBitrate = httpflvPullSession.GetStat().Bitrate
			}
			nazalog.Debugf("stat bitrate. push=%+v, pull=%+v", pushSession.GetStat().Bitrate, pullBitrate)
		}
	}()

	prevTS := int64(-1)
	for _, tag := range tags {
		h := remux.FLVTagHeader2RTMPHeader(tag.Header)
		chunks := rtmp.Message2Chunks(tag.Raw[11:11+h.MsgLen], &h)

		if prevTS >= 0 && int64(h.TimestampAbs) > prevTS {
			diff := int64(h.TimestampAbs) - prevTS
			time.Sleep(time.Duration(diff) * time.Millisecond)
		}
		prevTS = int64(h.TimestampAbs)

		mu.Lock()
		tagKey := nazamd5.MD5(tag.Raw[11 : 11+h.MsgLen])
		if _, exist := tagKey2writeTime[tagKey]; exist {
			nazalog.Errorf("tag key already exist. key=%s", tagKey)
		}
		tagKey2writeTime[tagKey] = time.Now()
		mu.Unlock()

		err = pushSession.AsyncWrite(chunks)
		nazalog.Assert(nil, err)
		//nazalog.Debugf("sent. %d", i)
	}

	min := int64(2147483647)
	max := int64(0)
	avg := int64(0)
	sum := int64(0)
	fp, _ := os.Create(detailFilename)
	defer fp.Close()
	for _, d := range delays {
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
		sum += d
		_, _ = fp.WriteString(fmt.Sprintf("%d\n", d))
	}
	if len(delays) > 0 {
		avg = sum / int64(len(delays))
	}
	nazalog.Debugf("len(tagKey2writeTime)=%d, delays(len=%d, avg=%d, min=%d, max=%d), detailFilename=%s", len(tagKey2writeTime), len(delays), avg, min, max, detailFilename)
}

func parseFlag() (filename, pushURL, pullURL string, pullType PullType) {
	f := flag.String("f", "", "specify flv file")
	o := flag.String("o", "", "specify rtmp/httpflv push url")
	i := flag.String("i", "", "specify rtmp/httpflv pull url")
	flag.Parse()
	if strings.HasPrefix(*i, "rtmp") {
		pullType = PullTypeRTMP
	} else if strings.HasSuffix(*i, ".flv") {
		pullType = PullTypeHTTPFLV
	} else {
		pullType = PullTypeUnknown
	}
	if *f == "" || *i == "" || *o == "" || pullType == PullTypeUnknown {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  %s -f test.flv -o rtmp://127.0.0.1:1935/live/test -i rtmp://127.0.0.1:1935/live/test
  %s -f test.flv -o rtmp://127.0.0.1:1935/live/test -i http://127.0.0.1:8080/live/test.flv
`, os.Args[0], os.Args[0])
		base.OSExitAndWaitPressIfWindows(1)
	}
	filename = *f
	pushURL = *o
	pullURL = *i
	return
}
