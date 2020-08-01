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

func (packer *MessagePacker) writeMessageHeader(csid int, bodyLen int, typeID uint8, streamID int) {
	// 目前这个函数只供发送信令时调用，信令的 csid 都是小于等于 63 的，如果传入的 csid 大于 63，直接 panic
	if csid > 63 {
		panic(csid)
	}

	fmt := 0
	// 0 0 0 是时间戳
	_, _ = packer.b.Write([]byte{uint8(fmt<<6 | csid), 0, 0, 0})
	_ = bele.WriteBEUint24(packer.b, uint32(bodyLen))
	_ = packer.b.WriteByte(typeID)
	_ = bele.WriteLE(packer.b, uint32(streamID))
}

func (packer *MessagePacker) writeProtocolControlMessage(writer io.Writer, typeID uint8, val int) error {
	packer.writeMessageHeader(csidProtocolControl, 4, typeID, 0)
	_ = bele.WriteBE(packer.b, uint32(val))
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeChunkSize(writer io.Writer, val int) error {
	return packer.writeProtocolControlMessage(writer, base.RTMPTypeIDSetChunkSize, val)
}

func (packer *MessagePacker) writeWinAckSize(writer io.Writer, val int) error {
	return packer.writeProtocolControlMessage(writer, base.RTMPTypeIDWinAckSize, val)
}

func (packer *MessagePacker) writePeerBandwidth(writer io.Writer, val int, limitType uint8) error {
	packer.writeMessageHeader(csidProtocolControl, 5, base.RTMPTypeIDBandwidth, 0)
	_ = bele.WriteBE(packer.b, uint32(val))
	_ = packer.b.WriteByte(limitType)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeConnect(writer io.Writer, appName, tcURL string) error {
	packer.writeMessageHeader(csidOverConnection, 0, base.RTMPTypeIDCommandMessageAMF0, 0)
	_ = AMF0.WriteString(packer.b, "connect")
	_ = AMF0.WriteNumber(packer.b, float64(tidClientConnect))

	objs := []ObjectPair{
		{Key: "app", Value: appName},
		{Key: "type", Value: "nonprivate"},
		{Key: "flashVer", Value: "FMLE/3.0 (compatible; Lal0.0.1)"},
		{Key: "tcUrl", Value: tcURL},
	}
	_ = AMF0.WriteObject(packer.b, objs)
	raw := packer.b.Bytes()
	bele.BEPutUint24(raw[4:], uint32(len(raw)-12))
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeConnectResult(writer io.Writer, tid int) error {
	packer.writeMessageHeader(csidOverConnection, 190, base.RTMPTypeIDCommandMessageAMF0, 0)
	_ = AMF0.WriteString(packer.b, "_result")
	_ = AMF0.WriteNumber(packer.b, float64(tid))
	objs := []ObjectPair{
		{Key: "fmsVer", Value: "FMS/3,0,1,123"},
		{Key: "capabilities", Value: 31},
	}
	_ = AMF0.WriteObject(packer.b, objs)
	objs = []ObjectPair{
		{Key: "level", Value: "status"},
		{Key: "code", Value: "NetConnection.Connect.Success"},
		{Key: "description", Value: "Connection succeeded."},
		{Key: "objectEncoding", Value: 0},
	}
	_ = AMF0.WriteObject(packer.b, objs)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeCreateStream(writer io.Writer) error {
	// 25 = 15 + 9 + 1
	packer.writeMessageHeader(csidOverConnection, 25, base.RTMPTypeIDCommandMessageAMF0, 0)
	_ = AMF0.WriteString(packer.b, "createStream")
	_ = AMF0.WriteNumber(packer.b, float64(tidClientCreateStream))
	_ = AMF0.WriteNull(packer.b)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeCreateStreamResult(writer io.Writer, tid int) error {
	packer.writeMessageHeader(csidOverConnection, 29, base.RTMPTypeIDCommandMessageAMF0, 0)
	_ = AMF0.WriteString(packer.b, "_result")
	_ = AMF0.WriteNumber(packer.b, float64(tid))
	_ = AMF0.WriteNull(packer.b)
	_ = AMF0.WriteNumber(packer.b, float64(MSID1))
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writePlay(writer io.Writer, streamName string, streamID int) error {
	packer.writeMessageHeader(csidOverStream, 0, base.RTMPTypeIDCommandMessageAMF0, streamID)
	_ = AMF0.WriteString(packer.b, "play")
	_ = AMF0.WriteNumber(packer.b, float64(tidClientPlay))
	_ = AMF0.WriteNull(packer.b)
	_ = AMF0.WriteString(packer.b, streamName)

	raw := packer.b.Bytes()
	bele.BEPutUint24(raw[4:], uint32(len(raw)-12))
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writePublish(writer io.Writer, appName string, streamName string, streamID int) error {
	packer.writeMessageHeader(csidOverStream, 0, base.RTMPTypeIDCommandMessageAMF0, streamID)
	_ = AMF0.WriteString(packer.b, "publish")
	_ = AMF0.WriteNumber(packer.b, float64(tidClientPublish))
	_ = AMF0.WriteNull(packer.b)
	_ = AMF0.WriteString(packer.b, streamName)
	_ = AMF0.WriteString(packer.b, appName)

	raw := packer.b.Bytes()
	bele.BEPutUint24(raw[4:], uint32(len(raw)-12))
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeOnStatusPublish(writer io.Writer, streamID int) error {
	packer.writeMessageHeader(csidOverStream, 105, base.RTMPTypeIDCommandMessageAMF0, streamID)
	_ = AMF0.WriteString(packer.b, "onStatus")
	_ = AMF0.WriteNumber(packer.b, 0)
	_ = AMF0.WriteNull(packer.b)
	objs := []ObjectPair{
		{Key: "level", Value: "status"},
		{Key: "code", Value: "NetStream.Publish.Start"},
		{Key: "description", Value: "Start publishing"},
	}
	_ = AMF0.WriteObject(packer.b, objs)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeOnStatusPlay(writer io.Writer, streamID int) error {
	packer.writeMessageHeader(csidOverStream, 96, base.RTMPTypeIDCommandMessageAMF0, streamID)
	_ = AMF0.WriteString(packer.b, "onStatus")
	_ = AMF0.WriteNumber(packer.b, 0)
	_ = AMF0.WriteNull(packer.b)
	objs := []ObjectPair{
		{Key: "level", Value: "status"},
		{Key: "code", Value: "NetStream.Play.Start"},
		{Key: "description", Value: "Start live"},
	}
	_ = AMF0.WriteObject(packer.b, objs)
	_, err := packer.b.WriteTo(writer)
	return err
}
