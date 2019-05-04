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

func writeMessageHeader(writer io.Writer, csid int, bodyLen int, typeId int, streamId int) error {
	if _, err := writer.Write([]byte{uint8(0<<6 | csid), 0, 0, 0}); err != nil {
		return err
	}
	if err := bele.WriteBeUint24(writer, uint32(bodyLen)); err != nil {
		return err
	}
	if _, err := writer.Write([]byte{uint8(typeId)}); err != nil {
		return err
	}
	if err := bele.WriteLe(writer, uint32(streamId)); err != nil {
		return err
	}
	return nil
}

func (packer *MessagePacker) writeChunkSize(writer io.Writer, val int) error {
	if err := writeMessageHeader(packer.b, csidProtocolControl, 4, typeidSetChunkSize, 0); err != nil {
		return err
	}
	if err := bele.WriteBe(packer.b, uint32(val)); err != nil {
		return err
	}
	log.Infof("<----- SetChunkSize %d", val)
	if _, err := packer.b.WriteTo(writer); err != nil {
		return err
	}
	return nil
}

func (packer *MessagePacker) writeConect(writer io.Writer, appName, tcUrl string) error {
	if err := writeMessageHeader(packer.b, csidOverConnection, 0, typeidCommandMessageAMF0, 0); err != nil {
		return err
	}
	if err := Amf0.writeString(packer.b, "connect"); err != nil {
		return err
	}
	if err := Amf0.writeNumber(packer.b, float64(tidClientConnect)); err != nil {
		return err
	}
	// TODO chef: hack lal in
	objs := []ObjectPair{
		{key: "app", value: appName},
		{key: "type", value: "nonprivate"},
		{key: "flashVer", value: "FMLE/3.0 (compatible; Lal0.0.1)"},
		{key: "tcUrl", value: tcUrl},
	}
	if err := Amf0.writeObject(packer.b, objs); err != nil {
		return err
	}
	raw := packer.b.Bytes()
	bele.BePutUint24(raw[4:], uint32(len(raw)-12))
	log.Infof("<----- connect('%s')", appName)
	if _, err := packer.b.WriteTo(writer); err != nil {
		return err
	}
	return nil
}
