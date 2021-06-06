// Copyright 2019, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp

// message_packer.go
// @pure
// 打包并发送 rtmp 信令

import (
	"bytes"
	"fmt"
	"io"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/naza/pkg/bele"
)

const (
	peerBandwidthLimitTypeHard    = uint8(0)
	peerBandwidthLimitTypeSoft    = uint8(1)
	peerBandwidthLimitTypeDynamic = uint8(2)
)

type MessagePacker struct {
	// 1. 增加一层缓冲，避免 write 一个信令时发生多次系统调用
	// 2. 因为 bytes.Buffer.Write 返回的 error 永远为 nil，所以本文件中所有对 b 的写操作都不判断返回值
	b *bytes.Buffer
}

func NewMessagePacker() *MessagePacker {
	return &MessagePacker{
		b: &bytes.Buffer{},
	}
}

func (packer *MessagePacker) writeMessageHeader(csid int, bodyLen int, typeid uint8, streamid int) {
	// 目前这个函数只供发送信令时调用，信令的 csid 都是小于等于 63 的，如果传入的 csid 大于 63，直接 panic
	if csid > 63 {
		panic(csid)
	}

	fmt := 0
	// 0 0 0 是时间戳
	_, _ = packer.b.Write([]byte{uint8(fmt<<6 | csid), 0, 0, 0})
	_ = bele.WriteBeUint24(packer.b, uint32(bodyLen))
	_ = packer.b.WriteByte(typeid)
	_ = bele.WriteLe(packer.b, uint32(streamid))
}

func (packer *MessagePacker) writeProtocolControlMessage(writer io.Writer, typeid uint8, val int) error {
	packer.writeMessageHeader(csidProtocolControl, 4, typeid, 0)
	_ = bele.WriteBe(packer.b, uint32(val))
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeChunkSize(writer io.Writer, val int) error {
	return packer.writeProtocolControlMessage(writer, base.RtmpTypeIdSetChunkSize, val)
}

func (packer *MessagePacker) writeWinAckSize(writer io.Writer, val int) error {
	return packer.writeProtocolControlMessage(writer, base.RtmpTypeIdWinAckSize, val)
}

func (packer *MessagePacker) writePeerBandwidth(writer io.Writer, val int, limitType uint8) error {
	packer.writeMessageHeader(csidProtocolControl, 5, base.RtmpTypeIdBandwidth, 0)
	_ = bele.WriteBe(packer.b, uint32(val))
	_ = packer.b.WriteByte(limitType)
	_, err := packer.b.WriteTo(writer)
	return err
}

// @param isPush: 推流为true，拉流为false
func (packer *MessagePacker) writeConnect(writer io.Writer, appName, tcUrl string, isPush bool) error {
	packer.writeMessageHeader(csidOverConnection, 0, base.RtmpTypeIdCommandMessageAmf0, 0)
	_ = Amf0.WriteString(packer.b, "connect")
	_ = Amf0.WriteNumber(packer.b, float64(tidClientConnect))

	var objs []ObjectPair
	objs = append(objs, ObjectPair{Key: "app", Value: appName})
	objs = append(objs, ObjectPair{Key: "type", Value: "nonprivate"})
	var flashVer string
	if isPush {
		flashVer = fmt.Sprintf("FMLE/3.0 (compatible; %s)", base.LalRtmpPushSessionConnectVersion)
	} else {
		flashVer = "LNX 9,0,124,2"
	}
	objs = append(objs, ObjectPair{Key: "flashVer", Value: flashVer})
	// fpad True if proxy is being used.
	objs = append(objs, ObjectPair{Key: "fpad", Value: false})
	objs = append(objs, ObjectPair{Key: "tcUrl", Value: tcUrl})
	_ = Amf0.WriteObject(packer.b, objs)
	raw := packer.b.Bytes()
	bele.BePutUint24(raw[4:], uint32(len(raw)-12))
	_, err := packer.b.WriteTo(writer)
	return err
}

// @param objectEncoding 设置0或者3，表示是Amf0或AMF3，上层可根据connect信令中的objectEncoding值设置该值
func (packer *MessagePacker) writeConnectResult(writer io.Writer, tid int, objectEncoding int) error {
	packer.writeMessageHeader(csidOverConnection, 0, base.RtmpTypeIdCommandMessageAmf0, 0)
	_ = Amf0.WriteString(packer.b, "_result")
	_ = Amf0.WriteNumber(packer.b, float64(tid))
	objs := []ObjectPair{
		{Key: "fmsVer", Value: "FMS/3,0,1,123"},
		{Key: "capabilities", Value: 31},
	}
	_ = Amf0.WriteObject(packer.b, objs)
	objs = []ObjectPair{
		{Key: "level", Value: "status"},
		{Key: "code", Value: "NetConnection.Connect.Success"},
		{Key: "description", Value: "Connection succeeded."},
		{Key: "objectEncoding", Value: objectEncoding},
		{Key: "version", Value: base.LalRtmpConnectResultVersion},
	}
	_ = Amf0.WriteObject(packer.b, objs)
	raw := packer.b.Bytes()
	bele.BePutUint24(raw[4:], uint32(len(raw)-12))
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeCreateStream(writer io.Writer) error {
	// 25 = 15 + 9 + 1
	packer.writeMessageHeader(csidOverConnection, 25, base.RtmpTypeIdCommandMessageAmf0, 0)
	_ = Amf0.WriteString(packer.b, "createStream")
	_ = Amf0.WriteNumber(packer.b, float64(tidClientCreateStream))
	_ = Amf0.WriteNull(packer.b)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeCreateStreamResult(writer io.Writer, tid int) error {
	packer.writeMessageHeader(csidOverConnection, 29, base.RtmpTypeIdCommandMessageAmf0, 0)
	_ = Amf0.WriteString(packer.b, "_result")
	_ = Amf0.WriteNumber(packer.b, float64(tid))
	_ = Amf0.WriteNull(packer.b)
	_ = Amf0.WriteNumber(packer.b, float64(Msid1))
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writePlay(writer io.Writer, streamName string, streamid int) error {
	packer.writeMessageHeader(csidOverStream, 0, base.RtmpTypeIdCommandMessageAmf0, streamid)
	_ = Amf0.WriteString(packer.b, "play")
	_ = Amf0.WriteNumber(packer.b, float64(tidClientPlay))
	_ = Amf0.WriteNull(packer.b)
	_ = Amf0.WriteString(packer.b, streamName)

	raw := packer.b.Bytes()
	bele.BePutUint24(raw[4:], uint32(len(raw)-12))
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writePublish(writer io.Writer, appName string, streamName string, streamid int) error {
	packer.writeMessageHeader(csidOverStream, 0, base.RtmpTypeIdCommandMessageAmf0, streamid)
	_ = Amf0.WriteString(packer.b, "publish")
	_ = Amf0.WriteNumber(packer.b, float64(tidClientPublish))
	_ = Amf0.WriteNull(packer.b)
	_ = Amf0.WriteString(packer.b, streamName)
	_ = Amf0.WriteString(packer.b, appName)

	raw := packer.b.Bytes()
	bele.BePutUint24(raw[4:], uint32(len(raw)-12))
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeOnStatusPublish(writer io.Writer, streamid int) error {
	packer.writeMessageHeader(csidOverStream, 105, base.RtmpTypeIdCommandMessageAmf0, streamid)
	_ = Amf0.WriteString(packer.b, "onStatus")
	_ = Amf0.WriteNumber(packer.b, 0)
	_ = Amf0.WriteNull(packer.b)
	objs := []ObjectPair{
		{Key: "level", Value: "status"},
		{Key: "code", Value: "NetStream.Publish.Start"},
		{Key: "description", Value: "Start publishing"},
	}
	_ = Amf0.WriteObject(packer.b, objs)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeOnStatusPlay(writer io.Writer, streamid int) error {
	packer.writeMessageHeader(csidOverStream, 96, base.RtmpTypeIdCommandMessageAmf0, streamid)
	_ = Amf0.WriteString(packer.b, "onStatus")
	_ = Amf0.WriteNumber(packer.b, 0)
	_ = Amf0.WriteNull(packer.b)
	objs := []ObjectPair{
		{Key: "level", Value: "status"},
		{Key: "code", Value: "NetStream.Play.Start"},
		{Key: "description", Value: "Start live"},
	}
	_ = Amf0.WriteObject(packer.b, objs)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeStreamIsRecorded(writer io.Writer, streamid uint32) error {
	packer.writeMessageHeader(csidProtocolControl, 6, base.RtmpTypeIdUserControl, 0)
	_ = bele.WriteBe(packer.b, uint16(base.RtmpUserControlRecorded))
	_ = bele.WriteBe(packer.b, uint32(streamid))
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeStreamBegin(writer io.Writer, streamid uint32) error {
	packer.writeMessageHeader(csidProtocolControl, 6, base.RtmpTypeIdUserControl, 0)
	_ = bele.WriteBe(packer.b, uint16(base.RtmpUserControlStreamBegin))
	_ = bele.WriteBe(packer.b, uint32(streamid))
	_, err := packer.b.WriteTo(writer)
	return err
}
