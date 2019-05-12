package rtmp

import (
	"errors"
)

var rtmpErr = errors.New("rtmp error")

var csidProtocolControl = 2
var csidOverConnection = 3
var csidOverStream = 5

var typeidSetChunkSize = 1
var typeidUserControl = 4
var typeidWinAckSize = 5
var typeidBandwidth = 6
var typeidAudio = 8
var typeidVideo = 9
var typeidDataMessageAMF0 = 18 // meta
var typeidCommandMessageAMF0 = 20

var tidClientConnect = 1
var tidClientCreateStream = 2
var tidClientPlay = 3
var tidClientPublish = 3

var maxTimestampInMessageHeader = 0xFFFFFF

var defaultChunkSize = 128
