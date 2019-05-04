package rtmp

import (
	"github.com/q191201771/lal/bele"
	"io"
)

var (
	amf0TypeMarkerNumber      uint8 = 0x00 //
	amf0TypeMarkerBoolean     uint8 = 0x01
	amf0TypeMarkerString      uint8 = 0x02 //
	amf0TypeMarkerObject      uint8 = 0x03 //
	amf0TypeMarkerMovieclip   uint8 = 0x04
	amf0TypeMarkerNull        uint8 = 0x05
	amf0TypeMarkerUndefined   uint8 = 0x06
	amf0TypeMarkerReference   uint8 = 0x07
	amf0TypeMarkerEcmaArray   uint8 = 0x08
	amf0TypeMarkerObjectEnd   uint8 = 0x09 //
	amf0TypeMarkerStrictArray uint8 = 0x0a
	amf0TypeMarkerData        uint8 = 0x0b
	amf0TypeMarkerLongString  uint8 = 0x0c //
	amf0TypeMarkerUnsupported uint8 = 0x0d
	amf0TypeMarkerRecordset   uint8 = 0x0e
	amf0TypeMarkerXmlDocument uint8 = 0x0f
	amf0TypeMarkerTypedObject uint8 = 0x10

	amf0TypeMarkerObjectEndBytes = []byte{0, 0, amf0TypeMarkerObjectEnd}
)

type amf0 struct {
}

var Amf0 amf0

func (amf0) writeString(writer io.Writer, val string) error {
	if len(val) < 65536 {
		if _, err := writer.Write([]byte{amf0TypeMarkerString}); err != nil {
			return err
		}
		if err := bele.WriteBe(writer, uint16(len(val))); err != nil {
			return err
		}
	} else {
		if _, err := writer.Write([]byte{amf0TypeMarkerLongString}); err != nil {
			return err
		}
		if err := bele.WriteBe(writer, uint32(len(val))); err != nil {
			return err
		}
	}
	_, err := writer.Write([]byte(val))
	return err
}

func (amf0) writeNumber(writer io.Writer, val float64) error {
	if _, err := writer.Write([]byte{amf0TypeMarkerNumber}); err != nil {
		return err
	}
	return bele.WriteBe(writer, val)
}

type ObjectPair struct {
	key   string
	value interface{}
}

func (amf0) writeObject(writer io.Writer, objs []ObjectPair) error {
	if _, err := writer.Write([]byte{amf0TypeMarkerObject}); err != nil {
		return err
	}
	for i := 0; i < len(objs); i++ {
		if err := bele.WriteBe(writer, uint16(len(objs[i].key))); err != nil {
			return err
		}
		if _, err := writer.Write([]byte(objs[i].key)); err != nil {
			return err
		}
		switch objs[i].value.(type) {
		case string:
			if err := Amf0.writeString(writer, objs[i].value.(string)); err != nil {
				return err
			}
		default:
			panic(objs[i].value)
		}
	}
	_, err := writer.Write(amf0TypeMarkerObjectEndBytes)
	return err
}
