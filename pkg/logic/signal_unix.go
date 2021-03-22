// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

// +build linux darwin netbsd freebsd openbsd dragonfly

package logic

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/cfeeling/naza/pkg/nazalog"
)

func runSignalHandler(cb func()) {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGUSR1, syscall.SIGUSR2)
	s := <-c
	log.Infof("recv signal. s=%+v", s)
	cb()
}
