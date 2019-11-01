// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/assert"
	log "github.com/q191201771/naza/pkg/nazalog"
)

// 读取 flv 文件，使用 rtmp 协议发送至服务端，再使用 rtmp 协议从服务端拉流，转换 flv 格式存储为文件，
// 检查两份 flv 文件是否完全一致。

var (
	serverAddr = ":10001"
	pushURL    = "rtmp://127.0.0.1:10001/live/test"
	pullURL    = "rtmp://127.0.0.1:10001/live/test"
	rFLVFile   = "testdata/test.flv"
	wFLVFile   = "testdata/out.flv"
	wgNum      = 4 // FLVFileReader -> [push -> pub -> sub -> pull] -> FLVFileWriter
)

var (
	pubSessionObs MockPubSessionObserver
	pullSession   *rtmp.PullSession
	subSession    *rtmp.ServerSession
	wg            sync.WaitGroup
	w             httpflv.FLVFileWriter
	//
	rc uint32
	bc uint32
	wc uint32
)

type MockServerObserver struct {
}

func (so *MockServerObserver) NewRTMPPubSessionCB(session *rtmp.ServerSession) bool {
	log.Debug("NewRTMPPubSessionCB")
	session.SetPubSessionObserver(&pubSessionObs)
	return true
}
func (so *MockServerObserver) NewRTMPSubSessionCB(session *rtmp.ServerSession) bool {
	log.Debug("NewRTMPSubSessionCB")
	subSession = session
	return true
}
func (so *MockServerObserver) DelRTMPPubSessionCB(session *rtmp.ServerSession) {
	log.Debug("DelRTMPPubSessionCB")
	subSession.Flush()
	subSession.Dispose()
	wg.Done()
}
func (so *MockServerObserver) DelRTMPSubSessionCB(session *rtmp.ServerSession) {
	log.Debug("DelRTMPSubSessionCB")
	wg.Done()
}

type MockPubSessionObserver struct {
}

func (pso *MockPubSessionObserver) OnReadRTMPAVMsg(msg rtmp.AVMsg) {
	bc++
	// 转发
	currHeader := logic.Trans.MakeDefaultRTMPHeader(msg.Header)
	var absChunks []byte
	absChunks = rtmp.Message2Chunks(msg.Message, &currHeader)
	subSession.AsyncWrite(absChunks)
}

func TestExample(t *testing.T) {
	var err error

	var r httpflv.FLVFileReader
	err = r.Open(rFLVFile)
	if err != nil {
		return
	}

	wg.Add(wgNum)

	var so MockServerObserver
	s := rtmp.NewServer(&so, serverAddr)
	go s.RunLoop()

	// 等待 server 开始监听
	time.Sleep(100 * time.Millisecond)

	go func() {
		pullSession = rtmp.NewPullSession()
		err = pullSession.Pull(pullURL, func(msg rtmp.AVMsg) {
			tag := logic.Trans.RTMPMsg2FLVTag(msg)
			w.WriteTag(*tag)
			atomic.AddUint32(&wc, 1)
		})
		log.Error(err)
	}()

	pushSession := rtmp.NewPushSession()
	err = pushSession.Push(pushURL)
	assert.Equal(t, nil, err)

	err = w.Open(wFLVFile)
	assert.Equal(t, nil, err)
	err = w.WriteRaw(httpflv.FLVHeader)
	assert.Equal(t, nil, err)

	for {
		tag, err := r.ReadTag()
		if err == io.EOF {
			break
		}
		assert.Equal(t, nil, err)
		rc++
		//log.Debugf("send tag. %d", tag.Header.Timestamp)
		msg := logic.Trans.FLVTag2RTMPMsg(tag)
		//log.Debugf("send msg. %d %d", msg.Header.Timestamp, msg.Header.TimestampAbs)
		chunks := rtmp.Message2Chunks(msg.Message, &msg.Header)
		//log.Debugf("%s", hex.Dump(chunks[:16]))
		err = pushSession.AsyncWrite(chunks)
		assert.Equal(t, nil, err)
	}

	r.Dispose()
	wg.Done()

	err = pushSession.Flush()
	assert.Equal(t, nil, err)
	pushSession.Dispose()
	wg.Done()

	wg.Wait()

	// 等待 pull 完成
	for {
		if atomic.LoadUint32(&wc) == rc {
			break
		}
		time.Sleep(1 * time.Nanosecond)
	}
	//time.Sleep(1 * time.Second)

	pullSession.Dispose()
	w.Dispose()

	s.Dispose()
	log.Debugf("rc=%d, bc=%d, wc=%d", rc, bc, atomic.LoadUint32(&wc))
	compareFile(t)
}

func compareFile(t *testing.T) {
	r, err := ioutil.ReadFile(rFLVFile)
	assert.Equal(t, nil, err)
	w, err := ioutil.ReadFile(wFLVFile)
	assert.Equal(t, nil, err)
	res := bytes.Compare(r, w)
	assert.Equal(t, 0, res)
	//err = os.Remove(wFLVFile)
	assert.Equal(t, nil, err)
}
