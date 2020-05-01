// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

// +build linux darwin netbsd freebsd openbsd dragonfly

package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/q191201771/naza/pkg/nazalog"
)

func runSignalHandler() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGUSR1, syscall.SIGUSR2)
	s := <-c
	log.Infof("recv signal. s=%+v", s)
	sm.Dispose()
}
