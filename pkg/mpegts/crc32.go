// Copyright 2023, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package mpegts

import "hash/crc32"

func CalcCrc32(crc uint32, buffer []byte) uint32 {
	table := crc32.MakeTable(crc32.IEEE)
	return crc32.Update(crc, table, buffer)
}
