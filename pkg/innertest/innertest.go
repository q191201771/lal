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
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

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
// 读取flv文件，使用rtmp协议推送至服务端
// 分别用rtmp协议以及httpflv协议从服务端拉流，再将拉取的流保存为flv文件
// 对比三份flv文件，看是否完全一致
// 并检查hls生成的m3u8和ts文件，是否和之前的完全一致

// TODO chef:
// - 加上relay push
// - 加上relay pull
// - 加上rtspserver的测试

var (
	tt *testing.T

	confFile = "testdata/lalserver.conf.json"

	rFlvFileName      = "testdata/test.flv"
	wFlvPullFileName  = "testdata/flvpull.flv"
	wRtmpPullFileName = "testdata/rtmppull.flv"

	pushUrl        string
	httpflvPullUrl string
	rtmpPullUrl    string

	fileReader    httpflv.FlvFileReader
	httpFlvWriter httpflv.FlvFileWriter
	rtmpWriter    httpflv.FlvFileWriter

	pushSession        *rtmp.PushSession
	httpflvPullSession *httpflv.PullSession
	rtmpPullSession    *rtmp.PullSession

	fileTagCount        nazaatomic.Uint32
	httpflvPullTagCount nazaatomic.Uint32
	rtmpPullTagCount    nazaatomic.Uint32
)

func InnerTestEntry(t *testing.T) {
	tt = t

	var err error

	logic.Init(confFile)
	go logic.RunLoop()
	time.Sleep(200 * time.Millisecond)

	config := logic.GetConfig()

	_ = os.RemoveAll(config.HlsConfig.OutPath)

	pushUrl = fmt.Sprintf("rtmp://127.0.0.1%s/live/innertest", config.RtmpConfig.Addr)
	httpflvPullUrl = fmt.Sprintf("http://127.0.0.1%s/live/innertest.flv", config.HttpflvConfig.HttpListenAddr)
	rtmpPullUrl = fmt.Sprintf("rtmp://127.0.0.1%s/live/innertest", config.RtmpConfig.Addr)

	err = fileReader.Open(rFlvFileName)
	assert.Equal(t, nil, err)

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
			option.ReadAvTimeoutMs = 500
		})
		err := rtmpPullSession.Pull(
			rtmpPullUrl,
			func(msg base.RtmpMsg) {
				tag := remux.RtmpMsg2FlvTag(msg)
				err := rtmpWriter.WriteTag(*tag)
				assert.Equal(tt, nil, err)
				rtmpPullTagCount.Increment()
			})
		if err != nil {
			nazalog.Error(err)
		}
		err = <-rtmpPullSession.WaitChan()
		nazalog.Debug(err)
	}()

	go func() {
		httpflvPullSession = httpflv.NewPullSession(func(option *httpflv.PullSessionOption) {
			option.ReadTimeoutMs = 500
		})
		err := httpflvPullSession.Pull(httpflvPullUrl, func(tag httpflv.Tag) {
			err := httpFlvWriter.WriteTag(tag)
			assert.Equal(t, nil, err)
			httpflvPullTagCount.Increment()
		})
		nazalog.Error(err)
	}()

	time.Sleep(200 * time.Millisecond)

	pushSession = rtmp.NewPushSession()
	err = pushSession.Push(pushUrl)
	assert.Equal(t, nil, err)

	for {
		tag, err := fileReader.ReadTag()
		if err == io.EOF {
			break
		}
		assert.Equal(t, nil, err)
		fileTagCount.Increment()
		msg := remux.FlvTag2RtmpMsg(tag)
		chunks := rtmp.Message2Chunks(msg.Payload, &msg.Header)
		err = pushSession.Write(chunks)
		assert.Equal(t, nil, err)
	}
	err = pushSession.Flush()
	assert.Equal(t, nil, err)

	time.Sleep(1 * time.Second)

	fileReader.Dispose()
	pushSession.Dispose()
	httpflvPullSession.Dispose()
	rtmpPullSession.Dispose()
	httpFlvWriter.Dispose()
	rtmpWriter.Dispose()
	// 由于windows没有信号，会导致编译错误，所以直接调用Dispose
	//_ = syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)
	logic.Dispose()

	nazalog.Debugf("count. %d %d %d", fileTagCount.Load(), httpflvPullTagCount.Load(), rtmpPullTagCount.Load())
	compareFile()

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
	err = os.Remove(wFlvPullFileName)
	assert.Equal(tt, nil, err)

	w2, err := ioutil.ReadFile(wRtmpPullFileName)
	assert.Equal(tt, nil, err)
	nazalog.Debugf("%s filesize:%d", wRtmpPullFileName, len(w2))
	res = bytes.Compare(r, w2)
	assert.Equal(tt, 0, res)
	err = os.Remove(wRtmpPullFileName)
	assert.Equal(tt, nil, err)
}
