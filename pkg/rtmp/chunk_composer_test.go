// Copyright 2023, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

import (
	"bytes"
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/assert"
	"testing"
	"time"
)

func TestChunkComposer(t *testing.T) {

	//case: 音视频混合发送的时候测试case

	//video payload 50
	//chunk size = 20
	videoMsg := base.RtmpMsg{
		Header: base.RtmpHeader{
			Csid:         6,
			MsgLen:       50,
			MsgTypeId:    base.RtmpTypeIdVideo,
			MsgStreamId:  Msid1,
			TimestampAbs: 1000,
		},
		Payload: make([]byte, 50),
	}
	//fmt = 0
	videoChunk1 := []byte{6, 0, 3, 232, 0, 0, 50, 9, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	//fmt = 3
	videoChunk2 := []byte{198, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	//fmt = 3
	videoChunk3 := []byte{198, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	//audio payload 15
	//chunk size = 20
	//ah :=
	audioMsg := base.RtmpMsg{
		Header: base.RtmpHeader{
			Csid:         5,
			MsgLen:       15,
			MsgTypeId:    base.RtmpTypeIdAudio,
			MsgStreamId:  Msid1,
			TimestampAbs: 1000,
		},
		Payload: make([]byte, 15),
	}
	//fmt = 0
	audioChunk1 := []byte{5, 0, 3, 232, 0, 0, 15, 8, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	rb := &bytes.Buffer{}
	rb.Write(videoChunk1)
	rb.Write(videoChunk2)
	rb.Write(audioChunk1)
	rb.Write(videoChunk3)

	cc := NewChunkComposer()
	cc.peerChunkSize = 20

	done := make(chan struct{}, 1)
	c := 2

	go cc.RunLoop(rb, func(stream *Stream) error {
		if stream.header.MsgTypeId == base.RtmpTypeIdVideo {
			assert.Equal(t, videoMsg.Header.TimestampAbs, stream.toAvMsg().Header.TimestampAbs)
			assert.Equal(t, videoMsg.Payload, stream.msg.buff.Bytes())
			c--
		} else if stream.header.MsgTypeId == base.RtmpTypeIdAudio {
			assert.Equal(t, audioMsg.Header.TimestampAbs, stream.toAvMsg().Header.TimestampAbs)
			assert.Equal(t, audioMsg.Payload, stream.toAvMsg().Payload)
			c--
		}

		if c == 0 {
			done <- struct{}{}
		}
		return nil
	})

	timer := time.NewTimer(1 * time.Second)

	select {
	case <-timer.C:
		assert.Equal(t, "", "error", "unit test timeout")
		break
	case <-done:
		break
	}
}
