// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

// Package base 提供被其他多个package依赖的基础内容，自身不依赖任何package
package base

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/q191201771/naza/pkg/bininfo"
)

// TODO chef: 考虑部分内容放入关联的协议package的子package中

var startTime string

var readableTimeLayout = "2006-01-02 15:04:05.999 Z0700 MST"

// ReadableNowTime 当前时间，可读字符串形式
func ReadableNowTime() string {
	return time.Now().Format(readableTimeLayout)
}

func ParseReadableTime(t string) (time.Time, error) {
	return time.Parse(readableTimeLayout, t)
}

func GetWd() string {
	dir, _ := os.Getwd()
	return dir
}

func LogoutStartInfo() {
	Log.Infof("     start: %s", startTime)
	Log.Infof("        wd: %s", GetWd())
	Log.Infof("      args: %s", strings.Join(os.Args, " "))
	Log.Infof("   bininfo: %s", bininfo.StringifySingleLine())
	Log.Infof("   version: %s", LalFullInfo)
	Log.Infof("    github: %s", LalGithubSite)
	Log.Infof("       doc: %s", LalDocSite)
}

func WrapReadConfigFile(theConfigFile string, defaultConfigFiles []string, hookBeforeExit func()) []byte {
	// TODO(chef): 统一本函数内的Log和stderr输出 202405

	// 如果没有指定配置文件，则尝试从默认路径找配置文件
	if theConfigFile == "" {
		Log.Warnf("config file did not specify in the command line, try to load it in the usual path.")
		for _, dcf := range defaultConfigFiles {
			fi, err := os.Stat(dcf)
			if err == nil && fi.Size() > 0 && !fi.IsDir() {
				Log.Warnf("%s exist. using it as config file.", dcf)
				theConfigFile = dcf
				break
			} else {
				Log.Warnf("%s not exist.", dcf)
			}
		}

		// 如果默认路径也没有配置文件，则退出
		if theConfigFile == "" {
			flag.Usage()
			if hookBeforeExit != nil {
				hookBeforeExit()
			}
			OsExitAndWaitPressIfWindows(1)
		}
	}

	// 读取配置文件
	rawContent, err := os.ReadFile(theConfigFile)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "read conf file failed. file=%s err=%+v", theConfigFile, err)
		OsExitAndWaitPressIfWindows(1)
	}
	return rawContent
}

func init() {
	startTime = ReadableNowTime()
}
