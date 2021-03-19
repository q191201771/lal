// Copyright 2020, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpts

var readBufSize = 256 //16384 // SubSession读取数据时
var wChanSize = 1024  // SubSession发送数据时channel的大小
var subSessionWriteTimeoutMS = 10000
