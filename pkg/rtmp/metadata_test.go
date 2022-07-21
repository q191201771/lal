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
	"testing"

	"github.com/q191201771/lal/pkg/base"

	"github.com/q191201771/lal/pkg/rtmp"
	"github.com/q191201771/naza/pkg/assert"
)

func TestMetadata(t *testing.T) {
	cache := base.LalRtmpBuildMetadataEncoder
	base.LalRtmpBuildMetadataEncoder = "lal0.30.1"
	defer func() {
		base.LalRtmpBuildMetadataEncoder = cache
	}()

	// -----
	b, err := rtmp.BuildMetadata(1024, 768, 10, 7)
	assert.Equal(t, nil, err)
	assert.Equal(t, "02000a6f6e4d6574614461746103000577696474680040900000000000000006686569676874004088000000000000000c617564696f636f6465636964004024000000000000000c766964656f636f646563696400401c000000000000000776657273696f6e0200096c616c302e33302e31000009", hex.EncodeToString(b))

	// -----
	opa, err := rtmp.ParseMetadata(b)
	assert.Equal(t, nil, err)
	assert.Equal(t, 5, len(opa))
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

	// -----
	wo, err := rtmp.MetadataEnsureWithoutSdf(b)
	assert.Equal(t, nil, err)
	assert.Equal(t, b, wo)
	w, err := rtmp.MetadataEnsureWithSdf(b)
	assert.Equal(t, nil, err)
	assert.Equal(t, "02000d40736574446174614672616d6502000a6f6e4d6574614461746103000577696474680040900000000000000006686569676874004088000000000000000c617564696f636f6465636964004024000000000000000c766964656f636f646563696400401c000000000000000776657273696f6e0200096c616c302e33302e31000009", hex.EncodeToString(w))
	wo, err = rtmp.MetadataEnsureWithoutSdf(b)
	assert.Equal(t, nil, err)
	assert.Equal(t, b, wo)
}
