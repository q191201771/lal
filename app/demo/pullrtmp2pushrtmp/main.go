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
	"time"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/nazalog"
)

func main() {
	_ = nazalog.Init(func(option *nazalog.Option) {
		option.AssertBehavior = nazalog.AssertFatal
	})
	defer nazalog.Sync()
	base.LogoutStartInfo()

	i := flag.String("i", "", "specify pull rtmp url")
	o := flag.String("o", "", "specify push rtmp url list, separated by a comma")
	flag.Parse()
	if *i == "" || *o == "" {
		flag.Usage()
		_, _ = fmt.Fprintf(os.Stderr, `Example:
  %s -i rtmp://127.0.0.1/live/test110 -o rtmp://127.0.0.1/live/test220
  %s -i rtmp://127.0.0.1/live/test110 -o rtmp://127.0.0.1/live/test220,rtmp://127.0.0.1/live/test330
`, os.Args[0], os.Args[0])
		base.OsExitAndWaitPressIfWindows(1)
	}

	ol := strings.Split(*o, ",")
	for i := range ol {
		ol[i] = strings.TrimSpace(ol[i])
	}

	t := NewTunnel(*i, ol)
	ec := t.Start()
	if ec.err != nil {
		nazalog.Errorf("tunnel start failed. err=%+v", ec)
		return
	}
	defer t.Close()
	ec = <-t.Wait()
	nazalog.Errorf("< tunnel wait. err=%+v", ec)

	time.Sleep(60 * time.Second)
}
