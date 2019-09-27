package logic

import (
	"github.com/q191201771/lal/pkg/httpflv"
	"github.com/q191201771/lal/pkg/rtmp"
)

var Trans trans

type trans struct {
}

//// TODO chef: rtmp msg 也弄成结构体
func (t trans) FlvTag2RTMPMsg(tag httpflv.Tag) (header rtmp.Header, timestampAbs uint32, message []byte) {
	header.MsgLen = tag.Header.DataSize
	header.MsgTypeID = tag.Header.T
	header.MsgStreamID = rtmp.MSID1 // TODO
	switch tag.Header.T {
	case httpflv.TagTypeMetadata:
		header.CSID = rtmp.CSIDAMF
	case httpflv.TagTypeAudio:
		header.CSID = rtmp.CSIDAudio
	case httpflv.TagTypeVideo:
		header.CSID = rtmp.CSIDVideo
	}
	header.Timestamp = tag.Header.Timestamp
	timestampAbs = tag.Header.Timestamp
	message = tag.Raw[11 : 11+header.MsgLen]
	return
}

func (t trans) RTMPMsg2FlvTag(header rtmp.Header, timestampAbs uint32, message []byte) httpflv.Tag {
	var tag httpflv.Tag
	tag.Header.T = header.MsgTypeID
	tag.Header.DataSize = header.MsgLen
	tag.Header.Timestamp = timestampAbs
	tag.Raw = httpflv.PackHTTPFlvTag(header.MsgTypeID, timestampAbs, message)
	return tag
}
