package avc

import (
	"errors"
	"github.com/q191201771/lal/pkg/util/bele"
)


var avcErr = errors.New("avc: fxxk")

var NaluStartCode = []byte{0x0, 0x0, 0x0, 0x1}

var NaluUintTypeMapping = map[uint8]string{
	1:"SLICE",
	5:"IDR",
	6:"SEI",
	7:"SPS",
	8:"PPS",
	9:"AUD",
}

// H.264-AVC-ISO_IEC_14496-15.pdf
// 5.2.4 Decoder configuration information
// <buf> body of tag
func ParseAVCSeqHeader(buf []byte) (sps, pps []byte, err error) {
	// TODO chef: check if read out of <buf> range

	if buf[0] != 0x17 || buf[1] != 0x00 || buf[2] != 0 || buf[3] != 0 || buf[4] != 0 {
		err = avcErr
		return
	}

	//configurationVersion := buf[5]
	//avcProfileIndication := buf[6]
	//profileCompatibility := buf[7]
	//avcLevelIndication := buf[8]
	//lengthSizeMinusOne := buf[9] & 0x03

	index := 10

	numOfSPS := int(buf[index] & 0x1F)
	index++
	// TODO chef: if the situation of multi sps exist?
	// only take the last one.
	for i := 0; i < numOfSPS; i++ {
		lenOfSPS := int(bele.BEUint16(buf[index:]))
		index += 2
		sps = append(sps, buf[index:index+lenOfSPS]...)
		index += lenOfSPS
	}

	numOfPPS := int(buf[index] & 0x1F)
	index++
	for i := 0; i < numOfPPS; i++ {
		lenOfPPS := int(bele.BEUint16(buf[index:]))
		index += 2
		pps = append(pps, buf[index:index+lenOfPPS]...)
		index += lenOfPPS
	}

	return
}
