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
	"syscall"
	"testing"
	"time"

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

// TODO chef:
// - 加上hls的检查
// - 加上relay push
// - 加上relay pull

var (
	tt *testing.T

	confFile = "testdata/lalserver.conf.json"

	rFLVFileName      = "testdata/test.flv"
	wFLVPullFileName  = "testdata/flvpull.flv"
	wRTMPPullFileName = "testdata/rtmppull.flv"

	pushURL        string
	httpflvPullURL string
	rtmpPullURL    string

	fileReader    httpflv.FLVFileReader
	httpFLVWriter httpflv.FLVFileWriter
	rtmpWriter    httpflv.FLVFileWriter

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

	go logic.Entry(confFile)
	time.Sleep(200 * time.Millisecond)

	config, err := logic.LoadConf(confFile)
	assert.Equal(t, nil, err)

	pushURL = fmt.Sprintf("rtmp://127.0.0.1%s/live/11111", config.RTMPConfig.Addr)
	httpflvPullURL = fmt.Sprintf("http://127.0.0.1%s/live/11111.flv", config.HTTPFLVConfig.SubListenAddr)
	rtmpPullURL = fmt.Sprintf("rtmp://127.0.0.1%s/live/11111", config.RTMPConfig.Addr)

	err = fileReader.Open(rFLVFileName)
	assert.Equal(t, nil, err)

	err = httpFLVWriter.Open(wFLVPullFileName)
	assert.Equal(t, nil, err)
	err = httpFLVWriter.WriteRaw(httpflv.FLVHeader)
	assert.Equal(t, nil, err)

	err = rtmpWriter.Open(wRTMPPullFileName)
	assert.Equal(t, nil, err)
	err = rtmpWriter.WriteRaw(httpflv.FLVHeader)
	assert.Equal(t, nil, err)

	go func() {
		rtmpPullSession = rtmp.NewPullSession(func(option *rtmp.PullSessionOption) {
			option.ReadAVTimeoutMS = 500
		})
		err := rtmpPullSession.Pull(
			rtmpPullURL,
			func(msg rtmp.AVMsg) {
				tag := logic.Trans.RTMPMsg2FLVTag(msg)
				err := rtmpWriter.WriteTag(*tag)
				assert.Equal(tt, nil, err)
				rtmpPullTagCount.Increment()
			})
		nazalog.Error(err)
		err = <-rtmpPullSession.Done()
		nazalog.Debug(err)
	}()

	go func() {
		httpflvPullSession = httpflv.NewPullSession(func(option *httpflv.PullSessionOption) {
			option.ReadTimeoutMS = 500
		})
		err := httpflvPullSession.Pull(httpflvPullURL, func(tag httpflv.Tag) {
			err := httpFLVWriter.WriteTag(tag)
			assert.Equal(t, nil, err)
			httpflvPullTagCount.Increment()
		})
		nazalog.Error(err)
	}()

	time.Sleep(200 * time.Millisecond)

	pushSession = rtmp.NewPushSession()
	err = pushSession.Push(pushURL)
	assert.Equal(t, nil, err)

	for {
		tag, err := fileReader.ReadTag()
		if err == io.EOF {
			break
		}
		assert.Equal(t, nil, err)
		fileTagCount.Increment()
		msg := logic.Trans.FLVTag2RTMPMsg(tag)
		chunks := rtmp.Message2Chunks(msg.Payload, &msg.Header)
		err = pushSession.AsyncWrite(chunks)
		assert.Equal(t, nil, err)
	}
	err = pushSession.Flush()
	assert.Equal(t, nil, err)

	time.Sleep(1 * time.Second)

	fileReader.Dispose()
	pushSession.Dispose()
	httpflvPullSession.Dispose()
	rtmpPullSession.Dispose()
	httpFLVWriter.Dispose()
	rtmpWriter.Dispose()
	_ = syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)

	nazalog.Debugf("count. %d %d %d", fileTagCount.Load(), httpflvPullTagCount.Load(), rtmpPullTagCount.Load())
	compareFile()
}

func compareFile() {
	r, err := ioutil.ReadFile(rFLVFileName)
	assert.Equal(tt, nil, err)
	nazalog.Debugf("%s filesize:%d", rFLVFileName, len(r))

	w, err := ioutil.ReadFile(wFLVPullFileName)
	assert.Equal(tt, nil, err)
	nazalog.Debugf("%s filesize:%d", wFLVPullFileName, len(w))
	res := bytes.Compare(r, w)
	assert.Equal(tt, 0, res)
	err = os.Remove(wFLVPullFileName)
	assert.Equal(tt, nil, err)

	w2, err := ioutil.ReadFile(wRTMPPullFileName)
	assert.Equal(tt, nil, err)
	nazalog.Debugf("%s filesize:%d", wRTMPPullFileName, len(w2))
	res = bytes.Compare(r, w2)
	assert.Equal(tt, 0, res)
	err = os.Remove(wRTMPPullFileName)
	assert.Equal(tt, nil, err)
}
