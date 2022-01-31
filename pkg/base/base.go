// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"os"
	"strings"
	"time"

	"github.com/q191201771/naza/pkg/bininfo"
	"github.com/q191201771/naza/pkg/nazalog"
)

// base包提供被其他多个package依赖的基础内容，自身不依赖任何package
//
// TODO chef: 考虑部分内容放入关联的协议package的子package中

var startTime string

// ReadableNowTime
//
// TODO(chef): refactor 使用ReadableNowTime
//
func ReadableNowTime() string {
	return time.Now().Format("2006-01-02 15:04:05.999")
}

func LogoutStartInfo() {
	dir, _ := os.Getwd()
	nazalog.Infof("     start: %s", startTime)
	nazalog.Infof("        wd: %s", dir)
	nazalog.Infof("      args: %s", strings.Join(os.Args, " "))
	nazalog.Infof("   bininfo: %s", bininfo.StringifySingleLine())
	nazalog.Infof("   version: %s", LalFullInfo)
	nazalog.Infof("    github: %s", LalGithubSite)
	nazalog.Infof("       doc: %s", LalDocSite)
}

func init() {
	startTime = ReadableNowTime()
}
