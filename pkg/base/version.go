// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

// 版本信息相关
// lal的一部分版本信息使用了naza.bininfo，手段是获取git信息。
// 另外，我们也在本文件提供另外一些信息：

// 版本，该变量由build脚本修改维护
var LALVersion = "v0.13.0"

// 以下字段固定不变
var (
	LALLibraryName = "lal"
	LALGitHubRepo  = "github.com/q191201771/lal"
	LALFullInfo    = LALLibraryName + " " + LALVersion + " (" + LALGitHubRepo + ")"
	//LALServerName    = "lalserver"
	//LALGitHubRepoURL = "https://github.com/q191201771/lal"
)

// 作为HTTP客户端时，考虑User-Agent
//
// RTMP metadata
// description     : Bilibili VXCode Swarm Transcoder v0.2.30(gap_fixed:False)
// encoder         : Lavf57.83.100
