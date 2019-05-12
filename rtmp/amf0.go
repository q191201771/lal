package rtmp

import (
	"bytes"
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

type ObjectPair struct {
	key   string
	value interface{}
}

type amf0 struct {
}

var AMF0 amf0

func (amf0) writeString(writer io.Writer, val string) error {
	if len(val) < 65536 {
		if _, err := writer.Write([]byte{amf0TypeMarkerString}); err != nil {
			return err
		}
		if err := bele.WriteBE(writer, uint16(len(val))); err != nil {
			return err
		}
	} else {
		if _, err := writer.Write([]byte{amf0TypeMarkerLongString}); err != nil {
			return err
		}
		if err := bele.WriteBE(writer, uint32(len(val))); err != nil {
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
	return bele.WriteBE(writer, val)
}

func (amf0) writeNull(writer io.Writer) error {
	_, err := writer.Write([]byte{amf0TypeMarkerNull})
	return err
}

func (amf0) writeObject(writer io.Writer, objs []ObjectPair) error {
	if _, err := writer.Write([]byte{amf0TypeMarkerObject}); err != nil {
		return err
	}
	for i := 0; i < len(objs); i++ {
		if err := bele.WriteBE(writer, uint16(len(objs[i].key))); err != nil {
			return err
		}
		if _, err := writer.Write([]byte(objs[i].key)); err != nil {
			return err
		}
		switch objs[i].value.(type) {
		case string:
			if err := AMF0.writeString(writer, objs[i].value.(string)); err != nil {
				return err
			}
		default:
			panic(objs[i].value)
		}
	}
	_, err := writer.Write(amf0TypeMarkerObjectEndBytes)
	return err
}

// @return <string val> <consumed len> <error>
func (amf0) readString(b []byte) (string, int, error) {
	// TODO chef: long string.
	if len(b) < 2 {
		return "", 0, rtmpErr
	}
	l := int(bele.BEUint16(b))
	if l > len(b)-2 {
		return "", 0, rtmpErr
	}
	return string(b[2 : 2+l]), 2 + l, nil
}

// @return <string val> <consumed len> <error>
func (amf0) readStringWithType(b []byte) (string, int, error) {
	if len(b) < 1 || b[0] != amf0TypeMarkerString {
		return "", 0, rtmpErr
	}
	val, l, err := AMF0.readString(b[1:])
	return val, l + 1, err
}

// @return <number val> <consumed len> <error>
func (amf0) readNumberWithType(b []byte) (float64, int, error) {
	if len(b) < 9 || b[0] != amf0TypeMarkerNumber {
		return 0, 0, rtmpErr
	}
	val := bele.BEFloat64(b[1:])
	return val, 9, nil
}

// @return <object val> <consumed len> <error>
func (amf0) readObject(b []byte) (map[string]interface{}, int, error) {
	if len(b) < 1 || b[0] != amf0TypeMarkerObject {
		return nil, 0, rtmpErr
	}

	index := 1
	obj := make(map[string]interface{})
	for {
		if len(b)-index >= 3 && bytes.Equal(b[index:index+3], amf0TypeMarkerObjectEndBytes) {
			return obj, index + 3, nil
		}

		k, l, err := AMF0.readString(b[index:])
		if err != nil {
			return nil, 0, err
		}
		index += l
		if len(b)-index < 1 {
			return nil, 0, rtmpErr
		}
		vt := b[index]
		switch vt {
		case amf0TypeMarkerString:
			v, l, err := AMF0.readStringWithType(b[index:])
			if err != nil {
				return nil, 0, rtmpErr
			}
			obj[k] = v
			index += l
		case amf0TypeMarkerNumber:
			v, l, err := AMF0.readNumberWithType(b[index:])
			if err != nil {
				return nil, 0, rtmpErr
			}
			obj[k] = v
			index += l
		default:
			panic(vt)
		}
	}

	// never reach here.
	return nil, 0, rtmpErr
}
