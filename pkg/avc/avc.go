package avc

import (
	"errors"
	"github.com/q191201771/lal/pkg/util/bele"
	utilErrors "github.com/q191201771/lal/pkg/util/errors"
	"io"
)

var avcErr = errors.New("avc: fxxk")

var NaluStartCode = []byte{0x0, 0x0, 0x0, 0x1}

var NaluUintTypeMapping = map[uint8]string{
	1: "SLICE",
	5: "IDR",
	6: "SEI",
	7: "SPS",
	8: "PPS",
	9: "AUD",
}

// 从 rtmp avc sequence header 中解析 sps 和 pps
// @param <payload> rtmp message的payload部分 或者 flv tag的payload部分
func ParseAVCSeqHeader(payload []byte) (sps, pps []byte, err error) {
	// TODO chef: check if read out of <payload> range

	if payload[0] != 0x17 || payload[1] != 0x00 || payload[2] != 0 || payload[3] != 0 || payload[4] != 0 {
		err = avcErr
		return
	}

	// H.264-AVC-ISO_IEC_14496-15.pdf
	// 5.2.4 Decoder configuration information

	//configurationVersion := payload[5]
	//avcProfileIndication := payload[6]
	//profileCompatibility := payload[7]
	//avcLevelIndication := payload[8]
	//lengthSizeMinusOne := payload[9] & 0x03

	index := 10

	numOfSPS := int(payload[index] & 0x1F)
	index++
	// TODO chef: if the situation of multi sps exist?
	// only take the last one.
	for i := 0; i < numOfSPS; i++ {
		lenOfSPS := int(bele.BEUint16(payload[index:]))
		index += 2
		sps = append(sps, payload[index:index+lenOfSPS]...)
		index += lenOfSPS
	}

	numOfPPS := int(payload[index] & 0x1F)
	index++
	for i := 0; i < numOfPPS; i++ {
		lenOfPPS := int(bele.BEUint16(payload[index:]))
		index += 2
		pps = append(pps, payload[index:index+lenOfPPS]...)
		index += lenOfPPS
	}

	return
}

// 将rtmp avc数据转换成avc裸流
// @param <payload> rtmp message的payload部分 或者 flv tag的payload部分
func CaptureAVC(w io.Writer, payload []byte) {
	// sps pps
	if payload[0] == 0x17 && payload[1] == 0x00 {
		sps, pps, err := ParseAVCSeqHeader(payload)
		utilErrors.PanicIfErrorOccur(err)
		_, _ = w.Write(NaluStartCode)
		_, _ = w.Write(sps)
		_, _ = w.Write(NaluStartCode)
		_, _ = w.Write(pps)
		return
	}

	// payload中可能存在多个nalu
	// 先跳过前面type的2字节，以及composition time的3字节
	for i := 5; i != len(payload); {
		naluLen := int(bele.BEUint32(payload[i:]))
		i += 4
		//naluUintType := payload[i] & 0x1f
		//log.Debugf("naluLen:%d t:%d %s\n", naluLen, naluUintType, avc.NaluUintTypeMapping[naluUintType])
		_, _ = w.Write(NaluStartCode)
		_, _ = w.Write(payload[i : i+naluLen])
		i += naluLen
		break
	}
}
