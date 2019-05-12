package rtmp

import (
	"bytes"
	"github.com/q191201771/lal/bele"
	"github.com/q191201771/lal/log"
	"io"
)

// TODO chef: direct use bufio.Writer

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

func (packer *MessagePacker) writeChunkSize(writer io.Writer, val int) error {
	if err := writeMessageHeader(packer.b, csidProtocolControl, 4, typeidSetChunkSize, 0); err != nil {
		return err
	}
	if err := bele.WriteBE(packer.b, uint32(val)); err != nil {
		return err
	}
	log.Infof("<----- SetChunkSize %d", val)
	if _, err := packer.b.WriteTo(writer); err != nil {
		return err
	}
	return nil
}

func (packer *MessagePacker) writeConnect(writer io.Writer, appName, tcURL string) error {
	if err := writeMessageHeader(packer.b, csidOverConnection, 0, typeidCommandMessageAMF0, 0); err != nil {
		return err
	}
	if err := AMF0.writeString(packer.b, "connect"); err != nil {
		return err
	}
	if err := AMF0.writeNumber(packer.b, float64(tidClientConnect)); err != nil {
		return err
	}
	// TODO chef: hack lal in
	objs := []ObjectPair{
		{key: "app", value: appName},
		{key: "type", value: "nonprivate"},
		{key: "flashVer", value: "FMLE/3.0 (compatible; Lal0.0.1)"},
		{key: "tcUrl", value: tcURL},
	}
	if err := AMF0.writeObject(packer.b, objs); err != nil {
		return err
	}
	raw := packer.b.Bytes()
	bele.BEPutUint24(raw[4:], uint32(len(raw)-12))
	log.Infof("<----- connect('%s')", appName)
	if _, err := packer.b.WriteTo(writer); err != nil {
		return err
	}
	return nil
}

func (packer *MessagePacker) writeCreateStream(writer io.Writer) error {
	// 25 = 15 + 9 + 1
	if err := writeMessageHeader(packer.b, csidOverConnection, 25, typeidCommandMessageAMF0, 0); err != nil {
		return err
	}
	if err := AMF0.writeString(packer.b, "createStream"); err != nil {
		return err
	}
	if err := AMF0.writeNumber(packer.b, float64(tidClientCreateStream)); err != nil {
		return err
	}
	if err := AMF0.writeNull(packer.b); err != nil {
		return err
	}
	log.Info("<----- createStream()")
	if _, err := packer.b.WriteTo(writer); err != nil {
		return err
	}
	return nil
}

func (packer *MessagePacker) writePlay(writer io.Writer, streamName string, streamID int) error {
	if err := writeMessageHeader(packer.b, csidOverStream, 0, typeidCommandMessageAMF0, streamID); err != nil {
		return err
	}
	if err := AMF0.writeString(packer.b, "play"); err != nil {
		return err
	}
	if err := AMF0.writeNumber(packer.b, float64(tidClientPlay)); err != nil {
		return err
	}
	if err := AMF0.writeNull(packer.b); err != nil {
		return err
	}
	if err := AMF0.writeString(packer.b, streamName); err != nil {
		return err
	}

	raw := packer.b.Bytes()
	bele.BEPutUint24(raw[4:], uint32(len(raw)-12))
	log.Infof("<----- play('%s')", streamName)
	if _, err := packer.b.WriteTo(writer); err != nil {
		return err
	}
	return nil
}

func (packer *MessagePacker) writePublish(writer io.Writer, appName string, streamName string, streamID int) error {
	if err := writeMessageHeader(packer.b, csidOverStream, 0, typeidCommandMessageAMF0, streamID); err != nil {
		return err
	}
	if err := AMF0.writeString(packer.b, "publish"); err != nil {
		return err
	}
	if err := AMF0.writeNumber(packer.b, float64(tidClientPublish)); err != nil {
		return err
	}
	if err := AMF0.writeNull(packer.b); err != nil {
		return err
	}
	if err := AMF0.writeString(packer.b, streamName); err != nil {
		return err
	}
	if err := AMF0.writeString(packer.b, appName); err != nil {
		return err
	}

	raw := packer.b.Bytes()
	bele.BEPutUint24(raw[4:], uint32(len(raw)-12))
	log.Infof("<----- publish('%s')", streamName)
	if _, err := packer.b.WriteTo(writer); err != nil {
		return err
	}
	return nil
}
