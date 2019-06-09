package rtmp

import (
	"bytes"
	"github.com/q191201771/lal/pkg/bele"
	"github.com/q191201771/lal/pkg/log"
	"io"
)

// TODO chef: direct use bufio.Writer

// TODO chef: add func to writeProtocolControlMessage.

const (
	peerBandwidthLimitTypeHard    = uint8(0)
	peerBandwidthLimitTypeSoft    = uint8(1)
	peerBandwidthLimitTypeDynamic = uint8(2)
)

type MessagePacker struct {
	b *bytes.Buffer
}

func NewMessagePacker() *MessagePacker {
	return &MessagePacker{
		b: &bytes.Buffer{},
	}
}

func writeMessageHeader(writer io.Writer, csid int, bodyLen int, typeID int, streamID int) error {
	if _, err := writer.Write([]byte{uint8(0<<6 | csid), 0, 0, 0}); err != nil {
		return err
	}
	if err := bele.WriteBEUint24(writer, uint32(bodyLen)); err != nil {
		return err
	}
	if _, err := writer.Write([]byte{uint8(typeID)}); err != nil {
		return err
	}
	if err := bele.WriteLE(writer, uint32(streamID)); err != nil {
		return err
	}
	return nil
}

func (packer *MessagePacker) writeProtocolControlMessage(writer io.Writer, typeID int, val int) error {
	if err := writeMessageHeader(packer.b, csidProtocolControl, 4, typeID, 0); err != nil {
		return err
	}
	if err := bele.WriteBE(packer.b, uint32(val)); err != nil {
		return err
	}
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeChunkSize(writer io.Writer, val int) error {
	log.Infof("<----- SetChunkSize %d", val)
	return packer.writeProtocolControlMessage(writer, typeidSetChunkSize, val)
}

func (packer *MessagePacker) writeWinAckSize(writer io.Writer, val int) error {
	log.Infof("<----- Window Acknowledgement Size %d", val)
	return packer.writeProtocolControlMessage(writer, typeidWinAckSize, val)
}

func (packer *MessagePacker) writePeerBandwidth(writer io.Writer, val int, limitType uint8) error {
	log.Infof("<----- Set Peer Bandwidth")

	if err := writeMessageHeader(packer.b, csidProtocolControl, 5, typeidBandwidth, 0); err != nil {
		return err
	}
	if err := bele.WriteBE(packer.b, uint32(val)); err != nil {
		return err
	}
	packer.b.WriteByte(limitType)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeConnect(writer io.Writer, appName, tcURL string) error {
	if err := writeMessageHeader(packer.b, csidOverConnection, 0, typeidCommandMessageAMF0, 0); err != nil {
		return err
	}
	if err := AMF0.WriteString(packer.b, "connect"); err != nil {
		return err
	}
	if err := AMF0.WriteNumber(packer.b, float64(tidClientConnect)); err != nil {
		return err
	}
	// TODO chef: hack lal in
	objs := []ObjectPair{
		{key: "app", value: appName},
		{key: "type", value: "nonprivate"},
		{key: "flashVer", value: "FMLE/3.0 (compatible; Lal0.0.1)"},
		{key: "tcUrl", value: tcURL},
	}
	if err := AMF0.WriteObject(packer.b, objs); err != nil {
		return err
	}
	raw := packer.b.Bytes()
	bele.BEPutUint24(raw[4:], uint32(len(raw)-12))
	log.Infof("<----- connect('%s')", appName)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeConnectResult(writer io.Writer, tid int) error {
	if err := writeMessageHeader(packer.b, csidOverConnection, 190, typeidCommandMessageAMF0, 0); err != nil {
		return err
	}
	if err := AMF0.WriteString(packer.b, "_result"); err != nil {
		return err
	}
	if err := AMF0.WriteNumber(packer.b, float64(tid)); err != nil {
		return err
	}
	objs := []ObjectPair{
		{key: "fmsVer", value: "FMS/3,0,1,123"},
		{key: "capabilities", value: 31},
	}
	if err := AMF0.WriteObject(packer.b, objs); err != nil {
		return err
	}
	objs = []ObjectPair{
		{key: "level", value: "status"},
		{key: "code", value: "NetConnection.Connect.Success"},
		{key: "description", value: "Connection succeeded."},
		{key: "objectEncoding", value: 0},
	}
	if err := AMF0.WriteObject(packer.b, objs); err != nil {
		return err
	}
	log.Infof("<----_result('NetConnection.Connect.Success')")
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeCreateStream(writer io.Writer) error {
	// 25 = 15 + 9 + 1
	if err := writeMessageHeader(packer.b, csidOverConnection, 25, typeidCommandMessageAMF0, 0); err != nil {
		return err
	}
	if err := AMF0.WriteString(packer.b, "createStream"); err != nil {
		return err
	}
	if err := AMF0.WriteNumber(packer.b, float64(tidClientCreateStream)); err != nil {
		return err
	}
	if err := AMF0.WriteNull(packer.b); err != nil {
		return err
	}
	log.Info("<----- createStream()")
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeCreateStreamResult(writer io.Writer, tid int) error {
	if err := writeMessageHeader(packer.b, csidOverConnection, 29, typeidCommandMessageAMF0, 0); err != nil {
		return err
	}
	if err := AMF0.WriteString(packer.b, "_result"); err != nil {
		return err
	}
	if err := AMF0.WriteNumber(packer.b, float64(tid)); err != nil {
		return err
	}
	if err := AMF0.WriteNull(packer.b); err != nil {
		return err
	}
	if err := AMF0.WriteNumber(packer.b, float64(msid)); err != nil {
		return err
	}
	log.Info("<----_result()")
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writePlay(writer io.Writer, streamName string, streamID int) error {
	if err := writeMessageHeader(packer.b, csidOverStream, 0, typeidCommandMessageAMF0, streamID); err != nil {
		return err
	}
	if err := AMF0.WriteString(packer.b, "play"); err != nil {
		return err
	}
	if err := AMF0.WriteNumber(packer.b, float64(tidClientPlay)); err != nil {
		return err
	}
	if err := AMF0.WriteNull(packer.b); err != nil {
		return err
	}
	if err := AMF0.WriteString(packer.b, streamName); err != nil {
		return err
	}

	raw := packer.b.Bytes()
	bele.BEPutUint24(raw[4:], uint32(len(raw)-12))
	log.Infof("<----- play('%s')", streamName)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writePublish(writer io.Writer, appName string, streamName string, streamID int) error {
	if err := writeMessageHeader(packer.b, csidOverStream, 0, typeidCommandMessageAMF0, streamID); err != nil {
		return err
	}
	if err := AMF0.WriteString(packer.b, "publish"); err != nil {
		return err
	}
	if err := AMF0.WriteNumber(packer.b, float64(tidClientPublish)); err != nil {
		return err
	}
	if err := AMF0.WriteNull(packer.b); err != nil {
		return err
	}
	if err := AMF0.WriteString(packer.b, streamName); err != nil {
		return err
	}
	if err := AMF0.WriteString(packer.b, appName); err != nil {
		return err
	}

	raw := packer.b.Bytes()
	bele.BEPutUint24(raw[4:], uint32(len(raw)-12))
	log.Infof("<----- publish('%s')", streamName)
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeOnStatusPublish(writer io.Writer, streamID int) error {
	if err := writeMessageHeader(packer.b, csidOverStream, 105, typeidCommandMessageAMF0, streamID); err != nil {
		return err
	}
	if err := AMF0.WriteString(packer.b, "onStatus"); err != nil {
		return err
	}
	if err := AMF0.WriteNumber(packer.b, 0); err != nil {
		return err
	}
	if err := AMF0.WriteNull(packer.b); err != nil {
		return err
	}
	objs := []ObjectPair{
		{key: "level", value: "status"},
		{key: "code", value: "NetStream.Publish.Start"},
		{key: "description", value: "Start publishing"},
	}
	if err := AMF0.WriteObject(packer.b, objs); err != nil {
		return err
	}
	log.Infof("<----onStatus('NetStream.Publish.Start')")
	_, err := packer.b.WriteTo(writer)
	return err
}

func (packer *MessagePacker) writeOnStatusPlay(writer io.Writer, streamID int) error {
	if err := writeMessageHeader(packer.b, csidOverStream, 96, typeidCommandMessageAMF0, streamID); err != nil {
		return err
	}
	if err := AMF0.WriteString(packer.b, "onStatus"); err != nil {
		return err
	}
	if err := AMF0.WriteNumber(packer.b, 0); err != nil {
		return err
	}
	if err := AMF0.WriteNull(packer.b); err != nil {
		return err
	}
	objs := []ObjectPair{
		{key: "level", value: "status"},
		{key: "code", value: "NetStream.Play.Start"},
		{key: "description", value: "Start live"},
	}
	if err := AMF0.WriteObject(packer.b, objs); err != nil {
		return err
	}
	log.Infof("<----onStatus('NetStream.Play.Start')")
	_, err := packer.b.WriteTo(writer)
	return err
}
