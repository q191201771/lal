package rtmp

import (
	"errors"
)

var rtmpErr = errors.New("rtmp error")

var csidProtocolControl = 2
var csidOverConnection = 3

var typeidSetChunkSize = 1
var typeidWinAckSize = 5
var typeidBandwidth = 6
var typeidCommandMessageAMF0 = 20

var tidClientConnect = 1

var maxTimestampInMessageHeader = 0xFFFFFF

var defaultChunkSize = 128
