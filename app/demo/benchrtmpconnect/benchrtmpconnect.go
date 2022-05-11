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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/nazalog"
)

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
		option.Level = nazalog.LevelLogNothing
	})
	defer nazalog.Sync()
	base.LogoutStartInfo()

	urlTmpl, num := parseFlag()
	urls := collect(urlTmpl, num)

	var mu sync.Mutex
	var succCosts []int64
	var failCosts []int64
	var wg sync.WaitGroup
	wg.Add(len(urls))

	go func() {
		for {
			mu.Lock()
			succ := len(succCosts)
			fail := len(failCosts)
			mu.Unlock()
			_, _ = fmt.Fprintf(os.Stderr, "task(num): total=%d, succ=%d, fail=%d\n", len(urls), succ, fail)
			time.Sleep(1 * time.Second)
		}
	}()

	totalB := time.Now()
	for _, url := range urls {
		go func(u string) {
			pullSession := rtmp.NewPullSession(func(option *rtmp.PullSessionOption) {
				option.PullTimeoutMs = 30000
				option.ReadAvTimeoutMs = 30000
				option.HandshakeComplexFlag = false
			})
			b := time.Now()
			err := pullSession.Pull(u)
			e := time.Now()
			cost := e.Sub(b).Milliseconds()
			// 耗时不够1毫秒，我们将值取整到1毫秒，并打印更精确的实际耗时
			if cost == 0 {
				_, _ = fmt.Fprintf(os.Stderr, "round to 1 ms but actual is %s\n", e.Sub(b).String())
				cost = 1
			}

			mu.Lock()
			if err == nil {
				succCosts = append(succCosts, cost)
			} else {
				failCosts = append(failCosts, cost)
			}
			mu.Unlock()
			wg.Done()
		}(url)
	}
	wg.Wait()
	totalE := time.Now()
	totalCost := totalE.Sub(totalB).Milliseconds()
	if totalCost == 0 {
		_, _ = fmt.Fprintf(os.Stderr, "round to 1 ms but actual is %s\n", totalE.Sub(totalB).String())
		totalCost = 1
	}
	min, max, avg := analyse(succCosts)
	_, _ = fmt.Fprintf(os.Stderr, "task(num): total=%d, succ=%d, fail=%d\n", len(urls), len(succCosts), len(failCosts))
	_, _ = fmt.Fprintf(os.Stderr, " cost(ms): total=%d, avg=%d, min=%d, max=%d\n", totalCost, avg, min, max)
}

func analyse(costs []int64) (min, max, avg int64) {
	min = 2147483647
	max = 0
	sum := int64(0)
	for _, cost := range costs {
		if cost < min {
			min = cost
		}
		if cost > max {
			max = cost
		}
		sum += cost
	}
	if len(costs) > 0 {
		avg = sum / int64(len(costs))
	}
	return
}

func collect(urlTmpl string, num int) (urls []string) {
	for i := 0; i < num; i++ {
		url := strings.Replace(urlTmpl, "{i}", strconv.Itoa(i), -1)
		urls = append(urls, url)
	}
	return
}

func parseFlag() (urlTmpl string, num int) {
	i := flag.String("i", "", "specify pull rtmp pull")
	n := flag.Int("n", 0, "specify num of pull connection")
	flag.Parse()
	if *i == "" || *n == 0 {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  %s -i rtmp://127.0.0.1:1935/live/test -n 1000
  %s -i rtmp://127.0.0.1:1935/live/test_{i} -n 1000
`, os.Args[0], os.Args[0])
		base.OsExitAndWaitPressIfWindows(1)
	}
	return *i, *n
}
