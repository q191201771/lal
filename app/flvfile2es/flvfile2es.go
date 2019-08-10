package main

import (
	"encoding/hex"
	"flag"
	"github.com/q191201771/lal/pkg/avc"
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/util/bele"
	"github.com/q191201771/lal/pkg/util/errors"
	"github.com/q191201771/lal/pkg/util/log"
	"io"
	"os"
)

var adtsHeader = make([]byte, 7)

func captureAVC(w io.Writer, payload []byte) {
	// sps pps
	if payload[0] == 0x17 && payload[1] == 0x00 {
		sps, pps, err := avc.ParseAVCSeqHeader(payload)
		errors.PanicIfErrorOccur(err)
		_, _ = w.Write(avc.NaluStartCode)
		_, _ = w.Write(sps)
		_, _ = w.Write(avc.NaluStartCode)
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
		_, _ = w.Write(avc.NaluStartCode)
		_, _ = w.Write(payload[i:i+naluLen])
		i += naluLen
		break
	}
	return
}

// TODO chef: mv to pkg
func captureAAC(w io.Writer, payload []byte) {
	soundFormat := payload[0] >> 4
	soundRate := (payload[0] >> 2) & 0x03
	soundSize := (payload[0] >> 1) & 0x01
	soundType := payload[0] & 0x01

	if payload[1] == 0 {
		audioObjectType := (payload[2] >> 3) & 0x1f // 5bit 编码结构类型
		samplingFrequencyIndex := ((payload[2] & 0x07) << 1) | (payload[3] >> 7) // 4bit 音频采样率索引值
		channelConfig := (payload[3] >> 3) & 0x0f // 4bit 音频输出声道

		// syncword                 12bit
		// ID                        1bit
		// layer                     2
		// protection_absent         1bit
		// profile                   2bit
		// sampling_frequency_index  4bit
		// private_bit               1bit
		// channel_configuration     3bit
		// origin_copy               1bit
		// home                      1bit
		adtsHeader[0] = 0xff // 8bit syncword 高8bit
		adtsHeader[1] = 0xf0 // 4bit syncword 低4bit
		// 1bit ID 0 for MPEG-4, 1 for MPEG-2
		// 2bit layer 0
		adtsHeader[1] |= 1 // 1bit protection absent
		adtsHeader[2] = (audioObjectType - 1) << 6

		log.Debugf(hex.Dump(payload[:4]))
		log.Debugf("%d %d %d %d\n", soundFormat, soundRate, soundSize, soundType)
		log.Debugf("%d %d %d", audioObjectType, samplingFrequencyIndex, channelConfig)
	}
}

func main() {
	var err error
	flvFileName, aacFileName, avcFileName := parseFlag()

	var ffr httpflv.FlvFileReader
	err = ffr.Open(flvFileName)
	errors.PanicIfErrorOccur(err)
	defer ffr.Dispose()
	log.Infof("open flv file succ.")

	afp, err := os.Create(aacFileName)
	errors.PanicIfErrorOccur(err)
	defer afp.Close()
	log.Infof("open es aac file succ.")

	vfp, err := os.Create(avcFileName)
	errors.PanicIfErrorOccur(err)
	defer vfp.Close()
	log.Infof("open es h264 file succ.")

	_, err = ffr.ReadFlvHeader()
	errors.PanicIfErrorOccur(err)

	for {
		tag, err := ffr.ReadTag()
		if err == io.EOF {
			log.Infof("EOF.")
			break
		}
		errors.PanicIfErrorOccur(err)

		payload := tag.Payload()

		switch tag.Header.T {
		case httpflv.TagTypeAudio:
			captureAAC(afp, payload)
		case httpflv.TagTypeVideo:
			captureAVC(vfp, payload)
		}
	}
}

func parseFlag() (string, string, string) {
	flv := flag.String("i", "", "specify flv file")
	aac := flag.String("a", "", "specify es aac file")
	avc := flag.String("v", "", "specify es h264 file")
	flag.Parse()
	if *flv == "" || *avc == "" || *aac == "" {
		flag.Usage()
		os.Exit(1)
	}
	return *flv, *aac, *avc
}