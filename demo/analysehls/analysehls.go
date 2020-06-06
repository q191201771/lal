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
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/q191201771/naza/pkg/lru"
	"github.com/q191201771/naza/pkg/nazahttp"
	"github.com/q191201771/naza/pkg/nazalog"
)

// 分析诊断HLS的时间戳。注意，这个程序还没有完成。
//
// TODO chef: 有的代码考虑弄到pkg/hls中

type M3U8PullSession struct {
}

type frag struct {
	extinf   float64
	filename string
}

func parseM3U8(content string) (ret []frag) {
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

func getTSURL(m3u8URL string, tsFilename string) string {
	index := strings.LastIndex(m3u8URL, "/")
	nazalog.Assert(true, index != -1)
	path := m3u8URL[:index+1]
	return path + tsFilename
}

func main() {
	m3u8URL := parseFlag()
	nazalog.Infof("m3u8 url=%s", m3u8URL)

	cache := lru.New(1024)

	var m sync.Mutex
	var frags []frag

	go func() {
		for {
			content, err := nazahttp.GetHTTPFile(m3u8URL, 3000)
			if err != nil {
				nazalog.Error(err)
				return
			}
			//nazalog.Debugf("\n-----m3u8-----\n%s", string(content))

			currFrags := parseM3U8(string(content))
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
			tsURL := getTSURL(m3u8URL, f.filename)
			nazalog.Debug(tsURL)
			content, err := nazahttp.GetHTTPFile(tsURL, 3000)
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
		os.Exit(1)
	}
	return *url
}
