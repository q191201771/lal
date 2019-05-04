package bele

import (
	"encoding/binary"
	"io"
)

// assume local is `le`

func BeUint16(p []byte) uint16 {
	return binary.BigEndian.Uint16(p)
}

func BeUint24(p []byte) (ret uint32) {
	ret = 0
	ret |= uint32(p[0]) << 16
	ret |= uint32(p[1]) << 8
	ret |= uint32(p[2])
	return
}

func BeUint32(p []byte) (ret uint32) {
	return binary.BigEndian.Uint32(p)
}

func LeUint32(p []byte) (ret uint32) {
	return binary.LittleEndian.Uint32(p)
}

func BePutUint24(out []byte, in uint32) {
	out[0] = byte(in >> 16)
	out[1] = byte(in >> 8)
	out[2] = byte(in)
}

func BePutUint32(out []byte, in uint32) {
	binary.BigEndian.PutUint32(out, in)
}

func WriteBeUint24(writer io.Writer, in uint32) error {
	_, err := writer.Write([]byte{uint8(in >> 16), uint8(in >> 8), uint8(in & 0xFF)})
	return err
}

func WriteBe(writer io.Writer, in interface{}) error {
	return binary.Write(writer, binary.BigEndian, in)
}

func WriteLe(writer io.Writer, in interface{}) error {
	return binary.Write(writer, binary.LittleEndian, in)
}
