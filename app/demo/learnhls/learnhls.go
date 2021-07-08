// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package main

import (
	"flag"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/lru"
	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
)

// 分析诊断HLS的时间戳。注意，这个程序还没有完成。
//
// TODO chef: 有的代码考虑弄到pkg/hls中

type M3u8PullSession struct {
}

type frag struct {
	extinf   float64
	filename string
}

func parseM3u8(content string) (ret []frag) {
	var err error

	lines := strings.Split(content, "\n")
	var f frag
	for _, line := range lines {
		if strings.HasPrefix(line, "#EXTINF:") {
			line = strings.TrimPrefix(line, "#EXTINF:")
			line = strings.TrimSuffix(line, ",")
			f.extinf, err = strconv.ParseFloat(line, 64)
			nazalog.Assert(nil, err)
		}
		if strings.Index(line, ".ts") != -1 {
			f.filename = line
			ret = append(ret, f)
		}
	}
	return
}

func getTsUrl(m3u8Url string, tsFilename string) string {
	index := strings.LastIndex(m3u8Url, "/")
	nazalog.Assert(true, index != -1)
	path := m3u8Url[:index+1]
	return path + tsFilename
}

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})
	defer nazalog.Sync()

	m3u8Url := parseFlag()
	nazalog.Infof("m3u8 url=%s", m3u8Url)

	cache := lru.New(1024)

	var m sync.Mutex
	var frags []frag

	go func() {
		for {
			content, err := nazahttp.GetHttpFile(m3u8Url, 3000)
			if err != nil {
				nazalog.Error(err)
				return
			}
			//nazalog.Debugf("\n-----m3u8-----\n%s", string(content))

			currFrags := parseM3u8(string(content))
			//nazalog.Debugf("%+v", currFrags)

			m.Lock()
			for _, f := range currFrags {
				if _, exist := cache.Get(f.filename); exist {
					continue
				}
				cache.Put(f.filename, nil)

				nazalog.Infof("> new frag. filename=%s", f.filename)
				frags = append(frags, f)
			}
			m.Unlock()

			time.Sleep(100 * time.Millisecond)
		}
	}()

	for {
		m.Lock()
		currFrags := frags
		frags = nil
		m.Unlock()

		for _, f := range currFrags {
			nazalog.Infof("< new frag. filename=%s", f.filename)
			tsUrl := getTsUrl(m3u8Url, f.filename)
			nazalog.Debug(tsUrl)
			content, err := nazahttp.GetHttpFile(tsUrl, 3000)
			nazalog.Assert(nil, err)
			nazalog.Debugf("TS len=%d", len(content))
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func parseFlag() string {
	url := flag.String("i", "", "specify m3u8 url")
	flag.Parse()
	if *url == "" {
		flag.Usage()
		base.OsExitAndWaitPressIfWindows(1)
	}
	return *url
}
