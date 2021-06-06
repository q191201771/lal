// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package hls_test

import (
	"testing"

	"github.com/q191201771/lal/pkg/hls"
	"github.com/q191201771/naza/pkg/assert"
)

func TestCalcM3u8Duration(t *testing.T) {
	golden := []byte(`
#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:5
#EXT-X-MEDIA-SEQUENCE:0

#EXT-X-DISCONTINUITY
#EXTINF:4.000,
1607342284-0.ts
#EXTINF:4.000,
1607342288-1.ts
#EXTINF:3.333,
1607342292-2.ts
#EXTINF:4.000,
1607342295-3.ts
#EXTINF:4.867,
1607342299-4.ts
#EXTINF:3.133,
1607342304-5.ts
#EXTINF:4.000,
1607342307-6.ts
#EXTINF:3.800,
1607342311-7.ts
#EXTINF:4.800,
1607342315-8.ts
#EXTINF:3.267,
1607342320-9.ts
#EXT-X-ENDLIST
`)
	duration, err := hls.CalcM3u8Duration(golden)
	assert.Equal(t, nil, err)
	assert.Equal(t, float64(39.2), duration)
}
