// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls

import "github.com/q191201771/naza/pkg/nazalog"

func SplitTS(content []byte) (ret [][]byte) {
	for {
		if len(content) < 188 {
			nazalog.Assert(0, len(content))
			break
		}

		ret = append(ret, content[0:188])
		content = content[188:]
	}
	return
}
