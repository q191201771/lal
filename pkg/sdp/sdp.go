// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
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

var ErrSDP = errors.New("lal.sdp: fxxk")

const (
	ARTPMapEncodingNameH265 = "H265"
	ARTPMapEncodingNameH264 = "H264"
	ARTPMapEncodingNameAAC  = "MPEG4-GENERIC"
)
