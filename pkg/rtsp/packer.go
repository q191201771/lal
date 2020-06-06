// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtsp

import (
	"fmt"
	"time"
)

var ResponseOptionsTmpl = "RTSP/1.0 200 OK\r\nCSeq: %s\r\nPublic:DESCRIBE, SETUP, TEARDOWN, PLAY, PAUSE\r\n\r\n"
var ResponseSetupTmpl = "RTSP/1.0 200 OK\r\nCSeq: %s\r\nDate: %s\r\nSession: 47112344\r\nTransport:%s;server_port=6256-6257"

func PackResponseOptions(cseq string) string {
	return fmt.Sprintf(ResponseOptionsTmpl, cseq)
}

func PackResponseSetup(cseq string, transportC string) string {
	date := time.Now().Format(time.RFC1123)
	return fmt.Sprintf(ResponseSetupTmpl, cseq, date, transportC)
}
