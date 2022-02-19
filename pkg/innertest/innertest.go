// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package innertest

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/q191201771/lal/pkg/httpts"
	"github.com/q191201771/naza/pkg/filebatch"

	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/naza/pkg/mock"

	"github.com/q191201771/naza/pkg/nazahttp"

	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/rtsp"
	"github.com/q191201771/lal/pkg/sdp"

	"github.com/q191201771/lal/pkg/remux"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/nazamd5"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/assert"
	"github.com/q191201771/naza/pkg/nazaatomic"
)

// 开启了一个lalserver
// rtmp pub              读取flv文件，使用rtmp协议推送至服务端
// rtmp sub, httpflv sub 分别用rtmp协议以及httpflv协议从服务端拉流，再将拉取的流保存为flv文件
// 对比三份flv文件，看是否完全一致
// hls                   并检查hls生成的m3u8和ts文件，是否和之前的完全一致

// TODO chef:
// - 加上relay push
// - 加上relay pull

var (
	tt *testing.T

	confFile              = "../../testdata/lalserver.conf.json"
	rFlvFileName          = "../../testdata/test.flv"
	wRtmpPullFileName     = "../../testdata/rtmppull.flv"
	wFlvPullFileName      = "../../testdata/flvpull.flv"
	wPlaylistM3u8FileName string
	wRecordM3u8FileName   string
	wHlsTsFilePath        string
	//wRtspPullFileName = "../../testdata/rtsppull.flv"

	pushUrl        string
	httpflvPullUrl string
	httptsPullUrl  string
	rtmpPullUrl    string
	rtspPullUrl    string

	fileTagCount          int
	httpflvPullTagCount   nazaatomic.Uint32
	rtmpPullTagCount      nazaatomic.Uint32
	rtspSdpCtx            sdp.LogicContext
	rtspPullAvPacketCount nazaatomic.Uint32

	httpFlvWriter httpflv.FlvFileWriter
	rtmpWriter    httpflv.FlvFileWriter

	pushSession        *rtmp.PushSession
	httpflvPullSession *httpflv.PullSession
	rtmpPullSession    *rtmp.PullSession
	rtspPullSession    *rtsp.PullSession
)

type RtspPullObserver struct {
}

func (r RtspPullObserver) OnSdp(sdpCtx sdp.LogicContext) {
	rtspSdpCtx = sdpCtx
}

func (r RtspPullObserver) OnRtpPacket(pkt rtprtcp.RtpPacket) {
}

func (r RtspPullObserver) OnAvPacket(pkt base.AvPacket) {
	rtspPullAvPacketCount.Increment()
}

func Entry(t *testing.T) {
	if _, err := os.Lstat(confFile); err != nil {
		Log.Warnf("lstat %s error. err=%+v", confFile, err)
		return
	}
	if _, err := os.Lstat(rFlvFileName); err != nil {
		Log.Warnf("lstat %s error. err=%+v", rFlvFileName, err)
		return
	}

	hls.Clock = mock.NewFakeClock()
	hls.Clock.Set(time.Date(2022, 1, 16, 23, 24, 25, 0, time.Local))
	httpts.SubSessionWriteChanSize = 0

	tt = t

	var err error

	sm := logic.NewServerManager(confFile)
	go sm.RunLoop()
	time.Sleep(200 * time.Millisecond)

	config := sm.Config()

	_ = os.RemoveAll(config.HlsConfig.OutPath)

	getAllHttpApi(config.HttpApiConfig.Addr)

	pushUrl = fmt.Sprintf("rtmp://127.0.0.1%s/live/innertest", config.RtmpConfig.Addr)
	httpflvPullUrl = fmt.Sprintf("http://127.0.0.1%s/live/innertest.flv", config.HttpflvConfig.HttpListenAddr)
	httptsPullUrl = fmt.Sprintf("http://127.0.0.1%s/live/innertest.ts", config.HttpflvConfig.HttpListenAddr)
	rtmpPullUrl = fmt.Sprintf("rtmp://127.0.0.1%s/live/innertest", config.RtmpConfig.Addr)
	rtspPullUrl = fmt.Sprintf("rtsp://127.0.0.1%s/live/innertest", config.RtspConfig.Addr)
	wPlaylistM3u8FileName = fmt.Sprintf("%sinnertest/playlist.m3u8", config.HlsConfig.OutPath)
	wRecordM3u8FileName = fmt.Sprintf("%sinnertest/record.m3u8", config.HlsConfig.OutPath)
	wHlsTsFilePath = fmt.Sprintf("%sinnertest/", config.HlsConfig.OutPath)

	tags, err := httpflv.ReadAllTagsFromFlvFile(rFlvFileName)
	assert.Equal(t, nil, err)
	fileTagCount = len(tags)

	err = httpFlvWriter.Open(wFlvPullFileName)
	assert.Equal(t, nil, err)
	err = httpFlvWriter.WriteRaw(httpflv.FlvHeader)
	assert.Equal(t, nil, err)

	err = rtmpWriter.Open(wRtmpPullFileName)
	assert.Equal(t, nil, err)
	err = rtmpWriter.WriteRaw(httpflv.FlvHeader)
	assert.Equal(t, nil, err)

	go func() {
		rtmpPullSession = rtmp.NewPullSession(func(option *rtmp.PullSessionOption) {
			option.ReadAvTimeoutMs = 10000
			option.ReadBufSize = 0
		})
		err := rtmpPullSession.Pull(
			rtmpPullUrl,
			func(msg base.RtmpMsg) {
				tag := remux.RtmpMsg2FlvTag(msg)
				err := rtmpWriter.WriteTag(*tag)
				assert.Equal(tt, nil, err)
				rtmpPullTagCount.Increment()
			})
		Log.Assert(nil, err)
		err = <-rtmpPullSession.WaitChan()
		Log.Debug(err)
	}()

	go func() {
		httpflvPullSession = httpflv.NewPullSession(func(option *httpflv.PullSessionOption) {
			option.ReadTimeoutMs = 10000
		})
		err := httpflvPullSession.Pull(httpflvPullUrl, func(tag httpflv.Tag) {
			err := httpFlvWriter.WriteTag(tag)
			assert.Equal(t, nil, err)
			httpflvPullTagCount.Increment()
		})
		Log.Assert(nil, err)
		err = <-httpflvPullSession.WaitChan()
		Log.Debug(err)
	}()

	go func() {
		//nazalog.Info("CHEFGREPME >")
		b, err := httpGet(httptsPullUrl)
		assert.Equal(t, 2216332, len(b))
		assert.Equal(t, "03f8eac7d2c3d5d85056c410f5fcc756", nazamd5.Md5(b))
		Log.Infof("CHEFGREPME %+v", err)
	}()

	time.Sleep(200 * time.Millisecond)

	// TODO(chef): [test] [2021.12.25] rtsp sub测试 由于rtsp sub不支持没有pub时sub，只能sub失败后重试，所以没有验证收到的数据
	// TODO(chef): [perf] [2021.12.25] rtmp推rtsp拉的性能。开启rtsp pull后，rtmp pull的总时长增加了
	go func() {
		for {
			rtspPullAvPacketCount.Store(0)
			var rtspPullObserver RtspPullObserver
			rtspPullSession = rtsp.NewPullSession(&rtspPullObserver, func(option *rtsp.PullSessionOption) {
				option.PullTimeoutMs = 500
			})
			err := rtspPullSession.Pull(rtspPullUrl)
			Log.Debug(err)
			if rtspSdpCtx.RawSdp != nil {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	pushSession = rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
		option.WriteBufSize = 4096
		option.WriteChanSize = 1024
	})
	err = pushSession.Push(pushUrl)
	assert.Equal(t, nil, err)

	for _, tag := range tags {
		assert.Equal(t, nil, err)
		chunks := remux.FlvTag2RtmpChunks(tag)
		//Log.Debugf("rtmp push: %d", fileTagCount.Load())
		err = pushSession.Write(chunks)
		assert.Equal(t, nil, err)
	}
	err = pushSession.Flush()
	assert.Equal(t, nil, err)

	getAllHttpApi(config.HttpApiConfig.Addr)

	for {
		if httpflvPullTagCount.Load() == uint32(fileTagCount) &&
			rtmpPullTagCount.Load() == uint32(fileTagCount) {
			time.Sleep(100 * time.Millisecond)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	Log.Debug("[innertest] start dispose.")

	pushSession.Dispose()
	httpflvPullSession.Dispose()
	rtmpPullSession.Dispose()
	//rtspPullSession.Dispose()

	httpFlvWriter.Dispose()
	rtmpWriter.Dispose()

	// 由于windows没有信号，会导致编译错误，所以直接调用Dispose
	//_ = syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
	sm.Dispose()

	Log.Debugf("tag count. in=%d, out httpflv=%d, out rtmp=%d, out rtsp=%d",
		fileTagCount, httpflvPullTagCount.Load(), rtmpPullTagCount.Load(), rtspPullAvPacketCount.Load())

	compareFile()
}

func compareFile() {
	r, err := ioutil.ReadFile(rFlvFileName)
	assert.Equal(tt, nil, err)
	Log.Debugf("%s filesize:%d", rFlvFileName, len(r))

	// 检查httpflv
	w, err := ioutil.ReadFile(wFlvPullFileName)
	assert.Equal(tt, nil, err)
	Log.Debugf("%s filesize:%d", wFlvPullFileName, len(w))
	res := bytes.Compare(r, w)
	assert.Equal(tt, 0, res)
	//err = os.Remove(wFlvPullFileName)
	assert.Equal(tt, nil, err)

	// 检查rtmp
	w2, err := ioutil.ReadFile(wRtmpPullFileName)
	assert.Equal(tt, nil, err)
	Log.Debugf("%s filesize:%d", wRtmpPullFileName, len(w2))
	res = bytes.Compare(r, w2)
	assert.Equal(tt, 0, res)
	//err = os.Remove(wRtmpPullFileName)
	assert.Equal(tt, nil, err)

	// 检查hls的m3u8文件
	playListM3u8, err := ioutil.ReadFile(wPlaylistM3u8FileName)
	assert.Equal(tt, nil, err)
	assert.Equal(tt, goldenPlaylistM3u8, string(playListM3u8))
	recordM3u8, err := ioutil.ReadFile(wRecordM3u8FileName)
	assert.Equal(tt, nil, err)
	assert.Equal(tt, []byte(goldenRecordM3u8), recordM3u8)

	// 检查hls的ts文件
	var allContent []byte
	var fileNum int
	err = filebatch.Walk(
		wHlsTsFilePath,
		false,
		".ts",
		func(path string, info os.FileInfo, content []byte, err error) []byte {
			allContent = append(allContent, content...)
			fileNum++
			return nil
		})
	assert.Equal(tt, nil, err)
	allContentMd5 := nazamd5.Md5(allContent)
	assert.Equal(tt, 8, fileNum)
	assert.Equal(tt, 2219152, len(allContent))
	assert.Equal(tt, "48db6251d40c271fd11b05650f074e0f", allContentMd5)
}

func getAllHttpApi(addr string) {
	var b []byte
	var err error

	b, err = httpGet(fmt.Sprintf("http://%s/api/list", addr))
	Log.Assert(nil, err)
	Log.Debugf("%s", string(b))

	b, err = httpGet(fmt.Sprintf("http://%s/api/stat/lal_info", addr))
	Log.Assert(nil, err)
	Log.Debugf("%s", string(b))

	b, err = httpGet(fmt.Sprintf("http://%s/api/stat/group?stream_name=innertest", addr))
	Log.Assert(nil, err)
	Log.Debugf("%s", string(b))

	b, err = httpGet(fmt.Sprintf("http://%s/api/stat/all_group", addr))
	Log.Assert(nil, err)
	Log.Debugf("%s", string(b))

	var acspr base.ApiCtrlStartPullReq
	b, err = httpPost(fmt.Sprintf("http://%s/api/ctrl/start_pull", addr), &acspr)
	Log.Assert(nil, err)
	Log.Debugf("%s", string(b))

	var ackos base.ApiCtrlKickOutSession
	b, err = httpPost(fmt.Sprintf("http://%s/api/ctrl/kick_out_session", addr), &ackos)
	Log.Assert(nil, err)
	Log.Debugf("%s", string(b))
}

// ---------------------------------------------------------------------------------------------------------------------

// TODO(chef): refactor 移入naza中

func httpGet(url string) ([]byte, error) {
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func httpPost(url string, info interface{}) ([]byte, error) {
	resp, err := nazahttp.PostJson(url, info, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

// ---------------------------------------------------------------------------------------------------------------------

var goldenPlaylistM3u8 = `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-ALLOW-CACHE:NO
#EXT-X-TARGETDURATION:5
#EXT-X-MEDIA-SEQUENCE:2

#EXTINF:3.333,
innertest-1642346665000-2.ts
#EXTINF:4.000,
innertest-1642346665000-3.ts
#EXTINF:4.867,
innertest-1642346665000-4.ts
#EXTINF:3.133,
innertest-1642346665000-5.ts
#EXTINF:4.000,
innertest-1642346665000-6.ts
#EXTINF:2.621,
innertest-1642346665000-7.ts
#EXT-X-ENDLIST
`

var goldenRecordM3u8 = `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:5
#EXT-X-MEDIA-SEQUENCE:0

#EXT-X-DISCONTINUITY
#EXTINF:4.000,
innertest-1642346665000-0.ts
#EXTINF:4.000,
innertest-1642346665000-1.ts
#EXTINF:3.333,
innertest-1642346665000-2.ts
#EXTINF:4.000,
innertest-1642346665000-3.ts
#EXTINF:4.867,
innertest-1642346665000-4.ts
#EXTINF:3.133,
innertest-1642346665000-5.ts
#EXTINF:4.000,
innertest-1642346665000-6.ts
#EXTINF:2.621,
innertest-1642346665000-7.ts
#EXT-X-ENDLIST
`
