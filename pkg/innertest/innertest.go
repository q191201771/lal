// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package innertest

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/q191201771/naza/pkg/nazabytes"
	"github.com/q191201771/naza/pkg/nazalog"

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
	t *testing.T

	mode         int // 0 正常 1 输入只有音频 2 输入只有视频
	confFilename = "../../testdata/lalserver.conf.json"
	rFlvFileName = "../../testdata/test.flv"

	pushUrl        string
	httpflvPullUrl string
	httptsPullUrl  string
	rtmpPullUrl    string
	rtspPullUrl    string

	wRtmpPullFileName     string
	wFlvPullFileName      string
	wPlaylistM3u8FileName string
	wRecordM3u8FileName   string
	wHlsTsFilePath        string
	wTsPullFileName       string

	fileTagCount          int
	httpflvPullTagCount   nazaatomic.Uint32
	rtmpPullTagCount      nazaatomic.Uint32
	httptsSize            nazaatomic.Uint32
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

func Entry(tt *testing.T) {
	t = tt

	mode = 0
	entry()

	mode = 1
	entry()

	mode = 2
	entry()
}

func entry() {
	Log.Debugf("> innertest")

	if _, err := os.Lstat(confFilename); err != nil {
		Log.Warnf("lstat %s error. err=%+v", confFilename, err)
		return
	}
	if _, err := os.Lstat(rFlvFileName); err != nil {
		Log.Warnf("lstat %s error. err=%+v", rFlvFileName, err)
		return
	}

	httpflvPullTagCount.Store(0)
	rtmpPullTagCount.Store(0)
	httptsSize.Store(0)
	hls.Clock = mock.NewFakeClock()
	hls.Clock.Set(time.Date(2022, 1, 16, 23, 24, 25, 0, time.UTC))
	httpts.SubSessionWriteChanSize = 0

	var err error

	sm := logic.NewServerManager(func(option *logic.Option) {
		option.ConfFilename = confFilename
	})
	config := sm.Config()

	//Log.Init(func(option *nazalog.Option) {
	//	option.Level = nazalog.LevelLogNothing
	//})
	_ = os.RemoveAll(config.HlsConfig.OutPath)

	go sm.RunLoop()
	time.Sleep(100 * time.Millisecond)

	getAllHttpApi(config.HttpApiConfig.Addr)

	pushUrl = fmt.Sprintf("rtmp://127.0.0.1%s/live/innertest", config.RtmpConfig.Addr)
	httpflvPullUrl = fmt.Sprintf("http://127.0.0.1%s/live/innertest.flv", config.HttpflvConfig.HttpListenAddr)
	httptsPullUrl = fmt.Sprintf("http://127.0.0.1%s/live/innertest.ts", config.HttpflvConfig.HttpListenAddr)
	rtmpPullUrl = fmt.Sprintf("rtmp://127.0.0.1%s/live/innertest", config.RtmpConfig.Addr)
	rtspPullUrl = fmt.Sprintf("rtsp://127.0.0.1%s/live/innertest", config.RtspConfig.Addr)

	wRtmpPullFileName = "../../testdata/rtmppull.flv"
	wFlvPullFileName = "../../testdata/flvpull.flv"
	wTsPullFileName = fmt.Sprintf("../../testdata/tspull_%d.ts", mode)
	wPlaylistM3u8FileName = fmt.Sprintf("%sinnertest/playlist.m3u8", config.HlsConfig.OutPath)
	wRecordM3u8FileName = fmt.Sprintf("%sinnertest/record.m3u8", config.HlsConfig.OutPath)
	wHlsTsFilePath = fmt.Sprintf("%sinnertest/", config.HlsConfig.OutPath)

	var tags []httpflv.Tag
	originTags, err := httpflv.ReadAllTagsFromFlvFile(rFlvFileName)
	assert.Equal(t, nil, err)
	if mode == 0 {
		tags = originTags
	} else if mode == 1 {
		for _, tag := range originTags {
			if tag.Header.Type == base.RtmpTypeIdMetadata || tag.Header.Type == base.RtmpTypeIdAudio {
				tags = append(tags, tag)
			}
		}
	} else if mode == 2 {
		for _, tag := range originTags {
			if tag.Header.Type == base.RtmpTypeIdMetadata || tag.Header.Type == base.RtmpTypeIdVideo {
				tags = append(tags, tag)
			}
		}
	}
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
		}).WithOnReadRtmpAvMsg(func(msg base.RtmpMsg) {
			tag := remux.RtmpMsg2FlvTag(msg)
			err := rtmpWriter.WriteTag(*tag)
			assert.Equal(t, nil, err)
			rtmpPullTagCount.Increment()
		})
		err := rtmpPullSession.Pull(rtmpPullUrl)
		Log.Assert(nil, err)
		err = <-rtmpPullSession.WaitChan()
		Log.Debug(err)
	}()

	go func() {
		var flvErr error
		httpflvPullSession = httpflv.NewPullSession(func(option *httpflv.PullSessionOption) {
			option.ReadTimeoutMs = 10000
		})
		err := httpflvPullSession.Pull(httpflvPullUrl, func(tag httpflv.Tag) {
			err := httpFlvWriter.WriteTag(tag)
			assert.Equal(t, nil, err)
			httpflvPullTagCount.Increment()
		})
		Log.Assert(nil, err)
		flvErr = <-httpflvPullSession.WaitChan()
		Log.Debug(flvErr)
	}()

	go func() {
		b, _ := getHttpts()
		_ = ioutil.WriteFile(wTsPullFileName, b, 0666)
		assert.Equal(t, goldenHttptsLenList[mode], len(b))
		assert.Equal(t, goldenHttptsMd5List[mode], nazamd5.Md5(b))
	}()
	time.Sleep(100 * time.Millisecond)

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

	time.Sleep(100 * time.Millisecond)

	pushSession = rtmp.NewPushSession(func(option *rtmp.PushSessionOption) {
		option.WriteBufSize = 4096
		//option.WriteChanSize = 1024
	})
	err = pushSession.Push(pushUrl)
	assert.Equal(t, nil, err)

	for _, tag := range tags {
		assert.Equal(t, nil, err)
		chunks := remux.FlvTag2RtmpChunks(tag)
		//Log.Debugf("rtmp push: %d", fileTagCount.Load())
		err := pushSession.Write(chunks)
		assert.Equal(t, nil, err)
	}
	err = pushSession.Flush()
	assert.Equal(t, nil, err)

	getAllHttpApi(config.HttpApiConfig.Addr)

	// 注意，先释放push，触发pub释放，从而刷新hls的结束时切片逻辑
	pushSession.Dispose()

	for {
		if httpflvPullTagCount.Load() == uint32(fileTagCount) &&
			rtmpPullTagCount.Load() == uint32(fileTagCount) &&
			httptsSize.Load() == uint32(goldenHttptsLenList[mode]) {
			break
		}
		nazalog.Debugf("%d(%d, %d) %d(%d)",
			fileTagCount, httpflvPullTagCount.Load(), rtmpPullTagCount.Load(), goldenHttptsLenList[mode], httptsSize.Load())
		time.Sleep(100 * time.Millisecond)
	}

	Log.Debug("[innertest] start dispose.")

	httpflvPullSession.Dispose()
	rtmpPullSession.Dispose()
	rtspPullSession.Dispose()

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
	assert.Equal(t, nil, err)
	Log.Debugf("%s filesize:%d", rFlvFileName, len(r))

	// 检查httpflv
	w, err := ioutil.ReadFile(wFlvPullFileName)
	assert.Equal(t, nil, err)
	assert.Equal(t, goldenHttpflvLenList[mode], len(w))
	assert.Equal(t, goldenHttpflvMd5List[mode], nazamd5.Md5(w))

	// 检查rtmp
	w, err = ioutil.ReadFile(wRtmpPullFileName)
	assert.Equal(t, nil, err)
	assert.Equal(t, goldenRtmpLenList[mode], len(w))
	assert.Equal(t, goldenRtmpMd5List[mode], nazamd5.Md5(w))

	// 检查hls的m3u8文件
	playListM3u8, err := ioutil.ReadFile(wPlaylistM3u8FileName)
	assert.Equal(t, nil, err)
	assert.Equal(t, goldenPlaylistM3u8List[mode], string(playListM3u8))
	recordM3u8, err := ioutil.ReadFile(wRecordM3u8FileName)
	assert.Equal(t, nil, err)
	assert.Equal(t, goldenRecordM3u8List[mode], string(recordM3u8))

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
	assert.Equal(t, nil, err)
	allContentMd5 := nazamd5.Md5(allContent)
	assert.Equal(t, goldenHlsTsNumList[mode], fileNum)
	assert.Equal(t, goldenHlsTsLenList[mode], len(allContent))
	assert.Equal(t, goldenHlsTsMd5List[mode], allContentMd5)
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

	var acspr base.ApiCtrlStartRelayPullReq
	b, err = httpPost(fmt.Sprintf("http://%s/api/ctrl/start_relay_pull", addr), &acspr)
	Log.Assert(nil, err)
	Log.Debugf("%s", string(b))

	var ackos base.ApiCtrlKickOutSession
	b, err = httpPost(fmt.Sprintf("http://%s/api/ctrl/kick_out_session", addr), &ackos)
	Log.Assert(nil, err)
	Log.Debugf("%s", string(b))
}

func getHttpts() ([]byte, error) {
	resp, err := http.DefaultClient.Get(httptsPullUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var buf nazabytes.Buffer
	buf.ReserveBytes(goldenHttptsLenList[mode])
	for {
		n, err := resp.Body.Read(buf.WritableBytes())
		if n > 0 {
			buf.Flush(n)
			httptsSize.Add(uint32(n))
		}
		if err != nil {
			return buf.Bytes(), err
		}
		if buf.Len() == goldenHttptsLenList[mode] {
			return buf.Bytes(), nil
		}
	}
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

var (
	goldenRtmpLenList = []int{2120047, 504722, 1615715}
	goldenRtmpMd5List = []string{
		"7d68f0e2ab85c1992f70740479c8d3db",
		"b889f690e07399c8c8353a3b1dba7efb",
		"b5a9759455039761b6d4dd3ed8e97634",
	}

	goldenHttpflvLenList = []int{2120047, 504722, 1615715}
	goldenHttpflvMd5List = []string{
		"7d68f0e2ab85c1992f70740479c8d3db",
		"b889f690e07399c8c8353a3b1dba7efb",
		"b5a9759455039761b6d4dd3ed8e97634",
	}

	goldenHlsTsNumList = []int{8, 10, 8}
	goldenHlsTsLenList = []int{2219152, 525648, 1696512}
	goldenHlsTsMd5List = []string{
		"48db6251d40c271fd11b05650f074e0f",
		"2eb19ad498688dadf950b3e749985922",
		"2d1e5c1a3ca385e0b55b2cff2f52b710",
	}

	goldenHttptsLenList = []int{2216332, 522264, 1693880}
	goldenHttptsMd5List = []string{
		"03f8eac7d2c3d5d85056c410f5fcc756",
		"0d102b6fb7fc3134e56a07f00292e888",
		"651a2b5c93370738d81995f8fd49af4d",
	}
)

var goldenPlaylistM3u8List = []string{
	`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-ALLOW-CACHE:NO
#EXT-X-TARGETDURATION:5
#EXT-X-MEDIA-SEQUENCE:2

#EXTINF:3.333,
innertest-1642375465000-2.ts
#EXTINF:4.000,
innertest-1642375465000-3.ts
#EXTINF:4.867,
innertest-1642375465000-4.ts
#EXTINF:3.133,
innertest-1642375465000-5.ts
#EXTINF:4.000,
innertest-1642375465000-6.ts
#EXTINF:2.621,
innertest-1642375465000-7.ts
#EXT-X-ENDLIST
`,
	`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-ALLOW-CACHE:NO
#EXT-X-TARGETDURATION:3
#EXT-X-MEDIA-SEQUENCE:4

#EXTINF:3.088,
innertest-1642375465000-4.ts
#EXTINF:3.088,
innertest-1642375465000-5.ts
#EXTINF:3.089,
innertest-1642375465000-6.ts
#EXTINF:3.088,
innertest-1642375465000-7.ts
#EXTINF:3.088,
innertest-1642375465000-8.ts
#EXTINF:2.113,
innertest-1642375465000-9.ts
#EXT-X-ENDLIST
`,
	`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-ALLOW-CACHE:NO
#EXT-X-TARGETDURATION:5
#EXT-X-MEDIA-SEQUENCE:2

#EXTINF:3.333,
innertest-1642375465000-2.ts
#EXTINF:4.000,
innertest-1642375465000-3.ts
#EXTINF:4.867,
innertest-1642375465000-4.ts
#EXTINF:3.133,
innertest-1642375465000-5.ts
#EXTINF:4.000,
innertest-1642375465000-6.ts
#EXTINF:2.600,
innertest-1642375465000-7.ts
#EXT-X-ENDLIST
`,
}

var goldenRecordM3u8List = []string{
	`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:5
#EXT-X-MEDIA-SEQUENCE:0

#EXT-X-DISCONTINUITY
#EXTINF:4.000,
innertest-1642375465000-0.ts
#EXTINF:4.000,
innertest-1642375465000-1.ts
#EXTINF:3.333,
innertest-1642375465000-2.ts
#EXTINF:4.000,
innertest-1642375465000-3.ts
#EXTINF:4.867,
innertest-1642375465000-4.ts
#EXTINF:3.133,
innertest-1642375465000-5.ts
#EXTINF:4.000,
innertest-1642375465000-6.ts
#EXTINF:2.621,
innertest-1642375465000-7.ts
#EXT-X-ENDLIST
`,
	`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:3
#EXT-X-MEDIA-SEQUENCE:0

#EXT-X-DISCONTINUITY
#EXTINF:3.088,
innertest-1642375465000-0.ts
#EXTINF:3.088,
innertest-1642375465000-1.ts
#EXTINF:3.089,
innertest-1642375465000-2.ts
#EXTINF:3.088,
innertest-1642375465000-3.ts
#EXTINF:3.088,
innertest-1642375465000-4.ts
#EXTINF:3.088,
innertest-1642375465000-5.ts
#EXTINF:3.089,
innertest-1642375465000-6.ts
#EXTINF:3.088,
innertest-1642375465000-7.ts
#EXTINF:3.088,
innertest-1642375465000-8.ts
#EXTINF:2.113,
innertest-1642375465000-9.ts
#EXT-X-ENDLIST
`,
	`#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:5
#EXT-X-MEDIA-SEQUENCE:0

#EXT-X-DISCONTINUITY
#EXTINF:4.000,
innertest-1642375465000-0.ts
#EXTINF:4.000,
innertest-1642375465000-1.ts
#EXTINF:3.333,
innertest-1642375465000-2.ts
#EXTINF:4.000,
innertest-1642375465000-3.ts
#EXTINF:4.867,
innertest-1642375465000-4.ts
#EXTINF:3.133,
innertest-1642375465000-5.ts
#EXTINF:4.000,
innertest-1642375465000-6.ts
#EXTINF:2.600,
innertest-1642375465000-7.ts
#EXT-X-ENDLIST
`,
}
