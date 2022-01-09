// Copyright 2022, Chef.  All rights reserved.
// https://github.com/q191201771/lal
//
// Use of this source code is governed by a MIT-style license
// that can be found in the License file.
//
// Author: Chef (191201771@qq.com)

package logic

import (
	"testing"

	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/naza/pkg/assert"
)

func TestSimpleAuthCalcSecret(t *testing.T) {
	res := SimpleAuthCalcSecret("q191201771", "test110")
	assert.Equal(t, "700997e1595a06c9ffa60ebef79105b0", res)
}

func TestSimpleAuthCtx(t *testing.T) {
	ctx := NewSimpleAuthCtx(SimpleAuthConfig{
		Key:           "q191201771",
		PubRtmpEnable: true,
	})
	var info base.PubStartInfo
	info.Protocol = base.ProtocolRtmp
	info.StreamName = "test110"
	info.UrlParam = "lal_secret=700997e1595a06c9ffa60ebef79105b0"
	res := ctx.OnPubStart(info)
	assert.Equal(t, nil, res)
}
