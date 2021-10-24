// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import "strings"

// 版本信息相关
// lal的一部分版本信息使用了naza.bininfo
// 另外，我们也在本文件提供另外一些信息
// 并且将这些信息打入可执行文件、日志、各协议中的标准版本字段中

// 版本，该变量由外部脚本修改维护
const LalVersion = "v0.26.0"

var (
	LalLibraryName = "lal"
	LalGithubRepo  = "github.com/q191201771/lal"
	LalGithubSite  = "https://github.com/q191201771/lal"
	LalDocSite     = "https://pengrl.com/lal"

	// e.g. lal v0.12.3 (github.com/q191201771/lal)
	LalFullInfo = LalLibraryName + " " + LalVersion + " (" + LalGithubRepo + ")"

	// e.g. 0.12.3
	LalVersionDot string

	// e.g. 0,12,3
	LalVersionComma string
)

var (
	// 植入rtmp握手随机字符串中
	// e.g. lal v0.12.3 (github.com/q191201771/lal)
	LalRtmpHandshakeWaterMark string

	// 植入rtmp server中的connect result信令中
	// 注意，有两个object，第一个object中的fmsVer我们保持通用公认的值，在第二个object中植入
	// e.g. 0,12,3
	LalRtmpConnectResultVersion string

	// e.g. lal0.12.3
	LalRtmpPushSessionConnectVersion string

	// e.g. lal0.12.3
	LalRtmpBuildMetadataEncoder string

	// e.g. lal/0.12.3
	LalHttpflvPullSessionUa string

	// e.g. lal0.12.3
	LalHttpflvSubSessionServer string

	// e.g. lal0.12.3
	LalHlsM3u8Server string

	// e.g. lal0.12.3
	LalHlsTsServer string

	// e.g. lal0.12.3
	LalRtspOptionsResponseServer string

	// e.g. lal0.12.3
	LalHttptsSubSessionServer string

	// e.g. lal0.12.3
	LalHttpApiServer string

	// e.g. lal/0.12.3
	LalRtspPullSessionUa string

	// e.g. lal 0.12.3
	LalPackSdp string
)

// - rtmp handshake random buf
// - rtmp server(pub & sub)
//     - rtmp message connect result
//         - cdnws: 第一个obj: `fmsVer: FMS/3,0,1,123` 第二个obj: `version: x,x,x,xxx`
// - rtmp client(push & pull)
//     - rtmp message connect
//	       - ffmpeg push: `flashVer: FMLE/3.0 (compatible; Lavf57.83.100)`
//         - ffmpeg pull: `flashVer: LNX 9,0,124,2` -- emulated Flash client version - 9.0.124.2 on Linux
// - rtmp/flv build metadata
//     - encoder: Lavf57.83.100
// - httpflv pull
// 	   - wget: User-Agent: Wget/1.19.1 (darwin15.6.0)
// - httpflv sub
//     - `server:`
// - hls
//     - m3u8
//         - `Server:`
//     - ts
//         - `Server:`
// - rtsp server(pub & sub)
//     - Options response `Server:`
// - rtsp client(pull)
//     - Options User-Agent
//
// - httpts sub
//     - `server:`
//
// - http api
//     - `server:`

func init() {
	LalVersionDot = strings.TrimPrefix(LalVersion, "v")
	LalVersionComma = strings.Replace(LalVersionDot, ".", ",", -1)

	LalRtmpConnectResultVersion = LalVersionComma

	LalRtmpPushSessionConnectVersion = LalLibraryName + LalVersionDot
	LalRtmpBuildMetadataEncoder = LalLibraryName + LalVersionDot
	LalHttpflvSubSessionServer = LalLibraryName + LalVersionDot
	LalHlsM3u8Server = LalLibraryName + LalVersionDot
	LalHlsTsServer = LalLibraryName + LalVersionDot
	LalRtspOptionsResponseServer = LalLibraryName + LalVersionDot
	LalHttptsSubSessionServer = LalLibraryName + LalVersionDot
	LalHttpApiServer = LalLibraryName + LalVersionDot

	LalHttpflvPullSessionUa = LalLibraryName + "/" + LalVersionDot
	LalRtspPullSessionUa = LalLibraryName + "/" + LalVersionDot

	LalRtmpHandshakeWaterMark = LalFullInfo

	LalPackSdp = LalLibraryName + " " + LalVersionDot
}
