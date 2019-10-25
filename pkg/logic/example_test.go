// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/q191201771/naza/pkg/nazaatomic"
	"github.com/q191201771/naza/pkg/nazalog"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"

	"github.com/q191201771/naza/pkg/assert"
)

var (
	tt *testing.T

	confFile = "testdata/lals.default.conf.json"

	rFLVFileName      = "testdata/test.flv"
	wFLVPullFileName  = "testdata/flvpull.flv"
	wRTMPPullFileName = "testdata/rtmppull.flv"

	pushURL        string
	httpflvPullURL string
	rtmpPullURL    string

	fileReader    httpflv.FLVFileReader
	HTTPFLVWriter httpflv.FLVFileWriter
	RTMPWriter    httpflv.FLVFileWriter

	pushSession        *rtmp.PushSession
	httpflvPullSession *httpflv.PullSession
	rtmpPullSession    *rtmp.PullSession

	fileTagCount        nazaatomic.Uint32
	httpflvPullTagCount nazaatomic.Uint32
	rtmpPullTagCount    nazaatomic.Uint32
)

type MockRTMPPullSessionObserver struct {
}

// TODO chef: httpflv 和 rtmp 两种协议的 pull 接口形式不统一
func (mrpso *MockRTMPPullSessionObserver) ReadRTMPAVMsgCB(header rtmp.Header, timestampAbs uint32, message []byte) {
	tag := Trans.RTMPMsg2FLVTag(header, timestampAbs, message)
	err := RTMPWriter.WriteTag(*tag)
	assert.Equal(tt, nil, err)
	rtmpPullTagCount.Increment()
}

func TestExample(t *testing.T) {
	tt = t

	var err error

	err = fileReader.Open(rFLVFileName)
	assert.Equal(t, nil, err)

	config, err := LoadConf(confFile)
	assert.IsNotNil(t, config)
	assert.Equal(t, nil, err)

	pushURL = fmt.Sprintf("rtmp://127.0.0.1%s/live/11111", config.RTMP.Addr)
	httpflvPullURL = fmt.Sprintf("http://127.0.0.1%s/live/11111.flv", config.HTTPFLV.SubListenAddr)
	rtmpPullURL = fmt.Sprintf("rtmp://127.0.0.1%s/live/11111", config.RTMP.Addr)

	sm := NewServerManager(config)
	go sm.RunLoop()

	time.Sleep(200 * time.Millisecond)

	err = HTTPFLVWriter.Open(wFLVPullFileName)
	assert.Equal(t, nil, err)
	err = HTTPFLVWriter.WriteRaw(httpflv.FLVHeader)
	assert.Equal(t, nil, err)

	err = RTMPWriter.Open(wRTMPPullFileName)
	assert.Equal(t, nil, err)
	err = RTMPWriter.WriteRaw(httpflv.FLVHeader)
	assert.Equal(t, nil, err)

	go func() {
		var mrpso MockRTMPPullSessionObserver
		rtmpPullSession = rtmp.NewPullSession(&mrpso, rtmp.PullSessionTimeout{
			ReadAVTimeoutMS: 500,
		})
		err := rtmpPullSession.Pull(rtmpPullURL)
		assert.Equal(t, nil, err)
	}()

	go func() {
		httpflvPullSession = httpflv.NewPullSession(httpflv.PullSessionConfig{
			ReadTimeoutMS: 500,
		})
		err := httpflvPullSession.Pull(httpflvPullURL, func(tag *httpflv.Tag) {
			err := HTTPFLVWriter.WriteTag(*tag)
			assert.Equal(t, nil, err)
			httpflvPullTagCount.Increment()
		})
		nazalog.Error(err)
	}()

	time.Sleep(200 * time.Millisecond)

	pushSession = rtmp.NewPushSession(rtmp.PushSessionTimeout{})
	err = pushSession.Push(pushURL)
	assert.Equal(t, nil, err)

	_, err = fileReader.ReadFLVHeader()
	assert.Equal(t, nil, err)
	for {
		tag, err := fileReader.ReadTag()
		if err == io.EOF {
			break
		}
		assert.Equal(t, nil, err)
		fileTagCount.Increment()
		h, _, m := Trans.FLVTag2RTMPMsg(*tag)
		chunks := rtmp.Message2Chunks(m, &h)
		err = pushSession.AsyncWrite(chunks)
		assert.Equal(t, nil, err)
	}
	err = pushSession.Flush()
	assert.Equal(t, nil, err)

	time.Sleep(1 * time.Second)

	fileReader.Dispose()
	pushSession.Dispose()
	httpflvPullSession.Dispose(nil)
	rtmpPullSession.Dispose()
	HTTPFLVWriter.Dispose()
	RTMPWriter.Dispose()
	sm.Dispose()

	nazalog.Debugf("count. %d %d %d", fileTagCount.Load(), httpflvPullTagCount.Load(), rtmpPullTagCount.Load())
	compareFile()
}

func compareFile() {
	r, err := ioutil.ReadFile(rFLVFileName)
	assert.Equal(tt, nil, err)
	w, err := ioutil.ReadFile(wFLVPullFileName)
	assert.Equal(tt, nil, err)
	res := bytes.Compare(r, w)
	assert.Equal(tt, 0, res)
	err = os.Remove(wFLVPullFileName)
	assert.Equal(tt, nil, err)
	w2, err := ioutil.ReadFile(wRTMPPullFileName)
	assert.Equal(tt, nil, err)
	res = bytes.Compare(r, w2)
	assert.Equal(tt, 0, res)
	err = os.Remove(wRTMPPullFileName)
	assert.Equal(tt, nil, err)
}
