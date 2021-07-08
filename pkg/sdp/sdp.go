// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package sdp

import (
	"errors"
)

// rfc4566

var ErrSdp = errors.New("lal.sdp: fxxk")

const (
	ARtpMapEncodingNameH265 = "H265"
	ARtpMapEncodingNameH264 = "H264"
	ARtpMapEncodingNameAac  = "MPEG4-GENERIC"
)
