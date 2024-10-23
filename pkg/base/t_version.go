// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package base

import (
	"fmt"
	"path/filepath"
	"strings"
)

// 版本信息相关
// lal的一部分版本信息使用了naza.bininfo
// 另外，我们也在本文件提供另外一些信息
// 并且将这些信息打入可执行文件、日志、各协议中的标准版本字段中

// LalVersion 整个lal工程的版本号。注意，该变量由外部脚本修改维护，不要手动在代码中修改
var LalVersion = "v0.37.4"

// ConfVersion lalserver的配置文件的版本号
var ConfVersion = "v0.4.1"

// HttpApiVersion lalserver的HTTP-API功能的版本号
var HttpApiVersion = "v0.4.9"

// HttpNotifyVersion lalserver的HTTP-Notify功能的版本号
var HttpNotifyVersion = "v0.2.5"

var HttpWebUiVersion = "v0.0.3"

var (
	LalLibraryName    = "lal"
	LalGithubRepo     = "lal"
	LalGithubSite     = "lal"
	LalDocSite        = "lal"
	LalDocSiteHTTPAPI = "lal"

	LalDefaultConfigFilename = "lalserver.conf.json"

	LalLogo = `
    __    ___    __
   / /   /   |  / /
  / /   / /| | / /
 / /___/ ___ |/ /___
/_____/_/  |_/_____/
`

	// LalFullInfo e.g. lal v0.12.3 (github.com/q191201771/lal)
	LalFullInfo string

	// LalVersionDot e.g. 0.12.3
	LalVersionDot string

	// LalVersionComma e.g. 0,12,3
	LalVersionComma string
)

var (
	// LalRtmpHandshakeWaterMark 植入rtmp握手随机字符串中
	// e.g. lal v0.12.3 (github.com/q191201771/lal)
	LalRtmpHandshakeWaterMark string

	// LalRtmpConnectResultVersion 植入rtmp server中的connect result信令中
	// 注意，有两个object，第一个object中的fmsVer我们保持通用公认的值，在第二个object中植入
	// e.g. 0,12,3
	LalRtmpConnectResultVersion string

	// LalRtmpPushSessionConnectVersion e.g. lal0.12.3
	LalRtmpPushSessionConnectVersion string

	// LalRtmpBuildMetadataEncoder e.g. lal0.12.3
	LalRtmpBuildMetadataEncoder string

	// LalHttpflvPullSessionUa e.g. lal/0.12.3
	LalHttpflvPullSessionUa string

	// LalHttpflvSubSessionServer e.g. lal0.12.3
	LalHttpflvSubSessionServer string

	// LalHlsM3u8Server e.g. lal0.12.3
	LalHlsM3u8Server string

	// LalHlsTsServer e.g. lal0.12.3
	LalHlsTsServer string

	// LalRtspOptionsResponseServer e.g. lal0.12.3
	LalRtspOptionsResponseServer string

	// LalHttptsSubSessionServer e.g. lal0.12.3
	LalHttptsSubSessionServer string

	// LalHttpApiServer e.g. lal0.12.3
	LalHttpApiServer string

	// LalRtspPullSessionUa e.g. lal/0.12.3
	LalRtspPullSessionUa string

	// LalPackSdp e.g. lal 0.12.3
	LalPackSdp string

	// LalRtspRealm e.g. lal
	LalRtspRealm string

	LalRtmpRandom1528Buf []byte

	LalFlvHttpResponseHeader []byte

	LalTsHttpResponseHeader []byte

	// LalRtspResponseOptionsTmpl CSeq
	// rfc2326 10.1 OPTIONS
	LalRtspResponseOptionsTmpl string

	// LalDefaultConfFilenameList 没有指定配置文件时，按顺序作为优先级，找到第一个存在的并使用
	LalDefaultConfFilenameList []string

	DespPageNotFound string
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

func MockVersion(
	lalVersion, confVersion, httpApiVersion, httpNotifyVersion, httpWebUiVersion,
	lalLibraryName, lalGithubRepo, lalGithubSite, lalDocSite, lalDocSiteHTTPAPI,
	lalDefaultConfigFilename, lalLogo string) {
	LalVersion = lalVersion
	ConfVersion = confVersion
	HttpApiVersion = httpApiVersion
	HttpNotifyVersion = httpNotifyVersion
	HttpWebUiVersion = httpWebUiVersion

	LalLibraryName = lalLibraryName
	LalGithubRepo = lalGithubRepo
	LalGithubSite = lalGithubSite
	LalDocSite = lalDocSite
	LalDocSiteHTTPAPI = lalDocSiteHTTPAPI

	LalDefaultConfigFilename = lalDefaultConfigFilename
	LalLogo = lalLogo

	initial()
}

func initial() {
	LalFullInfo = LalLibraryName + " " + LalVersion + " (" + LalGithubRepo + ")"
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

	LalRtspRealm = LalLibraryName

	DespPageNotFound = "page not found, check this document out: " + LalDocSiteHTTPAPI

	LalRtmpRandom1528Buf = make([]byte, 1528)
	hack := []byte(fmt.Sprintf("random buf of rtmp handshake gen by %s", LalRtmpHandshakeWaterMark))
	for i := 0; i < 1528; i += len(hack) {
		copy(LalRtmpRandom1528Buf[i:], hack)
	}

	LalFlvHttpResponseHeader = []byte("HTTP/1.1 200 OK\r\n" +
		"Server: " + LalHttpflvSubSessionServer + "\r\n" +
		"Cache-Control: no-cache\r\n" +
		"Content-Type: video/x-flv\r\n" +
		"Connection: close\r\n" +
		"Expires: -1\r\n" +
		"Pragma: no-cache\r\n" +
		CorsHeaders +
		"\r\n")

	LalTsHttpResponseHeader = []byte("HTTP/1.1 200 OK\r\n" +
		"Server: " + LalHttptsSubSessionServer + "\r\n" +
		"Cache-Control: no-cache\r\n" +
		"Content-Type: video/mp2t\r\n" +
		"Connection: close\r\n" +
		"Expires: -1\r\n" +
		"Pragma: no-cache\r\n" +
		CorsHeaders +
		"\r\n")

	LalRtspResponseOptionsTmpl = "RTSP/1.0 200 OK\r\n" +
		"Server: " + LalRtspOptionsResponseServer + "\r\n" +
		"CSeq: %s\r\n" +
		"Public: DESCRIBE, ANNOUNCE, SETUP, PLAY, PAUSE, RECORD, TEARDOWN\r\n" +
		"\r\n"

	LalDefaultConfFilenameList = []string{
		filepath.FromSlash(LalDefaultConfigFilename),
		filepath.FromSlash("./conf/" + LalDefaultConfigFilename),
		filepath.FromSlash("../" + LalDefaultConfigFilename),
		filepath.FromSlash("../conf/" + LalDefaultConfigFilename),
		filepath.FromSlash("../../" + LalDefaultConfigFilename),
		filepath.FromSlash("../../conf/" + LalDefaultConfigFilename),
		filepath.FromSlash("../../../" + LalDefaultConfigFilename),
		filepath.FromSlash("../../../conf/" + LalDefaultConfigFilename),
		filepath.FromSlash("lal/conf/" + LalDefaultConfigFilename),
	}
}

func init() {
	initial()
}
