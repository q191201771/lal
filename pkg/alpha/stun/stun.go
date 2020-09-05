// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package stun

import "errors"

// TODO chef:
// - attr soft

// Session Traversal Utilities for NAT
//
// rfc 5389
//

var ErrStun = errors.New("lal.stun: fxxk")

var DefaultPort = 3478
