package avc

// H.264-AVC-ISO_IEC_14496-15.pdf
// 5.2.4 Decoder configuration information

// <buf> body of tag
//func parseAVCSeqHeader(buf []byte) (sps, pps []byte, err error) {
//	// TODO chef: check if read out of <buf> range
//
//	if buf[0] != AVCKey || buf[1] != isAVCKeySeqHeader || buf[2] != 0 || buf[3] != 0 || buf[4] != 0 {
//		log.Error("parse avc seq header failed.")
//		err = httpFlvErr
//		return
//	}
//
//	//configurationVersion := buf[5]
//	//avcProfileIndication := buf[6]
//	//profileCompatibility := buf[7]
//	//avcLevelIndication := buf[8]
//	//lengthSizeMinusOne := buf[9] & 0x03
//
//	index := 10
//
//	numOfSPS := int(buf[index] & 0x1F)
//	index++
//	// TODO chef: if the situation of multi sps exist?
//	// only take the last one.
//	for i := 0; i < numOfSPS; i++ {
//		lenOfSPS := int(bele.BEUint16(buf[index:]))
//		index += 2
//		sps = append(sps, buf[index:index+lenOfSPS]...)
//		index += lenOfSPS
//	}
//
//	numOfPPS := int(buf[index] & 0x1F)
//	index++
//	for i := 0; i < numOfPPS; i++ {
//		lenOfPPS := int(bele.BEUint16(buf[index:]))
//		index += 2
//		pps = append(pps, buf[index:index+lenOfPPS]...)
//		index += lenOfPPS
//	}
//
//	return
//}
