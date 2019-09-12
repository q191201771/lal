package rtmp

// message_packer.go
// @pure
// 打包并发送 rtmp 信令

import (
	"bytes"
	"github.com/q191201771/nezha/pkg/bele"
	"io"
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

func (packer *MessagePacker) writeMessageHeader(csid int, bodyLen int, typeID int, streamID int) {
	// 目前这个函数只供发送信令时调用，信令的 csid 都是小于等于 63 的，如果传入的 csid 大于 63，直接 panic
	if csid > 63 {
		panic(csid)
	}

	fmt := 0
	// 0 0 0 是时间戳
	_, _ = packer.b.Write([]byte{uint8(fmt<<6 | csid), 0, 0, 0})
	_ = bele.WriteBEUint24(packer.b, uint32(bodyLen))
	_, _ = packer.b.Write([]byte{uint8(typeID)})
	_ = bele.WriteLE(packer.b, uint32(streamID))
}

func (packer *MessagePacker) writeProtocolControlMessage(writer io.Writer, typeID int, val int) error {
	packer.writeMessageHeader(csidProtocolControl, 4, typeID, 0)
	_ = bele.WriteBE(packer.b, uint32(val))
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeChunkSize(writer io.Writer, val int) error {
	return packer.writeProtocolControlMessage(writer, typeidSetChunkSize, val)
}

func (packer *MessagePacker) writeWinAckSize(writer io.Writer, val int) error {
	return packer.writeProtocolControlMessage(writer, typeidWinAckSize, val)
}

func (packer *MessagePacker) writePeerBandwidth(writer io.Writer, val int, limitType uint8) error {
	packer.writeMessageHeader(csidProtocolControl, 5, typeidBandwidth, 0)
	_ = bele.WriteBE(packer.b, uint32(val))
	_ = packer.b.WriteByte(limitType)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeConnect(writer io.Writer, appName, tcURL string) error {
	packer.writeMessageHeader(csidOverConnection, 0, typeidCommandMessageAMF0, 0)
	_ = AMF0.WriteString(packer.b, "connect")
	_ = AMF0.WriteNumber(packer.b, float64(tidClientConnect))

	// TODO chef: hack lal in
	objs := []ObjectPair{
		{key: "app", value: appName},
		{key: "type", value: "nonprivate"},
		{key: "flashVer", value: "FMLE/3.0 (compatible; Lal0.0.1)"},
		{key: "tcUrl", value: tcURL},
	}
	_ = AMF0.WriteObject(packer.b, objs)
	raw := packer.b.Bytes()
	bele.BEPutUint24(raw[4:], uint32(len(raw)-12))
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeConnectResult(writer io.Writer, tid int) error {
	packer.writeMessageHeader(csidOverConnection, 190, typeidCommandMessageAMF0, 0)
	_ = AMF0.WriteString(packer.b, "_result")
	_ = AMF0.WriteNumber(packer.b, float64(tid))
	objs := []ObjectPair{
		{key: "fmsVer", value: "FMS/3,0,1,123"},
		{key: "capabilities", value: 31},
	}
	_ = AMF0.WriteObject(packer.b, objs)
	objs = []ObjectPair{
		{key: "level", value: "status"},
		{key: "code", value: "NetConnection.Connect.Success"},
		{key: "description", value: "Connection succeeded."},
		{key: "objectEncoding", value: 0},
	}
	_ = AMF0.WriteObject(packer.b, objs)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeCreateStream(writer io.Writer) error {
	// 25 = 15 + 9 + 1
	packer.writeMessageHeader(csidOverConnection, 25, typeidCommandMessageAMF0, 0)
	_ = AMF0.WriteString(packer.b, "createStream")
	_ = AMF0.WriteNumber(packer.b, float64(tidClientCreateStream))
	_ = AMF0.WriteNull(packer.b)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeCreateStreamResult(writer io.Writer, tid int) error {
	packer.writeMessageHeader(csidOverConnection, 29, typeidCommandMessageAMF0, 0)
	_ = AMF0.WriteString(packer.b, "_result")
	_ = AMF0.WriteNumber(packer.b, float64(tid))
	_ = AMF0.WriteNull(packer.b)
	_ = AMF0.WriteNumber(packer.b, float64(MSID1))
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writePlay(writer io.Writer, streamName string, streamID int) error {
	packer.writeMessageHeader(csidOverStream, 0, typeidCommandMessageAMF0, streamID)
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
	packer.writeMessageHeader(csidOverStream, 0, typeidCommandMessageAMF0, streamID)
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
	packer.writeMessageHeader(csidOverStream, 105, typeidCommandMessageAMF0, streamID)
	_ = AMF0.WriteString(packer.b, "onStatus")
	_ = AMF0.WriteNumber(packer.b, 0)
	_ = AMF0.WriteNull(packer.b)
	objs := []ObjectPair{
		{key: "level", value: "status"},
		{key: "code", value: "NetStream.Publish.Start"},
		{key: "description", value: "Start publishing"},
	}
	_ = AMF0.WriteObject(packer.b, objs)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeOnStatusPlay(writer io.Writer, streamID int) error {
	packer.writeMessageHeader(csidOverStream, 96, typeidCommandMessageAMF0, streamID)
	_ = AMF0.WriteString(packer.b, "onStatus")
	_ = AMF0.WriteNumber(packer.b, 0)
	_ = AMF0.WriteNull(packer.b)
	objs := []ObjectPair{
		{key: "level", value: "status"},
		{key: "code", value: "NetStream.Play.Start"},
		{key: "description", value: "Start live"},
	}
	_ = AMF0.WriteObject(packer.b, objs)
	_, err := packer.b.WriteTo(writer)
	return err
}
