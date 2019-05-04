package httpflv

import (
	"github.com/q191201771/lal/bele"
	"github.com/q191201771/lal/log"
)

// TODO chef: move me to other packet

// H.264-AVC-ISO_IEC_14496-15.pdf
// 5.2.4 Decoder configuration information

// <buf> body of tag
func parseAvcSeqHeader(buf []byte) (sps, pps []byte, err error) {
	// TODO chef: check if read out of <buf> range

	if buf[0] != AvcKey || buf[1] != AvcPacketTypeSeqHeader || buf[2] != 0 || buf[3] != 0 || buf[4] != 0 {
		log.Error("parse avc seq header failed.")
		err = fxxkErr
		return
	}

	//configurationVersion := buf[5]
	//avcProfileIndication := buf[6]
	//profileCompatibility := buf[7]
	//avcLevelIndication := buf[8]
	//lengthSizeMinusOne := buf[9] & 0x03

	index := 10

	numOfSps := int(buf[index] & 0x1F)
	index++
	// TODO chef: if the situation of multi sps exist?
	// only take the last one.
	for i := 0; i < numOfSps; i++ {
		lenOfSps := int(bele.BeUint16(buf[index:]))
		index += 2
		sps = append(sps, buf[index:index+lenOfSps]...)
		index += lenOfSps
	}

	numOfPps := int(buf[index] & 0x1F)
	index++
	for i := 0; i < numOfPps; i++ {
		lenOfPps := int(bele.BeUint16(buf[index:]))
		index += 2
		pps = append(pps, buf[index:index+lenOfPps]...)
		index += lenOfPps
	}

	return
}
