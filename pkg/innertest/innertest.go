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
	"os"
	"testing"
	"time"

	"github.com/q191201771/lal/pkg/rtprtcp"
	"github.com/q191201771/lal/pkg/rtsp"
	"github.com/q191201771/lal/pkg/sdp"

	"github.com/q191201771/lal/pkg/remux"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/filebatch"
	"github.com/q191201771/naza/pkg/nazamd5"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/assert"
	"github.com/q191201771/naza/pkg/nazaatomic"
	"github.com/q191201771/naza/pkg/nazalog"
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

	confFile          = "../../testdata/lalserver.conf.json"
	rFlvFileName      = "../../testdata/test.flv"
	wFlvPullFileName  = "../../testdata/flvpull.flv"
	wRtmpPullFileName = "../../testdata/rtmppull.flv"
	wRtspPullFileName = "../../testdata/rtsppull.flv"

	pushUrl        string
	httpflvPullUrl string
	rtmpPullUrl    string
	rtspPullUrl    string

	httpFlvWriter httpflv.FlvFileWriter
	rtmpWriter    httpflv.FlvFileWriter

	pushSession        *rtmp.PushSession
	httpflvPullSession *httpflv.PullSession
	rtmpPullSession    *rtmp.PullSession
	rtspPullSession    *rtsp.PullSession

	fileTagCount          int
	httpflvPullTagCount   nazaatomic.Uint32
	rtmpPullTagCount      nazaatomic.Uint32
	rtspSdpCtx            sdp.LogicContext
	rtspPullAvPacketCount nazaatomic.Uint32
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
		nazalog.Warnf("lstat %s error. err=%+v", confFile, err)
		return
	}
	if _, err := os.Lstat(rFlvFileName); err != nil {
		nazalog.Warnf("lstat %s error. err=%+v", rFlvFileName, err)
		return
	}

	tt = t

	var err error

	sm := logic.NewServerManager(confFile)
	go sm.RunLoop()
	time.Sleep(200 * time.Millisecond)

	config := sm.Config()

	_ = os.RemoveAll(config.HlsConfig.OutPath)

	pushUrl = fmt.Sprintf("rtmp://127.0.0.1%s/live/innertest", config.RtmpConfig.Addr)
	httpflvPullUrl = fmt.Sprintf("http://127.0.0.1%s/live/innertest.flv", config.HttpflvConfig.HttpListenAddr)
	rtmpPullUrl = fmt.Sprintf("rtmp://127.0.0.1%s/live/innertest", config.RtmpConfig.Addr)
	rtspPullUrl = fmt.Sprintf("rtsp://127.0.0.1%s/live/innertest", config.RtspConfig.Addr)

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
		nazalog.Assert(nil, err)
		err = <-rtmpPullSession.WaitChan()
		nazalog.Debug(err)
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
		nazalog.Assert(nil, err)
		err = <-httpflvPullSession.WaitChan()
		nazalog.Debug(err)
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
			nazalog.Debug(err)
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

	nazalog.Debugf("CHEFERASEME start push %+v", time.Now())
	for _, tag := range tags {
		assert.Equal(t, nil, err)
		chunks := remux.FlvTag2RtmpChunks(tag)
		//nazalog.Debugf("rtmp push: %d", fileTagCount.Load())
		err = pushSession.Write(chunks)
		assert.Equal(t, nil, err)
	}
	err = pushSession.Flush()
	assert.Equal(t, nil, err)

	for {
		if httpflvPullTagCount.Load() == uint32(fileTagCount) &&
			rtmpPullTagCount.Load() == uint32(fileTagCount) {
			time.Sleep(100 * time.Millisecond)
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	nazalog.Debug("[innertest] start dispose.")

	pushSession.Dispose()
	httpflvPullSession.Dispose()
	rtmpPullSession.Dispose()
	//rtspPullSession.Dispose()

	httpFlvWriter.Dispose()
	rtmpWriter.Dispose()

	// 由于windows没有信号，会导致编译错误，所以直接调用Dispose
	//_ = syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
	sm.Dispose()

	nazalog.Debugf("tag count. in=%d, out httpflv=%d, out rtmp=%d, out rtsp=%d",
		fileTagCount, httpflvPullTagCount.Load(), rtmpPullTagCount.Load(), rtspPullAvPacketCount.Load())

	compareFile()

	// 检查hls的ts文件
	var allContent []byte
	var fileNum int
	err = filebatch.Walk(
		fmt.Sprintf("%sinnertest", config.HlsConfig.OutPath),
		false,
		".ts",
		func(path string, info os.FileInfo, content []byte, err error) []byte {
			allContent = append(allContent, content...)
			fileNum++
			return nil
		})
	assert.Equal(t, nil, err)
	allContentMd5 := nazamd5.Md5(allContent)
	assert.Equal(t, 8, fileNum)
	assert.Equal(t, 2219152, len(allContent))
	assert.Equal(t, "48db6251d40c271fd11b05650f074e0f", allContentMd5)
}

func compareFile() {
	r, err := ioutil.ReadFile(rFlvFileName)
	assert.Equal(tt, nil, err)
	nazalog.Debugf("%s filesize:%d", rFlvFileName, len(r))

	w, err := ioutil.ReadFile(wFlvPullFileName)
	assert.Equal(tt, nil, err)
	nazalog.Debugf("%s filesize:%d", wFlvPullFileName, len(w))
	res := bytes.Compare(r, w)
	assert.Equal(tt, 0, res)
	//err = os.Remove(wFlvPullFileName)
	assert.Equal(tt, nil, err)

	w2, err := ioutil.ReadFile(wRtmpPullFileName)
	assert.Equal(tt, nil, err)
	nazalog.Debugf("%s filesize:%d", wRtmpPullFileName, len(w2))
	res = bytes.Compare(r, w2)
	assert.Equal(tt, 0, res)
	//err = os.Remove(wRtmpPullFileName)
	assert.Equal(tt, nil, err)
}
