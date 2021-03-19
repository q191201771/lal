// Copyright 2019, Chef.  All rights reserved.
// https://github.com/cfeeling/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package httpflv

var readBufSize = 256 //16384 // ClientPullSession 和 SubSession 读取数据时
var wChanSize = 1024  // SubSession 发送数据时 channel 的大小
var subSessionWriteTimeoutMS = 10000

var FLVHeader = []byte{0x46, 0x4c, 0x56, 0x01, 0x05, 0x00, 0x00, 0x00, 0x09, 0x00, 0x00, 0x00, 0x00}
