// Copyright 2020, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package rtmp_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/assert"
)

func TestMetadata(t *testing.T) {
	// -----
	b, err := rtmp.BuildMetadata(1024, 768, 10, 7)
	assert.Equal(t, nil, err)
	ver := hex.EncodeToString([]byte(base.LalVersionDot))
	expected := "02000a6f6e4d6574614461746103000577696474680040900000000000000006686569676874004088000000000000000c617564696f636f6465636964004024000000000000000c766964656f636f646563696400401c000000000000000776657273696f6e0200096c616c" +
		ver + "00036c616c020006" + ver + "000009"
	assert.Equal(t, expected, hex.EncodeToString(b))

	// -----
	opa, err := rtmp.ParseMetadata(b)
	assert.Equal(t, nil, err)
	assert.Equal(t, 6, len(opa))
	v := opa.Find("width")
	assert.Equal(t, float64(1024), v.(float64))
	v = opa.Find("height")
	assert.Equal(t, float64(768), v.(float64))
	v = opa.Find("audiocodecid")
	assert.Equal(t, float64(10), v.(float64))
	v = opa.Find("videocodecid")
	assert.Equal(t, float64(7), v.(float64))
	v = opa.Find("version")
	assert.Equal(t, base.LalRtmpBuildMetadataEncoder, v.(string))
	v = opa.Find("lal")
	assert.Equal(t, base.LalVersionDot, v.(string))

	// -----
	wo, err := rtmp.MetadataEnsureWithoutSdf(b)
	assert.Equal(t, nil, err)
	assert.Equal(t, b, wo)

	w, err := rtmp.MetadataEnsureWithSdf(b)
	assert.Equal(t, nil, err)
	exp2 := "02000d40736574446174614672616d6502000a6f6e4d6574614461746103000577696474680040900000000000000006686569676874004088000000000000000c617564696f636f6465636964004024000000000000000c766964656f636f646563696400401c000000000000000776657273696f6e0200096c616c302e33322e3000036c616c020006302e33322e30000009"
	// 注意，替换版本为当前版本
	exp2 = strings.Replace(exp2, "302e33322e30", ver, -1)
	assert.Equal(t, exp2, hex.EncodeToString(w))

	wo, err = rtmp.MetadataEnsureWithoutSdf(b)
	assert.Equal(t, nil, err)
	assert.Equal(t, b, wo)
}
