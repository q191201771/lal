// Copyright 2021, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic_test

import (
	"encoding/hex"
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/logic"
	"github.com/q191201771/naza/pkg/assert"
	"github.com/q191201771/naza/pkg/nazalog"
	"strconv"
	"strings"
	"testing"
)

func TestDummyAudioFilter(t *testing.T) {
	// case1 一个音视频都有的流
	{
		in := []base.RtmpMsg{
			helperUnpackRtmpMsg("header={Csid:4 MsgLen:378 MsgTypeId:18 MsgStreamId:1 TimestampAbs:0}, payload=02000d40"),
			helperUnpackRtmpMsg("header={Csid:6 MsgLen:48 MsgTypeId:9 MsgStreamId:1 TimestampAbs:0}, payload=17000000"),
			helperUnpackRtmpMsg("header={Csid:4 MsgLen:7 MsgTypeId:8 MsgStreamId:1 TimestampAbs:0}, payload=af001210"),
			helperUnpackRtmpMsg("header={Csid:4 MsgLen:26 MsgTypeId:8 MsgStreamId:1 TimestampAbs:0}, payload=af01de04"),
			helperUnpackRtmpMsg("header={Csid:6 MsgLen:1170 MsgTypeId:9 MsgStreamId:1 TimestampAbs:23}, payload=17010000"),
			helperUnpackRtmpMsg("header={Csid:4 MsgLen:8 MsgTypeId:8 MsgStreamId:1 TimestampAbs:23}, payload=af012110"),
			helperUnpackRtmpMsg("header={Csid:4 MsgLen:8 MsgTypeId:8 MsgStreamId:1 TimestampAbs:46}, payload=af012120"),
			helperUnpackRtmpMsg("header={Csid:4 MsgLen:849 MsgTypeId:8 MsgStreamId:1 TimestampAbs:69}, payload=af01214c"),
			helperUnpackRtmpMsg("header={Csid:6 MsgLen:372 MsgTypeId:9 MsgStreamId:1 TimestampAbs:90}, payload=27010000"),
		}
		var out []base.RtmpMsg
		filter := logic.NewDummyAudioFilter("test1", 150, func(msg base.RtmpMsg) {
			out = append(out, msg)
		})
		//filter.Feed(helperUnpackRtmpMsg(""))
		for i := 0; i <= 1; i++ {
			filter.Feed(in[i])
			assert.Equal(t, nil, out)
		}
		for i := 2; i < len(in); i++ {
			filter.Feed(in[i])
			assert.Equal(t, in[:i+1], out)
		}
	}

	// case2 一个只有视频的流
	{
		in := []base.RtmpMsg{
			helperUnpackRtmpMsg("header={Csid:4 MsgLen:269 MsgTypeId:18 MsgStreamId:1 TimestampAbs:0}, payload=02000d4073657444"),
			helperUnpackRtmpMsg("header={Csid:6 MsgLen:48 MsgTypeId:9 MsgStreamId:1 TimestampAbs:0}, payload=1700000000016400"),
			helperUnpackRtmpMsg("header={Csid:6 MsgLen:1170 MsgTypeId:9 MsgStreamId:1 TimestampAbs:23}, payload=1701000000000002"),
			helperUnpackRtmpMsg("header={Csid:6 MsgLen:372 MsgTypeId:9 MsgStreamId:1 TimestampAbs:90}, payload=2701000000000001"),
			helperUnpackRtmpMsg("header={Csid:6 MsgLen:1226 MsgTypeId:9 MsgStreamId:1 TimestampAbs:156}, payload=2701000000000004"),
			helperUnpackRtmpMsg("header={Csid:6 MsgLen:1541 MsgTypeId:9 MsgStreamId:1 TimestampAbs:223}, payload=2701000000000005"),
			helperUnpackRtmpMsg("header={Csid:6 MsgLen:1931 MsgTypeId:9 MsgStreamId:1 TimestampAbs:290}, payload=2701000000000005"),
		}
		var out []base.RtmpMsg
		filter := logic.NewDummyAudioFilter("test1", 150, func(msg base.RtmpMsg) {
			out = append(out, msg)
		})
		for i := 0; i <= 4; i++ {
			filter.Feed(in[i])
			assert.Equal(t, nil, out)
		}
		filter.Feed(in[5])
		assert.Equal(t, 17, len(out))
		assert.Equal(t, in[0], out[0])
		assert.Equal(t, helperUnpackRtmpMsg("header={Csid:6 MsgLen:4 MsgTypeId:8 MsgStreamId:1 TimestampAbs:0}, payload=af001190"), out[1])
		assert.Equal(t, in[1], out[2])
		assert.Equal(t, helperUnpackRtmpMsg("header={Csid:6 MsgLen:8 MsgTypeId:8 MsgStreamId:1 TimestampAbs:215}, payload=af01211004608c1c"), out[15])
		assert.Equal(t, in[5], out[16])

		filter.Feed(in[6])
		assert.Equal(t, 21, len(out))
		assert.Equal(t, helperUnpackRtmpMsg("header={Csid:6 MsgLen:8 MsgTypeId:8 MsgStreamId:1 TimestampAbs:236}, payload=af01211004608c1c"), out[17])
		assert.Equal(t, in[6], out[20])
	}

	// case3 一个只有音频的流
	{
		in := []base.RtmpMsg{
			helperUnpackRtmpMsg("header={Csid:4 MsgLen:278 MsgTypeId:18 MsgStreamId:1 TimestampAbs:0}, payload=02000d4073657444"),
			helperUnpackRtmpMsg("header={Csid:4 MsgLen:7 MsgTypeId:8 MsgStreamId:1 TimestampAbs:0}, payload=af00121056e500"),
			helperUnpackRtmpMsg("header={Csid:4 MsgLen:26 MsgTypeId:8 MsgStreamId:1 TimestampAbs:0}, payload=af01de04004c6176"),
			helperUnpackRtmpMsg("header={Csid:4 MsgLen:8 MsgTypeId:8 MsgStreamId:1 TimestampAbs:23}, payload=af01211004608c1c"),
		}
		var out []base.RtmpMsg
		filter := logic.NewDummyAudioFilter("test1", 150, func(msg base.RtmpMsg) {
			out = append(out, msg)
		})
		filter.Feed(in[0])
		assert.Equal(t, nil, out)
		for i := 1; i <= 3; i++ {
			filter.Feed(in[i])
			assert.Equal(t, in[:i+1], out)
		}
	}
}

// @param logstr e.g. "header={Csid:4 MsgLen:378 MsgTypeId:18 MsgStreamId:1 TimestampAbs:0}"
///
func helperUnpackRtmpMsg(logstr string) base.RtmpMsg {
	var fetchItemFn = func(str string, prefix string, suffix string) string {
		b := strings.Index(str, prefix)
		if suffix == "" {
			return str[b+len(prefix):]
		}
		e := strings.Index(str[b:], suffix)
		return str[b+len(prefix) : b+e]
	}
	var fetchIntItemFn = func(str string, prefix string, suffix string) int {
		ret, err := strconv.Atoi(fetchItemFn(str, prefix, suffix))
		nazalog.Assert(nil, err)
		return ret
	}

	var header base.RtmpHeader
	header.Csid = fetchIntItemFn(logstr, "Csid:", " ")
	header.MsgLen = uint32(fetchIntItemFn(logstr, "MsgLen:", " "))
	header.MsgTypeId = uint8(fetchIntItemFn(logstr, "MsgTypeId:", " "))
	header.MsgStreamId = fetchIntItemFn(logstr, "MsgStreamId:", " ")
	header.TimestampAbs = uint32(fetchIntItemFn(logstr, "TimestampAbs:", "}"))

	hexStr := fetchItemFn(logstr, "payload=", "")
	payload, err := hex.DecodeString(hexStr)
	nazalog.Assert(nil, err)

	return base.RtmpMsg{
		Header:  header,
		Payload: payload,
	}
}
